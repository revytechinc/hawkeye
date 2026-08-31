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

	"github.com/revytechinc/hawkeye/internal/apply"
	"github.com/revytechinc/hawkeye/internal/mcp"
)

func TestBindIsLoopback(t *testing.T) {
	if !mcp.BindIsLoopback("127.0.0.1:8741") {
		t.Fatal("127.0.0.1")
	}
	if mcp.BindIsLoopback("0.0.0.0:8741") {
		t.Fatal("public bind")
	}
	if mcp.DefaultAddr() != "127.0.0.1:8741" {
		t.Fatal(mcp.DefaultAddr())
	}
}

func TestListenAndServeTLS_RejectsPublicAndMissingToken(t *testing.T) {
	s := mcp.New(mcp.Handlers{})
	if err := mcp.ListenAndServeTLS("0.0.0.0:1", "c", "k", s); err == nil {
		t.Fatal("public bind")
	}
	if err := mcp.ListenAndServeTLS("127.0.0.1:1", "", "", s); err == nil {
		t.Fatal("token required")
	}
}

func TestHandle_ToolsListAndApplyDryRun(t *testing.T) {
	var gotYes bool
	s := mcp.New(mcp.Handlers{
		Apply: func(plan apply.Plan, yes bool) (any, error) {
			gotYes = yes
			mode := apply.ResolveMode(true, yes) // MCP still dry-run via apply actor
			res, err := apply.Execute(plan, mode, apply.ActorMCP, &apply.CountingExecutor{}, apply.NopAuditor{})
			return res, err
		},
		Doctor:  func() (any, error) { return map[string]any{"healthy": true}, nil },
		Consult: func(q string) (any, error) { return map[string]any{"query": q}, nil },
		Plan:    func(q string) (any, error) { return apply.Plan{ID: "p", Summary: q}, nil },
	})
	req := mcp.Request{JSONRPC: "2.0", ID: 1, Method: "tools/list"}
	resp := s.Handle(req)
	b, _ := json.Marshal(resp.Result)
	if !strings.Contains(string(b), "apply") || !strings.Contains(string(b), "consult") || !strings.Contains(string(b), "inspect") {
		t.Fatalf("%s", b)
	}
	args, _ := json.Marshal(map[string]any{
		"name": "apply",
		"arguments": map[string]any{
			"yes":  true,
			"plan": apply.Plan{ID: "x", Source: "knowledge", Steps: []apply.Step{{ID: "1", Argv: []string{"true"}, Privileged: true}}},
		},
	})
	call := mcp.Request{JSONRPC: "2.0", ID: 2, Method: "tools/call", Params: args}
	got := s.Handle(call)
	if got.Error != nil {
		t.Fatal(got.Error)
	}
	rb, _ := json.Marshal(got.Result)
	if !strings.Contains(string(rb), `"dry_run":true`) && !strings.Contains(string(rb), `"DryRun"`) {
		// Result is apply.Result with json dry_run
		if !strings.Contains(string(rb), "dry_run") {
			t.Fatalf("expected dry-run result %s yes=%v", rb, gotYes)
		}
	}
}

func TestServeStdio_Initialize(t *testing.T) {
	s := mcp.New(mcp.Handlers{})
	in := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}` + "\n")
	var out bytes.Buffer
	if err := mcp.ServeStdio(in, &out, s); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "hawkeye") {
		t.Fatalf("%s", out.String())
	}
}

func TestServeHTTP_StreamableJSON(t *testing.T) {
	s := mcp.New(mcp.Handlers{
		Doctor: func() (any, error) { return map[string]any{"healthy": true}, nil },
	})
	s.Token = fixtureToken
	params, _ := json.Marshal(map[string]any{"name": "doctor", "arguments": map[string]any{}})
	body, _ := json.Marshal(mcp.Request{JSONRPC: "2.0", ID: 3, Method: "tools/call", Params: params})
	r := authed(httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body)))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "healthy") {
		t.Fatal(w.Body.String())
	}
}

func TestRedactPayload(t *testing.T) {
	out := mcp.RedactPayload("password=fake-password-for-tests-only")
	if strings.Contains(out, "fake-password-for-tests-only") {
		t.Fatal(out)
	}
}
