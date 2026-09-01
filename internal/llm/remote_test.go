// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package llm_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/revytechinc/hawkeye/internal/llm"
)

func TestRemote_Complete_OpenAICompatible(t *testing.T) {
	var sawAuth, sawBody string
	r := llm.Remote{
		Provider: "grok",
		Endpoint: "https://api.example.test/v1/chat/completions",
		APIKey:   "FAKE_TEST_KEY_NOT_A_REAL_SECRET",
		HTTPDo: func(req *http.Request) (*http.Response, error) {
			sawAuth = req.Header.Get("Authorization")
			b, _ := io.ReadAll(req.Body)
			sawBody = string(b)
			body := `{"choices":[{"message":{"content":"zpool status looks fine"}}]}`
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		},
	}
	resp, err := r.Complete(context.Background(), llm.Request{Prompt: "diagnose zpool"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp.Text, "zpool status") {
		t.Fatalf("text %q", resp.Text)
	}
	if resp.Backend != "grok" {
		t.Fatalf("backend %q", resp.Backend)
	}
	if !strings.HasPrefix(sawAuth, "Bearer FAKE_TEST_KEY") {
		t.Fatalf("auth %q", sawAuth)
	}
	if !strings.Contains(sawBody, "diagnose zpool") {
		t.Fatalf("body %q", sawBody)
	}
}

func TestRemote_Complete_RedactsPrompt(t *testing.T) {
	var sawBody string
	r := llm.Remote{
		Provider: "claude",
		Endpoint: "https://api.example.test/v1/chat/completions",
		APIKey:   "FAKE_TEST_KEY_NOT_A_REAL_SECRET",
		HTTPDo: func(req *http.Request) (*http.Response, error) {
			b, _ := io.ReadAll(req.Body)
			sawBody = string(b)
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"choices":[{"message":{"content":"ok"}}]}`)),
				Header:     make(http.Header),
			}, nil
		},
	}
	secret := "password=fake-password-for-tests-only"
	_, err := r.Complete(context.Background(), llm.Request{Prompt: "check " + secret})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(sawBody, "fake-password-for-tests-only") {
		t.Fatalf("secret leaked to remote: %s", sawBody)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(sawBody), &payload); err != nil {
		t.Fatal(err)
	}
}

func TestRemote_Complete_RequiresEndpointAndKey(t *testing.T) {
	_, err := (llm.Remote{Provider: "grok"}).Complete(context.Background(), llm.Request{Prompt: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
	_, err = (llm.Remote{Provider: "grok", Endpoint: "https://x", APIKey: ""}).Complete(context.Background(), llm.Request{Prompt: "x"})
	if err == nil {
		t.Fatal("expected missing key")
	}
}

func TestRemote_Complete_HTTPError(t *testing.T) {
	r := llm.Remote{
		Provider: "freegrok",
		Endpoint: "https://api.example.test/v1/chat/completions",
		APIKey:   "FAKE_TEST_KEY_NOT_A_REAL_SECRET",
		HTTPDo: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 401,
				Body:       io.NopCloser(strings.NewReader(`{"error":"nope"}`)),
				Header:     make(http.Header),
			}, nil
		},
	}
	_, err := r.Complete(context.Background(), llm.Request{Prompt: "hi"})
	if err == nil {
		t.Fatal("expected http error")
	}
	if strings.Contains(err.Error(), "FAKE_TEST_KEY") {
		t.Fatalf("key leaked in error: %v", err)
	}
}

func TestSelectCompleter_PrefersRemoteWhenLocalEmptyTier2(t *testing.T) {
	c := llm.SelectCompleter(llm.SelectOpts{
		Tier:           2,
		LocalBackend:   "llama.cpp",
		LocalModelPath: "",
		RemoteProvider: "grok",
		RemoteEndpoint: "https://api.example.test/v1/chat/completions",
		RemoteAPIKey:   "FAKE_TEST_KEY_NOT_A_REAL_SECRET",
	})
	if _, ok := c.(llm.Remote); !ok {
		t.Fatalf("got %T", c)
	}
}

func TestSelectCompleter_LocalWhenModelSet(t *testing.T) {
	c := llm.SelectCompleter(llm.SelectOpts{
		Tier:           2,
		LocalBackend:   "llama.cpp",
		LocalModelPath: "/models/x.gguf",
		RemoteProvider: "grok",
		RemoteEndpoint: "https://api.example.test/v1/chat/completions",
		RemoteAPIKey:   "FAKE_TEST_KEY_NOT_A_REAL_SECRET",
	})
	if _, ok := c.(llm.Local); !ok {
		t.Fatalf("got %T", c)
	}
}
