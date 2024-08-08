package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/MoonshotAI/moonpalace/detector/repeat"
	"github.com/MoonshotAI/moonpalace/merge"
	"github.com/spf13/cobra"
)

func startCommand() *cobra.Command {
	var (
		port            int16
		key             string
		detectRepeat    bool
		repeatThreshold float64
		repeatMinLength int32
		forceStream     bool
	)
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the MoonPalace proxy server",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, stop := signal.NotifyContext(context.Background(),
				syscall.SIGINT,
				syscall.SIGTERM)
			defer stop()
			httpServer.Handler = http.HandlerFunc(buildProxy(
				key,
				detectRepeat,
				repeatThreshold,
				repeatMinLength,
				forceStream,
			))
			httpServer.Addr = "127.0.0.1:" + strconv.Itoa(int(port))
			go func() {
				if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					logFatal(err)
				}
			}()
			logServerStarts("http://" + httpServer.Addr + "/v1")
			<-ctx.Done()
			stop()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := httpServer.Shutdown(shutdownCtx); err != nil {
				logFatal(err)
			}
		},
	}
	flags := cmd.PersistentFlags()
	flags.Int16VarP(&port, "port", "p", 9988, "port to listen on")
	flags.StringVarP(&key, "key", "k", "", "API key by default")
	flags.BoolVar(&detectRepeat, "detect-repeat", false, "detect and prevent repeating tokens in streaming output")
	flags.Float64Var(&repeatThreshold, "repeat-threshold", 0.5, "repeat threshold, a float between [0, 1]")
	flags.Int32Var(&repeatMinLength, "repeat-min-length", 100, "repeat min length, minimum string length to detect repeat")
	flags.BoolVar(&forceStream, "force-stream", false, "force streaming for all chat completions requests")
	return cmd
}

var (
	httpServer = &http.Server{
		ReadHeaderTimeout: 1 * time.Minute,
		WriteTimeout:      5 * time.Minute,
		ErrorLog:          serverErrorLogger,
	}
	httpClient = &http.Client{
		Timeout: time.Minute * 5,
	}

	loggingMutex  sync.Mutex
	detectorsPool = &sync.Pool{
		New: func() any {
			return make(map[int]*RepeatDetector)
		},
	}
	completionPool = &sync.Pool{
		New: func() any {
			return make(map[string]any)
		},
	}
	merger = &merge.Merger{
		StreamFields: []string{"content", "arguments"},
		IndexFields:  []string{"index"},
	}
)

func putDetectors(detectors map[int]*RepeatDetector) {
	for index, detector := range detectors {
		detector.Automaton.Clear()
		delete(detectors, index)
	}
	detectorsPool.Put(detectors)
}

func putCompletion(completion map[string]any) {
	for objectKey := range completion {
		delete(completion, objectKey)
	}
	completionPool.Put(completion)
}

func mergeIn(completion map[string]any, value []byte) {
	chunk := completionPool.Get().(map[string]any)
	defer putCompletion(chunk)
	decoder := json.NewDecoder(bytes.NewReader(value))
	decoder.UseNumber()
	if err := decoder.Decode(&chunk); err == nil {
		merger.MergeObject(completion, chunk)
	}
}

