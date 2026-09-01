// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

// MCP Streamable HTTP POST SSE probe for FreeBSD e2e (no python3 required).
// FAKE tokens only. Build: GOOS=freebsd CGO_ENABLED=0 go build -o mcpcheck scripts/mcpcheck.go
package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	base := "http://127.0.0.1:8741/mcp"
	if len(os.Args) > 1 {
		base = os.Args[1]
	}
	token := os.Getenv("HAWKEYE_MCP_TOKEN")
	body := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`)
	client := &http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequest(http.MethodPost, base, bytes.NewReader(body))
	if err != nil {
		fail(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		fail(err)
	}
	b, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	ct := resp.Header.Get("Content-Type")
	fmt.Printf("CT=%s\n%s\n", ct, b)
	if !strings.Contains(ct, "text/event-stream") || !bytes.Contains(b, []byte("event: message")) {
		fail(fmt.Errorf("want SSE event: message, got ct=%q body=%s", ct, b))
	}

	req2, err := http.NewRequest(http.MethodPost, base, bytes.NewReader(body))
	if err != nil {
		fail(err)
	}
	req2.Header.Set("Accept", "text/event-stream")
	req2.Header.Set("Content-Type", "application/json")
	resp2, err := client.Do(req2)
	if err != nil {
		fail(err)
	}
	_, _ = io.Copy(io.Discard, resp2.Body)
	_ = resp2.Body.Close()
	ct2 := resp2.Header.Get("Content-Type")
	fmt.Printf("UNAUTH_CODE=%d CT=%s\n", resp2.StatusCode, ct2)
	if resp2.StatusCode != http.StatusUnauthorized || strings.Contains(ct2, "text/event-stream") {
		fail(fmt.Errorf("want 401 JSON, got %d %q", resp2.StatusCode, ct2))
	}
	fmt.Println("MCP_SSE_OK")
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
