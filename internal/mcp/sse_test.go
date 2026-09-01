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
	body, _ := json.Marshal(mcp.Request{JSONRPC: "2.0", ID: 7, Method: "tools/call", Params: params})
	r := authed(httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body)))
	r.Header.Set("Accept", "application/json, text/event-stream")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("code %d body=%s", w.Code, w.Body.String())
	}
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/event-stream") {
		t.Fatalf("content-type %q", ct)
	}
	out := w.Body.String()
	if !strings.Contains(out, "event: message") {
		t.Fatalf("missing event: message: %s", out)
	}
	if !strings.Contains(out, "data:") || !strings.Contains(out, "healthy") {
		t.Fatalf("missing SSE data payload: %s", out)
	}
	if strings.Contains(out, fixtureToken) {
		t.Fatal("sse leaked token")
	}
}

func TestServeHTTP_POST_JSONWhenAcceptJSONOnly(t *testing.T) {
	s := mcp.New(mcp.Handlers{
		Doctor: func() (any, error) { return map[string]any{"healthy": true}, nil },
	})
	s.Token = fixtureToken
	params, _ := json.Marshal(map[string]any{"name": "doctor", "arguments": map[string]any{}})
	body, _ := json.Marshal(mcp.Request{JSONRPC: "2.0", ID: 8, Method: "tools/call", Params: params})
	r := authed(httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body)))
	r.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("content-type %q", w.Header().Get("Content-Type"))
	}
	if strings.Contains(w.Body.String(), "event:") {
		t.Fatalf("JSON Accept must not SSE: %s", w.Body.String())
	}
}

func TestServeHTTP_POST_NotificationReturns202(t *testing.T) {
	s := mcp.New(mcp.Handlers{})
	s.Token = fixtureToken
	body, _ := json.Marshal(mcp.Request{JSONRPC: "2.0", Method: "notifications/initialized"})
	r := authed(httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body)))
	r.Header.Set("Accept", "application/json, text/event-stream")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusAccepted {
		t.Fatalf("notification: got %d body=%s", w.Code, w.Body.String())
	}
}

func TestServeHTTP_GET_SSEIncludesMessageReady(t *testing.T) {
	s := mcp.New(mcp.Handlers{})
	s.Token = fixtureToken
	r := authed(httptest.NewRequest(http.MethodGet, "/mcp", nil))
	r.Header.Set("Accept", "text/event-stream")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatal(w.Body.String())
	}
	out := w.Body.String()
	if !strings.Contains(out, "event: message") && !strings.Contains(out, "event: endpoint") {
		t.Fatalf("expected SSE events: %s", out)
	}
	if !strings.Contains(w.Header().Get("Cache-Control"), "no-cache") {
		t.Fatalf("cache-control %q", w.Header().Get("Cache-Control"))
	}
}