func buildProxy(
	key string,
	detectRepeat bool,
	repeatThreshold float64,
	repeatMinLength int32,
	forceStream bool,
) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			err                       error
			encoder                   = json.NewEncoder(w)
			newRequest                *http.Request
			newResponse               *http.Response
			requestAcceptEncodingGzip bool
			requestUseStream          bool
			requestBody               []byte
			responseBody              []byte
			requestID                 = r.Header.Get("X-Request-Id")
			requestContentType        = filterHeaderFlags(r.Header.Get("Content-Type"))
			requestMethod             = r.Method
			requestPath               = r.URL.Path
			requestQuery              = r.URL.RawQuery
			moonshot                  *Moonshot
			moonshotID                string
			moonshotGID               string
			moonshotUID               string
			moonshotRequestID         string
			moonshotServerTiming      int
			moonshotContextCacheID    string
			responseStatus            string
			responseStatusCode        int
			responseContentType       string
			responseTTFT              int
			createdAt                 = time.Now()
		)
		defer func() {
			go func() {
				loggingMutex.Lock()
				defer loggingMutex.Unlock()
				logRequest(
					requestMethod,
					requestPath,
					requestQuery,
					requestContentType,
					requestID,
					responseStatus,
					responseContentType,
					responseTTFT,
					moonshotRequestID,
					moonshotServerTiming,
					moonshotContextCacheID,
					moonshotUID,
					moonshotGID,
					moonshot,
					err,
				)
				var lastInsertID int64
				lastInsertID, err = persistence.Persistence(
					requestID,
					requestContentType,
					requestMethod,
					requestPath,
					requestQuery,
					moonshotID,
					moonshotGID,
					moonshotUID,
					moonshotRequestID,
					moonshotServerTiming,
					responseStatusCode,
					responseContentType,
					formatHeader(newRequest),
					string(requestBody),
					formatHeader(newResponse),
					string(responseBody),
					toErrMsg(err),
					responseTTFT,
					createdAt.Format(time.DateTime),
				)
				if err != nil {
					logFatal(err)
				}
				logNewRow(lastInsertID)
			}()
		}()
		requestBody, err = io.ReadAll(r.Body)
		if err != nil {
			writeProxyError(encoder, "read_request_body", err)
			return
		}
		if forceStream {
			var streamRequest MoonshotStreamRequest
			json.Unmarshal(requestBody, &streamRequest)
			requestUseStream = streamRequest.Stream
			if !requestUseStream {
				requestBody = forceUseStream(requestBody)
			}
		}
		newRequest, err = http.NewRequestWithContext(
			r.Context(),
			r.Method,
			endpoint+requestPath,
			bytes.NewReader(requestBody),
		)
		if err != nil {
			writeProxyError(encoder, "make_new_request", err)
			return
		}
		if encodings := r.Header.Values("Accept-Encoding"); encodings != nil {
		INSPECT:
			for _, encoding := range encodings {
				accepts := strings.Split(encoding, ",")
				for _, accept := range accepts {
					if accept = strings.TrimSpace(accept); accept == "gzip" {
						requestAcceptEncodingGzip = true
						break INSPECT
					}
				}
			}
		}
		for header, values := range r.Header {
			for _, value := range values {
				newRequest.Header.Add(header, value)
			}
		}
		if key != "" {
			newRequest.Header.Set("Authorization", "Bearer "+key)
		}
		if requestAcceptEncodingGzip {
			newRequest.Header.Set("Accept-Encoding", "gzip")
		} else {
			newRequest.Header.Del("Accept-Encoding")
		}
		createdAt = time.Now()
		newResponse, err = httpClient.Do(newRequest)
		if err != nil {
			writeProxyError(encoder, "send_new_request", err)
			return
		}
		defer newResponse.Body.Close()
		for header, values := range newResponse.Header {
			for _, value := range values {
				w.Header().Add(header, value)
			}
		}
		w.WriteHeader(newResponse.StatusCode)
		if contentType := filterHeaderFlags(newResponse.Header.Get("Content-Type")); contentType == "text/event-stream" {
			var detectors map[int]*RepeatDetector
			if detectRepeat {
				detectors = detectorsPool.Get().(map[int]*RepeatDetector)
				defer putDetectors(detectors)
			}
			var completion map[string]any
			if forceStream && !requestUseStream {
				completion = completionPool.Get().(map[string]any)
				defer putCompletion(completion)
			}
			scanner := bufio.NewScanner(newResponse.Body)
		READLINES:
			for scanner.Scan() {
				line := scanner.Bytes()
				if !(forceStream && !requestUseStream) {
					w.Write(line)
					w.Write([]byte("\n\n"))
				}
				responseBody = append(responseBody, line...)
				responseBody = append(responseBody, '\n')
				if field, value, ok := bytes.Cut(line, []byte{':'}); ok {
					field, value = bytes.TrimSpace(field), bytes.TrimSpace(value)
					if bytes.Equal(field, []byte("data")) && !bytes.Equal(value, []byte("[DONE]")) {
						if forceStream && !requestUseStream {
							mergeIn(completion, value)
						}
						var chunk MoonshotChunk
						if err = json.Unmarshal(value, &chunk); err == nil && chunk.ID != "" {
							if moonshot == nil {
								moonshot = new(Moonshot)
							}
							moonshot.ID = chunk.ID
							moonshotID = moonshot.ID
							if chunk.Choices != nil && len(chunk.Choices) > 0 {
								for _, choice := range chunk.Choices {
									if choice.Delta.Content != "" && responseTTFT == 0 {
										responseTTFT = int(time.Since(createdAt) / time.Millisecond)
									}
									if choice.Usage != nil {
										if moonshot.Usage == nil {
											moonshot.Usage = &MoonshotUsage{
												PromptTokens:     choice.Usage.PromptTokens,
												CompletionTokens: choice.Usage.CompletionTokens,
												TotalTokens:      choice.Usage.TotalTokens,
												CachedTokens:     choice.Usage.CachedTokens,
											}
										} else {
											moonshot.Usage.CompletionTokens += choice.Usage.CompletionTokens
											moonshot.Usage.TotalTokens += choice.Usage.CompletionTokens
										}
									}
									if choice.FinishReason != nil && *choice.FinishReason == "length" {
										err = errors.New("it seems that your max_tokens value is too small, please set a larger value")
									}
									if detectRepeat {
										var detector *RepeatDetector
										if _, exists := detectors[choice.Index]; exists {
											detector = detectors[choice.Index]
										} else {
											detector = &RepeatDetector{Automaton: repeat.NewSuffixAutomaton()}
											detectors[choice.Index] = detector
										}
										if choice.FinishReason != nil {
											detector.FinishReason = *choice.FinishReason
										}
										detector.Automaton.AddString(choice.Delta.Content)
										if detector.Automaton.Length() > repeatMinLength && detector.Automaton.GetRepeatness() < repeatThreshold {
											err = errors.New("it appears that there is an issue with content repeating in the current response")
											for index, snapshot := range detectors {
												if snapshot.FinishReason == "" {
													finishChunk := []byte(fmt.Sprintf(
														"{\"choices\":[{\"delta\":{},\"finish_reason\":\"repeat\",\"index\":%d}],"+
															"\"created\":%d,\"id\":\"%s\",\"model\":\"%s\",\"object\":\"%s\"}",
														index,
														chunk.Created,
														chunk.ID,
														chunk.Model,
														chunk.Object,
													))
													if forceStream && !requestUseStream {
														mergeIn(completion, finishChunk)
													} else {
														w.Write([]byte("data: "))
														w.Write(finishChunk)
														w.Write([]byte("\n\n"))
													}
												}
											}
											if !(forceStream && !requestUseStream) {
												w.Write([]byte("[DONE]"))
											}
											break READLINES
										}
									}
								}
							}
						}
					}
				}
			}
			if forceStream && !requestUseStream {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.Header().Del("Content-Encoding")
				if choicesValue, exists := completion["choices"]; exists {
					if choices, isArr := choicesValue.([]any); isArr {
						for _, choiceValue := range choices {
							if choice, isObj := choiceValue.(map[string]any); isObj {
								if delta, exists := choice["delta"]; exists {
									choice["message"] = delta
									delete(choice, "delta")
								}
							}
						}
					}
				}
				json.NewEncoder(w).Encode(completion)
			}
		} else {
			responseBody, err = io.ReadAll(newResponse.Body)
			if err != nil {
				writeProxyError(encoder, "read_response_body", err)
				return
			}
			w.Write(responseBody)
			if isGzip(newResponse.Header) {
				var gzipReader *gzip.Reader
				gzipReader, err = gzip.NewReader(bytes.NewReader(responseBody))
				if err != nil {
					return
				}
				responseBody, err = io.ReadAll(gzipReader)
				if err != nil {
					return
				}
			}
			var completion MoonshotCompletion
			if err = json.Unmarshal(responseBody, &completion); err == nil && completion.ID != "" {
				if moonshot == nil {
					moonshot = new(Moonshot)
				}
				moonshot.ID = completion.ID
				moonshotID = moonshot.ID
				if completion.Usage != nil {
					moonshot.Usage = &MoonshotUsage{
						PromptTokens:     completion.Usage.PromptTokens,
						CompletionTokens: completion.Usage.CompletionTokens,
						TotalTokens:      completion.Usage.TotalTokens,
						CachedTokens:     completion.Usage.CachedTokens,
					}
				}
				if completion.Choices != nil && len(completion.Choices) > 0 {
					for _, choice := range completion.Choices {
						if choice.FinishReason != nil && *choice.FinishReason == "length" {
							err = fmt.Errorf("it seems that your max_tokens value is too small, please set a value greater than %d",
								completion.Usage.CompletionTokens)
						}
					}
				}
			}
		}
		moonshotGID = newResponse.Header.Get("Msh-Gid")
		moonshotUID = newResponse.Header.Get("Msh-Uid")
		moonshotRequestID = newResponse.Header.Get("Msh-Request-Id")
		if serverTiming := newResponse.Header.Get("Server-Timing"); serverTiming != "" {
			parts := strings.Split(serverTiming, ";")
			for _, part := range parts {
				if part = strings.TrimSpace(part); strings.HasPrefix(part, "dur=") {
					timing := strings.TrimPrefix(part, "dur=")
					moonshotServerTiming, _ = strconv.Atoi(timing)
					break
				}
			}
		}
		moonshotContextCacheID = newResponse.Header.Get("Msh-Context-Cache-Id")
		responseStatus = newResponse.Status
		responseStatusCode = newResponse.StatusCode
		responseContentType = filterHeaderFlags(newResponse.Header.Get("Content-Type"))
		if responseStatusCode != http.StatusOK {
			err = &moonshotError{message: string(responseBody)}
		}
	}
}

