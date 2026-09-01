// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package mcp_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/revytechinc/hawkeye/internal/mcp"
)

func TestServeHTTP_POST_SSEWhenAcceptEventStream(t *testing.T) {
	s := mcp.New(mcp.Handlers{
		Doctor: func() (any, error) { return map[string]any{"healthy": true}, nil },
	})
	s.Token = fixtureToken
	params, _ := json.Marshal(map[string]any{"name": "doctor", "arguments": map[string]any{}})
	body, _ := json.Marshal(mcp.Request{JSONRPC: "2.0", ID: 3, Method: "tools/call", Params: params})
	r := authed(httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body)))
	r.Header.Set("Accept", "application/json, text/event-stream")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("got %d body=%s", w.Code, w.Body.String())
	}
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/event-stream") {
		t.Fatalf("POST with Accept event-stream must be SSE, got %q body=%s", ct, w.Body.String())
	}
	got := w.Body.String()
	if !strings.Contains(got, "event: message") {
		t.Fatalf("SSE must use event: message:\n%s", got)
	}
	if !strings.Contains(got, "data: ") || !strings.Contains(got, `"healthy"`) {
		t.Fatalf("SSE data must carry the JSON-RPC result:\n%s", got)
	}
	if strings.HasPrefix(strings.TrimSpace(got), "{") {
		t.Fatalf("must not be a bare JSON body:\n%s", got)
	}
	if strings.Contains(got, fixtureToken) {
		t.Fatal("sse leaked token")
	}
}

func TestServeHTTP_POST_JSONWhenAcceptJSONOnly(t *testing.T) {
	s := mcp.New(mcp.Handlers{
		Doctor: func() (any, error) { return map[string]any{"healthy": true}, nil },
	})
	s.Token = fixtureToken
	params, _ := json.Marshal(map[string]any{"name": "doctor", "arguments": map[string]any{}})
	body, _ := json.Marshal(mcp.Request{JSONRPC: "2.0", ID: 4, Method: "tools/call", Params: params})
	r := authed(httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body)))
	r.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatal(w.Body.String())
	}
	if strings.Contains(w.Header().Get("Content-Type"), "text/event-stream") {
		t.Fatalf("JSON-only Accept must stay JSON: %s", w.Header().Get("Content-Type"))
	}
	if !json.Valid([]byte(strings.TrimSpace(w.Body.String()))) {
		t.Fatalf("want JSON: %s", w.Body.String())
	}
}

func TestServeHTTP_POST_SSEUnauthorized(t *testing.T) {
	s := mcp.New(mcp.Handlers{})
	s.Token = fixtureToken
	r := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`))
	r.Header.Set("Accept", "text/event-stream")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("SSE POST without token: got %d", w.Code)
	}
	if strings.Contains(w.Header().Get("Content-Type"), "text/event-stream") {
		t.Fatal("401 must not be an SSE stream")
	}
}

func TestWantsSSE(t *testing.T) {
	if !mcp.WantsSSE("text/event-stream") {
		t.Fatal("bare event-stream")
	}
	if !mcp.WantsSSE("application/json, text/event-stream") {
		t.Fatal("spec client Accept")
	}
	if !mcp.WantsSSE("text/event-stream; charset=utf-8") {
		t.Fatal("event-stream with parameter")
	}
	if mcp.WantsSSE("application/json") {
		t.Fatal("json only")
	}
	if mcp.WantsSSE("") {
		t.Fatal("empty")
	}
	if mcp.WantsSSE("text/html") {
		t.Fatal("html")
	}
}
