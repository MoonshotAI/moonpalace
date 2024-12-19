package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/MoonshotAI/moonpalace/detector/repeat"
	"github.com/MoonshotAI/moonpalace/merge"
)

type StartConfig struct {
	Port         int16               `yaml:"port"`
	Key          string              `yaml:"key"`
	DetectRepeat *DetectRepeatConfig `yaml:"detect-repeat"`
	ForceStream  bool                `yaml:"force-stream"`
	AutoCache    *AutoCacheConfig    `yaml:"auto-cache"`
}

type DetectRepeatConfig struct {
	Threshold float64 `yaml:"threshold"`
	MinLength int32   `yaml:"min-length"`
}

type AutoCacheConfig struct {
	MinBytes int `yaml:"min-bytes"`
	TTL      int `yaml:"ttl"`
	Cleanup  int `yaml:"cleanup"`
}

const (
	defaultPort = 9988

	defaultRepeatThreshold = 0.5
	defaultRepeatMinLength = 100

	defaultCacheMinBytes = 4 * 1024
	defaultCacheTTL      = 60
	defaultCacheCleanup  = 86400
)

var (
	// defaultAutoCacheConfig is a sentinel variable used to detect whether the
	// user has manually set the --auto-cache option.
	defaultAutoCacheConfig = &AutoCacheConfig{
		MinBytes: defaultCacheMinBytes,
		TTL:      defaultCacheTTL,
		Cleanup:  defaultCacheCleanup,
	}
)