type Moonshot struct {
	ID    string         `json:"id"`
	Usage *MoonshotUsage `json:"usage"`
}

type MoonshotStreamRequest struct {
	Stream bool `json:"stream"`
}

type MoonshotChunk = MoonshotCompletion

type MoonshotCompletion struct {
	ID      string            `json:"id"`
	Created int64             `json:"created"`
	Model   string            `json:"model"`
	Object  string            `json:"object"`
	Choices []*MoonshotChoice `json:"choices"`
	Usage   *MoonshotUsage    `json:"usage"`
}

type MoonshotChoice struct {
	Index        int              `json:"index"`
	Delta        *MoonshotMessage `json:"delta"`
	Message      *MoonshotMessage `json:"message"`
	FinishReason *string          `json:"finish_reason"`
	Usage        *MoonshotUsage   `json:"usage"`
}

type MoonshotMessage struct {
	Content string `json:"content"`
}

type MoonshotUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
	CachedTokens     int `json:"cached_tokens"`
}

func isGzip(header http.Header) bool {
	if encodings := header.Values("Content-Encoding"); encodings != nil {
		for _, encoding := range encodings {
			if filterHeaderFlags(encoding) == "gzip" {
				return true
			}
		}
	}
	return false
}

func filterHeaderFlags(content string) string {
	for i, char := range content {
		if char == ' ' || char == ';' {
			return content[:i]
		}
	}
	return content
}

