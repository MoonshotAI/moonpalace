package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/MoonshotAI/moonpalace/detector/repeat"
	"github.com/spf13/cobra"
)

var (
	httpProxyKey             string
	httpProxyDetectRepeat    bool
	httpProxyRepeatThreshold float64
	httpProxyRepeatMinLength int32
)

func startCommand() *cobra.Command {
	var (
		port int16
	)
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the MoonPalace proxy server.",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, stop := signal.NotifyContext(context.Background(),
				syscall.SIGINT,
				syscall.SIGTERM)
			defer stop()
			if err := persistence.createTable(ctx); err != nil {
				logFatal(err)
			}
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
	flags.StringVarP(&httpProxyKey, "key", "k", "", "API key by default")
	flags.BoolVar(&httpProxyDetectRepeat, "detect-repeat", false, "detect and prevent repeating tokens in streaming output")
	flags.Float64Var(&httpProxyRepeatThreshold, "repeat-threshold", 0.5, "repeat threshold, a float between [0, 1]")
	flags.Int32Var(&httpProxyRepeatMinLength, "repeat-min-length", 100, "repeat min length, minimum string length to detect repeat")
	return cmd
}

var (
	httpServer = &http.Server{
		Handler:      http.HandlerFunc(proxy),
		ReadTimeout:  5 * time.Minute,
		WriteTimeout: 5 * time.Minute,
		ErrorLog:     serverErrorLogger,
	}
	httpClient = &http.Client{
		Timeout: time.Minute * 5,
	}

	detectorsPool = &sync.Pool{
		New: func() any {
			return make(map[int]*repeat.SuffixAutomaton)
		},
	}
)

func proxy(w http.ResponseWriter, r *http.Request) {
	var (
		err                       error
		encoder                   = json.NewEncoder(w)
		newRequest                *http.Request
		newResponse               *http.Response
		requestAcceptEncodingGzip bool
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
		responseStatus            string
		responseStatusCode        int
		responseContentType       string
	)
	defer func() {
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
		)
		if err != nil {
			logFatal(err)
		}
		logNewRow(lastInsertID)
	}()
	defer func() {
		logRequest(
			requestMethod,
			requestPath,
			requestQuery,
			requestContentType,
			requestID,
			responseStatus,
			responseContentType,
			moonshotRequestID,
			moonshotServerTiming,
			moonshotUID,
			moonshotGID,
			moonshot,
			err,
		)
	}()
	requestBody, err = io.ReadAll(r.Body)
	if err != nil {
		writeProxyError(encoder, "read_request_body", err)
		return
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
	if httpProxyKey != "" {
		newRequest.Header.Set("Authorization", "Bearer "+httpProxyKey)
	}
	if requestAcceptEncodingGzip {
		newRequest.Header.Set("Accept-Encoding", "gzip")
	} else {
		newRequest.Header.Del("Accept-Encoding")
	}
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
		detectors := detectorsPool.Get().(map[int]*repeat.SuffixAutomaton)
		defer func() {
			for index, detector := range detectors {
				detector.Clear()
				delete(detectors, index)
			}
			detectorsPool.Put(detectorsPool)
		}()
		scanner := bufio.NewScanner(newResponse.Body)
		for scanner.Scan() {
			line := scanner.Bytes()
			responseBody = append(responseBody, line...)
			responseBody = append(responseBody, '\n')
			if field, value, ok := bytes.Cut(line, []byte{':'}); ok {
				field, value = bytes.TrimSpace(field), bytes.TrimSpace(value)
				if bytes.Equal(field, []byte("data")) && !bytes.Equal(value, []byte("[DONE]")) {
					var chunk MoonshotChunk
					if err = json.Unmarshal(value, &chunk); err == nil && chunk.ID != "" {
						if moonshot == nil {
							moonshot = new(Moonshot)
						}
						moonshot.ID = chunk.ID
						moonshotID = moonshot.ID
						if chunk.Choices != nil && len(chunk.Choices) > 0 {
							for _, choice := range chunk.Choices {
								if choice.Usage != nil {
									if moonshot.Usage == nil {
										moonshot.Usage = &MoonshotUsage{
											PromptTokens:     choice.Usage.PromptTokens,
											CompletionTokens: choice.Usage.CompletionTokens,
											TotalTokens:      choice.Usage.TotalTokens,
										}
									} else {
										moonshot.Usage.PromptTokens += choice.Usage.PromptTokens
										moonshot.Usage.CompletionTokens += choice.Usage.CompletionTokens
										moonshot.Usage.TotalTokens += choice.Usage.TotalTokens
									}
								}
								if httpProxyDetectRepeat {
									var detector *repeat.SuffixAutomaton
									if _, exists := detectors[choice.Index]; exists {
										detector = detectors[choice.Index]
									} else {
										detector = repeat.NewSuffixAutomaton()
										detectors[choice.Index] = detector
									}
									detector.AddString(choice.Delta.Content)
									if detector.Length() > httpProxyRepeatMinLength && detector.GetRepeatness() < httpProxyRepeatThreshold {
										err = errors.New("it appears that there is an issue with content repeating in the current response")
										w.Write([]byte("[DONE]"))
										goto FINISHING
									}
								}
							}
						}
					}
				}
			}
			w.Write(line)
			w.Write([]byte("\n"))
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
		if err = json.Unmarshal(responseBody, &moonshot); err == nil && moonshot.ID != "" {
			moonshotID = moonshot.ID
		}
	}
FINISHING:
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
	responseStatus = newResponse.Status
	responseStatusCode = newResponse.StatusCode
	responseContentType = filterHeaderFlags(newResponse.Header.Get("Content-Type"))
	if responseStatusCode != http.StatusOK {
		err = &moonshotError{message: string(responseBody)}
	}
}

type Moonshot struct {
	ID    string         `json:"id"`
	Usage *MoonshotUsage `json:"usage"`
}

type MoonshotChunk struct {
	ID      string `json:"id"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		Usage *MoonshotUsage `json:"usage"`
	} `json:"choices"`
}

type MoonshotUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
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
