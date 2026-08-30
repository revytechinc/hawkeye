// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package mcp_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/revytechinc/hawkeye/internal/mcp"
)

const fixtureToken = "test-mcp-token-fixture-not-production"
const wrongToken = "wrong-mcp-token-fixture"

func authed(r *http.Request) *http.Request {
	r.Header.Set("Authorization", "Bearer "+fixtureToken)
	return r
}

func TestServeHTTP_MissingTokenIs401(t *testing.T) {
	s := mcp.New(mcp.Handlers{
		Doctor: func() (any, error) { return map[string]any{"healthy": true}, nil },
	})
	s.Token = fixtureToken
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`)))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("missing token: got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Header().Get("WWW-Authenticate"), "Bearer") {
		t.Fatal("expected WWW-Authenticate Bearer")
	}
	if strings.Contains(w.Body.String(), fixtureToken) {
		t.Fatal("response leaked fixture token")
	}
}

func TestServeHTTP_WrongTokenIs401(t *testing.T) {
	s := mcp.New(mcp.Handlers{})
	s.Token = fixtureToken
	r := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`))
	r.Header.Set("Authorization", "Bearer "+wrongToken)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("wrong token: got %d", w.Code)
	}
	if strings.Contains(w.Body.String(), fixtureToken) || strings.Contains(w.Body.String(), wrongToken) {
		t.Fatal("response leaked token material")
	}
}

func TestServeHTTP_EmptyServerTokenIs401(t *testing.T) {
	s := mcp.New(mcp.Handlers{})
	r := authed(httptest.NewRequest(http.MethodGet, "/mcp", nil))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("empty server token must not authorize, got %d", w.Code)
	}
}

func TestServeHTTP_ValidTokenNot401(t *testing.T) {
	s := mcp.New(mcp.Handlers{})
	s.Token = fixtureToken
	r := authed(httptest.NewRequest(http.MethodGet, "/mcp", nil))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code == http.StatusUnauthorized {
		t.Fatal("valid token must not 401")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("got %d body=%s", w.Code, w.Body.String())
	}
}

func TestServeHTTP_ValidTokenInitialize(t *testing.T) {
	s := mcp.New(mcp.Handlers{})
	s.Token = fixtureToken
	body, _ := json.Marshal(mcp.Request{JSONRPC: "2.0", ID: 1, Method: "initialize"})
	r := authed(httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body)))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("got %d %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "hawkeye") {
		t.Fatal(w.Body.String())
	}
	if strings.Contains(w.Body.String(), fixtureToken) {
		t.Fatal("initialize leaked token")
	}
}

func TestListenAndServe_RejectsPublicAndMissingToken(t *testing.T) {
	s := mcp.New(mcp.Handlers{})
	if err := mcp.ListenAndServe("0.0.0.0:1", "", "", fixtureToken, s); err == nil {
		t.Fatal("public bind")
	}
	if err := mcp.ListenAndServe("127.0.0.1:1", "", "", "", s); err == nil {
		t.Fatal("token required")
	}
}

func TestResolveToken_FromEnvAndFile(t *testing.T) {
	env := map[string]string{mcp.DefaultTokenEnv: fixtureToken}
	got, err := mcp.ResolveToken(func(k string) string { return env[k] }, "")
	if err != nil || got != fixtureToken {
		t.Fatalf("env: %q %v", got, err)
	}

	dir := t.TempDir()
	p := filepath.Join(dir, "mcp.token")
	if err := os.WriteFile(p, []byte(fixtureToken+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	fileEnv := map[string]string{mcp.TokenFileEnv: p}
	got, err = mcp.ResolveToken(func(k string) string { return fileEnv[k] }, mcp.DefaultTokenEnv)
	if err != nil || got != fixtureToken {
		t.Fatalf("file: %q %v", got, err)
	}

	if err := os.Chmod(p, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := mcp.ReadTokenFile(p); err == nil {
		t.Fatal("0644 token file must be rejected")
	}

	if _, err := mcp.ResolveToken(func(string) string { return "" }, mcp.DefaultTokenEnv); err == nil {
		t.Fatal("missing token")
	}
}

func TestDefaultTokenEnv(t *testing.T) {
	if mcp.DefaultTokenEnv != "HAWKEYE_MCP_TOKEN" {
		t.Fatal(mcp.DefaultTokenEnv)
	}
}