func formatHeader[R *http.Request | *http.Response](r R) string {
	if r == nil {
		return ""
	}
	var header http.Header
	switch any(r).(type) {
	case *http.Request:
		header = any(r).(*http.Request).Header
	case *http.Response:
		header = any(r).(*http.Response).Header
	}
	if header == nil {
		return ""
	}
	header.Del("Authorization")
	var headerBuilder strings.Builder
	header.Write(&headerBuilder)
	return headerBuilder.String()
}

type object map[string]any

func writeProxyError(encoder *json.Encoder, typ string, err error) {
	encoder.Encode(object{
		"error": object{
			"code":    "proxy_server_error",
			"type":    typ,
			"message": err.Error(),
		},
	})
}

type moonshotError struct {
	message string
}

func (m *moonshotError) Error() string {
	return m.message
}

func toErrMsg(err error) string {
	if err == nil {
		return ""
	}
	if errors.As(err, new(*moonshotError)) {
		return ""
	}
	return err.Error()
}

var asciiSpace = [256]uint8{'\t': 1, '\n': 1, '\v': 1, '\f': 1, '\r': 1, ' ': 1}

const streamOptions = `"stream":true,"stream_options":{"include_usage":true},`

func forceUseStream(data []byte) []byte {
	if !json.Valid(data) {
		return data
	}
	newData := make([]byte, 0, len(data)+len(streamOptions))
	insertIndex := 0
	for i, b := range data {
		if asciiSpace[b] == 1 {
			continue
		}
		if b == '{' {
			insertIndex = i + 1
			break
		}
	}
	newData = append(newData, '{')
	newData = append(newData, streamOptions...)
	newData = append(newData, data[insertIndex:]...)
	return newData
}

type RepeatDetector struct {
	Automaton    *repeat.SuffixAutomaton
	FinishReason string
}
