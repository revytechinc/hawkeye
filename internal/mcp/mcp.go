// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package mcp

import (
	"crypto/subtle"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/revytechinc/hawkeye/internal/apply"
	"github.com/revytechinc/hawkeye/internal/redact"
)

const DefaultTokenEnv = "HAWKEYE_MCP_TOKEN"
const TokenFileEnv = "HAWKEYE_MCP_TOKEN_FILE"

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type Response struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Result  any    `json:"result,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type ToolCall struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type Handlers struct {
	Consult func(query string) (any, error)
	Plan    func(query string) (any, error)
	Apply   func(plan apply.Plan, yes bool) (any, error)
	Doctor  func() (any, error)
}

type Server struct {
	Handlers Handlers
	Token    string
}

func New(h Handlers) *Server { return &Server{Handlers: h} }

func (s *Server) Handle(req Request) Response {
	resp := Response{JSONRPC: "2.0", ID: req.ID}
	switch req.Method {
	case "initialize":
		resp.Result = map[string]any{
			"protocolVersion": "2025-03-26",
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "hawkeye", "version": "0.1.0"},
		}
	case "tools/list":
		resp.Result = map[string]any{"tools": toolList()}
	case "tools/call":
		var call ToolCall
		if err := json.Unmarshal(req.Params, &call); err != nil {
			resp.Error = &Error{Code: -32602, Message: err.Error()}
			return resp
		}
		result, err := s.callTool(call)
		if err != nil {
			resp.Error = &Error{Code: -32000, Message: redact.String(err.Error())}
			return resp
		}
		resp.Result = result
	case "notifications/initialized", "initialized":
		resp.Result = map[string]any{"ok": true}
	default:
		resp.Error = &Error{Code: -32601, Message: "method not found: " + req.Method}
	}
	return resp
}

func toolList() []map[string]any {
	return []map[string]any{
		{"name": "consult", "description": "Diagnose using knowledge FTS and optional LLM. Never writes."},
		{"name": "plan", "description": "Produce a JSON plan. No mutation."},
		{"name": "apply", "description": "Apply a plan. Defaults to dry-run. Privileged mutation is operator-only."},
		{"name": "doctor", "description": "Service health: config, permissions, pidfile, dependencies, headroom."},
	}
}

func (s *Server) callTool(call ToolCall) (any, error) {
	args := map[string]any{}
	if len(call.Arguments) > 0 {
		_ = json.Unmarshal(call.Arguments, &args)
	}
	redactMap(args)
	switch call.Name {
	case "consult":
		q, _ := args["query"].(string)
		if s.Handlers.Consult == nil {
			return nil, fmt.Errorf("consult handler missing")
		}
		return s.Handlers.Consult(q)
	case "plan":
		q, _ := args["query"].(string)
		if s.Handlers.Plan == nil {
			return nil, fmt.Errorf("plan handler missing")
		}
		return s.Handlers.Plan(q)
	case "apply":
		yes, _ := args["yes"].(bool)
		var plan apply.Plan
		if raw, ok := args["plan"]; ok {
			b, _ := json.Marshal(raw)
			_ = json.Unmarshal(b, &plan)
		}
		if s.Handlers.Apply == nil {
			return nil, fmt.Errorf("apply handler missing")
		}
		// MCP actor: privileged apply is dry-run even if yes=true.
		return s.Handlers.Apply(plan, yes)
	case "doctor":
		if s.Handlers.Doctor == nil {
			return nil, fmt.Errorf("doctor handler missing")
		}
		return s.Handlers.Doctor()
	default:
		return nil, fmt.Errorf("unknown tool %q", call.Name)
	}
}

func redactMap(m map[string]any) {
	for k, v := range m {
		switch t := v.(type) {
		case string:
			m[k] = redact.String(t)
		case map[string]any:
			redactMap(t)
		}
	}
}

func ServeStdio(r io.Reader, w io.Writer, s *Server) error {
	dec := json.NewDecoder(r)
	enc := json.NewEncoder(w)
	for {
		var req Request
		if err := dec.Decode(&req); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		resp := s.Handle(req)
		if err := enc.Encode(resp); err != nil {
			return err
		}
	}
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(h) < len(prefix) {
		return ""
	}
	if !strings.EqualFold(h[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(h[len(prefix):])
}

func tokenOK(got, want string) bool {
	if want == "" {
		return false
	}
	gb := []byte(got)
	wb := []byte(want)
	if len(gb) != len(wb) {
		_ = subtle.ConstantTimeCompare(wb, wb)
		return false
	}
	return subtle.ConstantTimeCompare(gb, wb) == 1
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="hawkeye"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"unauthorized"}` + "\n"))
}

func (s *Server) authorized(r *http.Request) bool {
	return tokenOK(bearerToken(r), s.Token)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		unauthorized(w)
		return
	}
	switch r.Method {
	case http.MethodPost:
		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		resp := s.Handle(req)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("MCP-Protocol-Version", "2025-03-26")
		_ = json.NewEncoder(w).Encode(resp)
	case http.MethodGet, http.MethodHead:
		// Streamable HTTP GET after auth opens an SSE stream (spec 2025-03-26).
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("MCP-Protocol-Version", "2025-03-26")
		w.Header().Set("Cache-Control", "no-cache, no-store")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)
		if r.Method != http.MethodHead {
			_, _ = io.WriteString(w, ": hawkeye mcp\n\n")
			_, _ = io.WriteString(w, "event: endpoint\ndata: /mcp\n\n")
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func ListenAndServe(addr, certFile, keyFile, token string, s *Server) error {
	if !BindIsLoopback(addr) {
		return fmt.Errorf("MCP HTTP must bind loopback, got %q", addr)
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("MCP HTTP requires a bearer token from env")
	}
	s.Token = token
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	if certFile != "" || keyFile != "" {
		if certFile == "" || keyFile == "" {
			_ = ln.Close()
			return fmt.Errorf("MCP TLS requires both tls_cert and tls_key")
		}
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			_ = ln.Close()
			return err
		}
		ln = tls.NewListener(ln, &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12})
	}
	mux := http.NewServeMux()
	mux.Handle("/mcp", s)
	mux.Handle("/mcp/", s)
	mux.Handle("/", s)
	srv := &http.Server{Addr: addr, Handler: mux}
	return srv.Serve(ln)
}

func ListenAndServeTLS(addr, certFile, keyFile string, s *Server) error {
	return ListenAndServe(addr, certFile, keyFile, s.Token, s)
}

func DefaultAddr() string { return "127.0.0.1:8741" }

func BindIsLoopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func RedactPayload(s string) string {
	return redact.String(s)
}

func ToolsUseApplyGate() bool { return true }

func NormalizeBind(addr string) string {
	return strings.TrimSpace(addr)
}

func ResolveToken(getenv func(string) string, envName string) (string, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	if strings.TrimSpace(envName) == "" {
		envName = DefaultTokenEnv
	}
	if t := strings.TrimSpace(getenv(envName)); t != "" {
		return t, nil
	}
	if p := strings.TrimSpace(getenv(TokenFileEnv)); p != "" {
		return ReadTokenFile(p)
	}
	return "", fmt.Errorf("MCP HTTP requires %s or %s", envName, TokenFileEnv)
}

func ReadTokenFile(path string) (string, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("MCP token file: %w", err)
	}
	if fi.Mode().Perm()&0o077 != 0 {
		return "", fmt.Errorf("MCP token file must be mode 0600")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("MCP token file: %w", err)
	}
	t := strings.TrimSpace(string(b))
	if t == "" {
		return "", fmt.Errorf("MCP token file is empty")
	}
	return t, nil
}
