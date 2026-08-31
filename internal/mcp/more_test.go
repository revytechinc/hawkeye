// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package mcp_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/revytechinc/hawkeye/internal/apply"
	"github.com/revytechinc/hawkeye/internal/mcp"
)

func TestHandle_InitializedAndMissingHandlers(t *testing.T) {
	s := mcp.New(mcp.Handlers{})
	resp := s.Handle(mcp.Request{JSONRPC: "2.0", ID: 1, Method: "initialized"})
	if resp.Error != nil {
		t.Fatal(resp.Error)
	}
	for _, name := range []string{"consult", "plan", "apply", "doctor", "inspect", "nope"} {
		args, _ := json.Marshal(map[string]any{"name": name, "arguments": map[string]any{"query": "x"}})
		got := s.Handle(mcp.Request{JSONRPC: "2.0", ID: 2, Method: "tools/call", Params: args})
		if got.Error == nil {
			t.Fatalf("expected error for %s", name)
		}
	}
}

func TestHandle_ToolsWithHandlers(t *testing.T) {
	s := mcp.New(mcp.Handlers{
		Consult: func(q string) (any, error) { return q, nil },
		Plan:    func(q string) (any, error) { return apply.Plan{Summary: q}, nil },
		Doctor:  func() (any, error) { return "ok", nil },
		Inspect: func() (any, error) { return "host", nil },
		Apply:   func(p apply.Plan, yes bool) (any, error) { return p.ID, nil },
	})
	for _, name := range []string{"consult", "plan", "doctor", "inspect", "apply"} {
		args, _ := json.Marshal(map[string]any{"name": name, "arguments": map[string]any{"query": "zfs", "plan": apply.Plan{ID: "1"}}})
		got := s.Handle(mcp.Request{JSONRPC: "2.0", ID: 3, Method: "tools/call", Params: args})
		if got.Error != nil {
			t.Fatalf("%s: %+v", name, got.Error)
		}
	}
}

func TestServeHTTP_MethodNotAllowedAndBadJSON(t *testing.T) {
	s := mcp.New(mcp.Handlers{})
	s.Token = fixtureToken
	w := httptest.NewRecorder()
	s.ServeHTTP(w, authed(httptest.NewRequest(http.MethodPut, "/mcp", nil)))
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatal(w.Code)
	}
	w = httptest.NewRecorder()
	s.ServeHTTP(w, authed(httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString("{"))))
	if w.Code == 200 {
		t.Fatal("bad json")
	}
}

func TestBindIsLoopback_LocalhostAndV6(t *testing.T) {
	if !mcp.BindIsLoopback("localhost:1") {
		t.Fatal("localhost")
	}
	if !mcp.BindIsLoopback("[::1]:8741") {
		t.Fatal("::1")
	}
	if mcp.BindIsLoopback("bad") {
		t.Fatal("bad")
	}
}

func TestHandle_BadToolParams(t *testing.T) {
	s := mcp.New(mcp.Handlers{})
	got := s.Handle(mcp.Request{JSONRPC: "2.0", ID: 1, Method: "tools/call", Params: json.RawMessage(`n`)})
	if got.Error == nil {
		t.Fatal("expected")
	}
}