func startCommand() *cobra.Command {
	var cfg *StartConfig
	if MoonConfig.Start != nil {
		cfg = MoonConfig.Start
	} else {
		cfg = &StartConfig{}
	}
	if cfg.Port == 0 {
		cfg.Port = defaultPort
	}
	if cfg.DetectRepeat == nil {
		cfg.DetectRepeat = &DetectRepeatConfig{
			Threshold: defaultRepeatThreshold,
			MinLength: defaultRepeatMinLength,
		}
	} else {
		if cfg.DetectRepeat.Threshold == 0 {
			cfg.DetectRepeat.Threshold = defaultRepeatThreshold
		}
		if cfg.DetectRepeat.MinLength == 0 {
			cfg.DetectRepeat.MinLength = defaultRepeatMinLength
		}
	}
	if cfg.AutoCache == nil {
		cfg.AutoCache = defaultAutoCacheConfig
	} else {
		if cfg.AutoCache.MinBytes == 0 {
			cfg.AutoCache.MinBytes = defaultCacheMinBytes
		}
		if cfg.AutoCache.TTL == 0 {
			cfg.AutoCache.TTL = defaultCacheTTL
		}
		if cfg.AutoCache.Cleanup == 0 {
			cfg.AutoCache.Cleanup = defaultCacheCleanup
		}
	}
	var (
		port            = cfg.Port
		key             = cfg.Key
		detectRepeat    = cfg.DetectRepeat != nil
		repeatThreshold = cfg.DetectRepeat.Threshold
		repeatMinLength = cfg.DetectRepeat.MinLength
		forceStream     = cfg.ForceStream
		autoCache       = cfg.AutoCache != defaultAutoCacheConfig
		cacheMinBytes   = cfg.AutoCache.MinBytes
		cacheTTL        = cfg.AutoCache.TTL
		cacheCleanup    = cfg.AutoCache.Cleanup
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
				autoCache,
				cacheMinBytes,
				cacheTTL,
				cacheCleanup,
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
	flags.Int16VarP(&port, "port", "p", port, "port to listen on")
	flags.StringVarP(&key, "key", "k", key, "API key by default")
	flags.BoolVar(&detectRepeat, "detect-repeat", detectRepeat, "detect and prevent repeating tokens in streaming output")
	flags.Float64Var(&repeatThreshold, "repeat-threshold", repeatThreshold, "repeat threshold, a float between [0, 1]")
	flags.Int32Var(&repeatMinLength, "repeat-min-length", repeatMinLength, "repeat min length, minimum string length to detect repeat")
	flags.BoolVar(&forceStream, "force-stream", forceStream, "force streaming for all chat completions requests")
	flags.BoolVar(&autoCache, "auto-cache", autoCache, "enable automatic caching for requests")
	flags.IntVar(&cacheMinBytes, "cache-min-bytes", cacheMinBytes, "minimum size of bytes to cache")
	flags.IntVar(&cacheTTL, "cache-ttl", cacheTTL, "time to live in seconds for cached requests")
	flags.IntVar(&cacheCleanup, "cache-cleanup", cacheCleanup, "time in seconds to cleanup expired caches")
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
	gzipReaderPool = &sync.Pool{
		New: func() any {
			return new(gzip.Reader)
		},
	}
	gzipWriterPool = &sync.Pool{
		New: func() any {
			z, _ := gzip.NewWriterLevel(nil, gzip.BestCompression)
			return z
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

func getGzipReader(reader io.Reader) (*gzip.Reader, error) {
	gzipReader := gzipReaderPool.Get().(*gzip.Reader)
	if err := gzipReader.Reset(reader); err != nil {
		putGzipReader(gzipReader)
		return nil, err
	}
	return gzipReader, nil
}

func putGzipReader(reader *gzip.Reader) {
	gzipReaderPool.Put(reader)
}

func getGzipWriter(writer io.Writer) *gzip.Writer {
	gzipWriter := gzipWriterPool.Get().(*gzip.Writer)
	gzipWriter.Reset(writer)
	return gzipWriter
}

func putGzipWriter(writer *gzip.Writer) {
	gzipWriterPool.Put(writer)
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
	autoCache bool,
	cacheMinBytes int,
	cacheTTL int,
	cacheCleanup int,
) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			err                       error
			warnings                  []error
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
			latency                   time.Duration
			tokenFinishLatency        time.Duration
		)
		defer func() {
			go func() {
				loggingMutex.Lock()
				defer loggingMutex.Unlock()
				if latency == 0 {
					latency = time.Since(createdAt)
				}
				var (
					responseTPOT int
					responseOTPS float64
				)
				if moonshot != nil {
					if usage := moonshot.Usage; usage != nil {
						timePerOutputToken := (float64(tokenFinishLatency) -
							float64(responseTTFT)*float64(time.Millisecond)) /
							float64(usage.CompletionTokens-_boolToInt(responseTTFT != 0))
						if timePerOutputToken > 0.0 && timePerOutputToken < math.Inf(1) {
							responseTPOT = int(timePerOutputToken / float64(time.Millisecond))
							responseOTPS = 1 / (timePerOutputToken / float64(time.Second))
						}
					}
				}
				var (
					requestHeader  http.Header
					responseHeader http.Header
				)
				if newRequest != nil {
					requestHeader = newRequest.Header
				} else {
					requestHeader = r.Header
				}
				if newResponse != nil {
					responseHeader = newResponse.Header
				} else {
					responseHeader = make(http.Header)
				}
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
					latency,
					tokenFinishLatency,
					err,
					warnings,
					requestHeader,
					responseHeader,
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
					responseTPOT,
					responseOTPS,
					createdAt.Format(time.DateTime),
					latency,
					endpoint,
				)
				if err != nil {
					logFatal(err)
				}
				logNewRow(lastInsertID)
			}()
		}()
		requestBody, err = io.ReadAll(r.Body)
		if err != nil {
			writeProxyError(
				encoder,
				w.Header(),
				w.WriteHeader,
				stepReadRequestBody,
				err,
			)
			return
		}
		if strings.HasSuffix(requestPath, "/chat/completions") && forceStream {
			var streamRequest MoonshotStreamRequest
			json.Unmarshal(requestBody, &streamRequest)
			if streamRequest.Stream != nil {
				requestUseStream = *streamRequest.Stream
			}
			if !requestUseStream {
				requestBody = forceUseStream(requestBody, streamRequest.Stream != nil)
			}
		}
		newRequest, err = http.NewRequestWithContext(
			r.Context(),
			r.Method,
			endpoint+requestPath,
			bytes.NewReader(requestBody),
		)
		if err != nil {
			writeProxyError(
				encoder,
				w.Header(),
				w.WriteHeader,
				stepMakeNewRequest,
				err,
			)
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
		if strings.HasSuffix(requestPath, "/chat/completions") && autoCache {
			cKey := key
			if cKey == "" {
				cKey = strings.TrimSpace(
					strings.TrimPrefix(
						newRequest.Header.Get("Authorization"),
						"Bearer",
					),
				)
			}
			go persistence.RemoveInactiveCaches(
				hashKey(cKey),
				time.Now().
					Add(-time.Duration(cacheCleanup)*time.Second).
					Format(time.DateTime),
			)
			var requestObject struct {
				Messages []*MoonshotMessage `json:"messages"`
				Tools    json.RawMessage    `json:"tools"`
			}
			if err = json.Unmarshal(requestBody, &requestObject); err == nil {
				hashList, nBytes := hashPrefix(
					cacheMinBytes,
					requestObject.Tools,
					requestObject.Messages,
				)
				if len(hashList) > 0 {
					var (
						cache   *Cache
						cacheID string
					)
					cacheID, err = persistence.GetCacheByHashList(r.Context(), hashList, nBytes/2, hashKey(cKey))
					switch {
					case err == nil:
						if cache, err = caching.Get(r.Context(), cKey, cacheID); err == nil && cache.Status != "error" {
							go persistence.UpdateCache(cacheID, time.Now().Format(time.DateTime))
							newRequest.Header.Set("X-Msh-Context-Cache", cacheID)
							newRequest.Header.Set("X-Msh-Context-Cache-Reset-TTL", strconv.Itoa(cacheTTL))
						}
					case errors.Is(err, sql.ErrNoRows):
						rawMessages := gjson.GetBytes(requestBody, "messages").String()
						cache = &Cache{
							Messages: json.RawMessage(rawMessages),
							Tools:    slices.Clone(requestObject.Tools),
							TTL:      cacheTTL,
						}
						if err = caching.Create(r.Context(), cKey, cache); err == nil {
							hash := hashList[len(hashList)-1]
							if err = persistence.SetCache(
								r.Context(),
								cache.ID,
								hash,
								nBytes,
								hashKey(cKey),
								time.Now().Format(time.DateTime),
							); err == nil {
								newRequest.Header.Set("X-Msh-Context-Cache", cache.ID)
								newRequest.Header.Set("X-Msh-Context-Cache-Reset-TTL", strconv.Itoa(cacheTTL))
							}
						}
					default:
						// To avoid affecting proxy requests, we will act as if nothing happened for other errors.
					}
				}
			}
		}
		createdAt = time.Now()
		newResponse, err = httpClient.Do(newRequest)
		if err != nil {
			writeProxyError(
				encoder,
				w.Header(),
				w.WriteHeader,
				stepSendNewRequest,
				err,
			)
			return
		}
		defer newResponse.Body.Close()
		for header, values := range newResponse.Header {
			for _, value := range values {
				w.Header().Add(header, value)
			}
		}
		responseContentType = filterHeaderFlags(newResponse.Header.Get("Content-Type"))
		if !(forceStream && !requestUseStream && responseContentType == "text/event-stream") {
			w.WriteHeader(newResponse.StatusCode)
		}
		if responseContentType == "text/event-stream" {
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
			var (
				scanner        *bufio.Scanner
				responseWriter io.Writer
			)
			if isGzip(newResponse.Header) {
				var (
					gzipReader *gzip.Reader
					gzipWriter *gzip.Writer
				)
				gzipReader, err = getGzipReader(newResponse.Body)
				if err != nil {
					return
				}
				defer putGzipReader(gzipReader)
				defer gzipReader.Close()
				scanner = bufio.NewScanner(gzipReader)
				scanner.Split(splitFunc)
				gzipWriter = getGzipWriter(w)
				defer putGzipWriter(gzipWriter)
				defer gzipWriter.Close()
				responseWriter = gzipWriter
			} else {
				scanner = bufio.NewScanner(newResponse.Body)
				scanner.Split(splitFunc)
				responseWriter = w
			}
		READLINES:
			for scanner.Scan() {
				line := scanner.Bytes()
				if !(forceStream && !requestUseStream) {
					responseWriter.Write(line)
					responseWriter.Write([]byte("\n\n"))
					if flusher, ok := responseWriter.(*gzip.Writer); ok {
						flusher.Flush()
					}
				}
				if len(bytes.TrimSpace(line)) == 0 {
					continue READLINES
				}
				responseBody = append(responseBody, line...)
				responseBody = append(responseBody, "\n\n"...)
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
									if responseTTFT == 0 && hasStreamToken(choice.Delta) {
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
										warnings = append(warnings, errors.New("it seems that your max_tokens value is too small, please set a larger value"))
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
											warnings = append(warnings, errors.New("it appears that there is an issue with content repeating in the current response"))
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
													responseBody = append(responseBody, "data: "...)
													responseBody = append(responseBody, finishChunk...)
													responseBody = append(responseBody, "\n\n"...)
													responseBody = append(responseBody, "data: [DONE]\n\n"...)
													if forceStream && !requestUseStream {
														mergeIn(completion, finishChunk)
													} else {
														responseWriter.Write([]byte("data: "))
														responseWriter.Write(finishChunk)
														responseWriter.Write([]byte("\n\n"))
													}
												}
											}
											if !(forceStream && !requestUseStream) {
												responseWriter.Write([]byte("data: [DONE]"))
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
			tokenFinishLatency = time.Since(createdAt)
			if forceStream && !requestUseStream {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(newResponse.StatusCode)
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
				json.NewEncoder(responseWriter).Encode(completion)
			}
		} else {
			responseBody, err = io.ReadAll(newResponse.Body)
			if err != nil {
				writeProxyError(
					encoder,
					w.Header(),
					w.WriteHeader,
					stepReadResponseBody,
					err,
				)
				return
			}
			tokenFinishLatency = time.Since(createdAt)
			w.Write(responseBody)
			if isGzip(newResponse.Header) {
				var gzipReader *gzip.Reader
				gzipReader, err = gzip.NewReader(bytes.NewReader(responseBody))
				if err != nil {
					return
				}
				defer gzipReader.Close()
				responseBody, err = io.ReadAll(gzipReader)
				if err != nil {
					return
				}
			}
			if strings.HasSuffix(requestPath, "/chat/completions") && responseContentType == "application/json" {
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
								warnings = append(warnings,
									fmt.Errorf("it seems that your max_tokens value is too small, please set a value greater than %d",
										completion.Usage.CompletionTokens))
							}
						}
					}
				}
			}
		}
		if tokenFinishLatency > 0 {
			latency = tokenFinishLatency
		} else {
			latency = time.Since(createdAt)
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
	Stream *bool `json:"stream"`
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
	Role      string `json:"role"`
	Content   string `json:"content"`
	ToolCalls []*struct {
		Function *struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"function"`
	} `json:"tool_calls"`
}

func hasStreamToken(message *MoonshotMessage) bool {
	if message.Content != "" {
		return true
	}
	for _, toolCall := range message.ToolCalls {
		if toolCall != nil && toolCall.Function != nil && toolCall.Function.Arguments != "" {
			return true
		}
	}
	return false
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

const (
	stepReadRequestBody  = "read_request_body"
	stepMakeNewRequest   = "make_new_request"
	stepSendNewRequest   = "send_new_request"
	stepReadResponseBody = "read_response_body"
)

func writeProxyError(
	encoder *json.Encoder,
	header http.Header,
	status func(int),
	typ string,
	err error,
) {
	header.Set("Content-Type", "application/json; charset=utf-8")
	status(http.StatusInternalServerError)
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

func forceUseStream(data []byte, hasStreamKey bool) []byte {
	if !json.Valid(data) {
		return data
	}
	if !hasStreamKey {
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
	sjsonOption := &sjson.Options{
		Optimistic:     true,
		ReplaceInPlace: true,
	}
	data, _ = sjson.SetBytesOptions(data, "stream", true, sjsonOption)
	data, _ = sjson.SetBytesOptions(data, "stream_options.include_usage", true, sjsonOption)
	return data
}

type RepeatDetector struct {
	Automaton    *repeat.SuffixAutomaton
	FinishReason string
}
