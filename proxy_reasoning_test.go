package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestForceStreamPreservesReasoningContent(t *testing.T) {
	const stream = "data: {\"id\":\"cmpl-test\",\"object\":\"chat.completion.chunk\",\"model\":\"kimi-k3\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"reasoning_content\":\"thinking \"}}]}\n\n" +
		"data: {\"id\":\"cmpl-test\",\"choices\":[{\"index\":0,\"delta\":{\"reasoning_content\":\"continues\",\"content\":\"answer\"},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n"
	upstreamRequests := make(chan []byte, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		upstreamRequests <- body
		w.Header().Set("Content-Type", "text/event-stream")
		w.(http.Flusher).Flush()
		io.WriteString(w, stream)
	}))
	defer upstream.Close()
	upstreamURL, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	previousClient := httpClient
	httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = upstreamURL.Scheme
		req.URL.Host = upstreamURL.Host
		req.Host = upstreamURL.Host
		return http.DefaultTransport.RoundTrip(req)
	})}
	defer func() { httpClient = previousClient }()

	proxy := httptest.NewServer(http.HandlerFunc(buildProxy("", false, 0.5, 100, true, false, 4096, 60, 86400)))
	defer proxy.Close()
	resp, err := http.Post(proxy.URL+"/v1/chat/completions", "application/json", strings.NewReader(`{"model":"kimi-k3","messages":[],"stream":false}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("proxy status = %d, body = %s", resp.StatusCode, responseBody)
	}
	var upstreamPayload map[string]any
	if err := json.Unmarshal(<-upstreamRequests, &upstreamPayload); err != nil {
		t.Fatal(err)
	}
	if upstreamPayload["stream"] != true {
		t.Fatalf("upstream stream = %v, want true", upstreamPayload["stream"])
	}
	var completion struct {
		Choices []struct {
			Message struct {
				Content          string `json:"content"`
				ReasoningContent string `json:"reasoning_content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(responseBody, &completion); err != nil {
		t.Fatal(err)
	}
	if len(completion.Choices) != 1 {
		t.Fatalf("choice count = %d, want 1", len(completion.Choices))
	}
	if got, want := completion.Choices[0].Message.ReasoningContent, "thinking continues"; got != want {
		t.Fatalf("reasoning_content = %q, want %q", got, want)
	}
	if got, want := completion.Choices[0].Message.Content, "answer"; got != want {
		t.Fatalf("content = %q, want %q", got, want)
	}
}
