// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/revytechinc/hawkeye/internal/redact"
)

// Remote is an OpenAI-compatible chat Completer (Grok / FreeGrok / Claude
// gateways that speak /v1/chat/completions). Secrets stay in env; never in JSON.
type Remote struct {
	Provider string
	Endpoint string
	APIKey   string
	// HTTPDo replaces http.DefaultClient.Do in tests.
	HTTPDo func(*http.Request) (*http.Response, error)
	Model  string
}

func (r Remote) Complete(ctx context.Context, req Request) (Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	endpoint := strings.TrimSpace(r.Endpoint)
	if endpoint == "" {
		return Response{}, fmt.Errorf("remote llm endpoint is not configured")
	}
	if strings.TrimSpace(r.APIKey) == "" {
		return Response{}, fmt.Errorf("remote llm api key is not configured")
	}
	prompt := redact.String(req.Prompt)
	model := strings.TrimSpace(r.Model)
	if model == "" {
		model = defaultRemoteModel(r.Provider)
	}
	payload := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return Response{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return Response{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+r.APIKey)
	do := r.HTTPDo
	if do == nil {
		client := &http.Client{Timeout: 120 * time.Second}
		do = client.Do
	}
	httpResp, err := do(httpReq)
	if err != nil {
		return Response{}, fmt.Errorf("remote llm %s: %w", sanitizeProvider(r.Provider), err)
	}
	defer httpResp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(httpResp.Body, 1<<20))
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return Response{}, fmt.Errorf("remote llm %s: http %d", sanitizeProvider(r.Provider), httpResp.StatusCode)
	}
	text, err := parseChatCompletion(raw)
	if err != nil {
		return Response{}, err
	}
	return Response{
		Text:    redact.String(text),
		Backend: sanitizeProvider(r.Provider),
		UsedGPU: false,
	}, nil
}

func sanitizeProvider(p string) string {
	p = strings.ToLower(strings.TrimSpace(p))
	if p == "" {
		return "remote"
	}
	return p
}

func defaultRemoteModel(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "claude":
		return "claude-3-5-sonnet-latest"
	case "freegrok", "free-grok":
		return "grok-2"
	case "grok", "xai":
		return "grok-2"
	default:
		return "gpt-4o-mini"
	}
}

func parseChatCompletion(raw []byte) (string, error) {
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("remote llm: decode response: %w", err)
	}
	if len(parsed.Choices) == 0 || strings.TrimSpace(parsed.Choices[0].Message.Content) == "" {
		return "", fmt.Errorf("remote llm: empty completion")
	}
	return parsed.Choices[0].Message.Content, nil
}

// SelectOpts picks local vs remote Completer for consult.
type SelectOpts struct {
	Tier           int
	LocalBackend   string
	LocalBin       string
	LocalModelPath string
	PreferGPU      bool
	RequireGPU     bool
	GPUPresent     bool
	RemoteProvider string
	RemoteEndpoint string
	RemoteAPIKey   string
	RemoteModel    string
	Local          Local // optional fully-built local (headroom filled by caller)
}

// SelectCompleter prefers a configured local GGUF; at tier >= 2 falls back to remote.
func SelectCompleter(opts SelectOpts) Completer {
	if strings.TrimSpace(opts.LocalModelPath) != "" && strings.TrimSpace(opts.LocalBackend) != "" {
		loc := opts.Local
		if loc.Backend == "" {
			loc = Local{
				Backend:    opts.LocalBackend,
				Bin:        opts.LocalBin,
				ModelPath:  opts.LocalModelPath,
				PreferGPU:  opts.PreferGPU,
				RequireGPU: opts.RequireGPU,
				GPUPresent: opts.GPUPresent,
			}
		}
		return loc
	}
	if opts.Tier >= 2 &&
		strings.TrimSpace(opts.RemoteProvider) != "" &&
		strings.TrimSpace(opts.RemoteEndpoint) != "" &&
		strings.TrimSpace(opts.RemoteAPIKey) != "" {
		return Remote{
			Provider: opts.RemoteProvider,
			Endpoint: opts.RemoteEndpoint,
			APIKey:   opts.RemoteAPIKey,
			Model:    opts.RemoteModel,
		}
	}
	return None{}
}
