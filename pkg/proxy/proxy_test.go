package proxy

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexandrmotologa/tracereplay/pkg/engine"
	"github.com/alexandrmotologa/tracereplay/pkg/models"
	"github.com/alexandrmotologa/tracereplay/pkg/storage"
)

func TestProxyOpenAIChat(t *testing.T) {
	store, err := storage.New(":memory:")
	if err != nil {
		t.Fatalf("failed to init storage: %v", err)
	}
	defer store.Close()

	eng := engine.New(store, nil)
	prx := New(eng, "", "")

	// 1. Standard OpenAI Chat Completion
	reqBody := map[string]any{
		"model": "gpt-4o",
		"messages": []map[string]string{
			{"role": "user", "content": "How do I fix memory leak in Go?"},
		},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TraceReplay-Session", "test-session-42")

	w := httptest.NewRecorder()
	prx.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse proxy response: %v", err)
	}
	if resp["object"] != "chat.completion" {
		t.Errorf("expected chat.completion object, got %v", resp["object"])
	}

	// 2. Streaming OpenAI Chat Completion
	streamReqBody := map[string]any{
		"model":  "gpt-4o",
		"stream": true,
		"messages": []map[string]string{
			{"role": "user", "content": "Explain goroutines"},
		},
	}
	streamBytes, _ := json.Marshal(streamReqBody)
	sReq := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(streamBytes))
	sReq.Header.Set("Content-Type", "application/json")
	sReq.Header.Set("X-TraceReplay-Session", "test-session-42")

	sW := httptest.NewRecorder()
	prx.ServeHTTP(sW, sReq)

	if sW.Code != http.StatusOK {
		t.Fatalf("expected 200 for stream, got %d", sW.Code)
	}
	if !strings.Contains(sW.Body.String(), "data: [DONE]") {
		t.Errorf("streaming output missing [DONE]")
	}

	// 3. Anthropic Messages
	anthropicReqBody := map[string]any{
		"model": "claude-3-7-sonnet",
		"messages": []map[string]string{
			{"role": "user", "content": "Refactor auth handler"},
		},
	}
	antBytes, _ := json.Marshal(anthropicReqBody)
	antReq := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(antBytes))
	antReq.Header.Set("Content-Type", "application/json")
	antReq.Header.Set("X-TraceReplay-Session", "anthropic-session-1")

	antW := httptest.NewRecorder()
	prx.ServeHTTP(antW, antReq)

	if antW.Code != http.StatusOK {
		t.Fatalf("expected 200 for Anthropic, got %d", antW.Code)
	}

	// Verify step count in storage
	trajs, err := store.ListTrajectories(models.FilterParams{})
	if err != nil || len(trajs) != 2 {
		t.Fatalf("expected 2 recorded trajectories, got %d", len(trajs))
	}
}
