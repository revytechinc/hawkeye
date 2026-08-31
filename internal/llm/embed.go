// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/revytechinc/hawkeye/internal/headroom"
	"github.com/revytechinc/hawkeye/internal/redact"
)

// Embed runs llama-embedding against a local GGUF. It does not reuse
// llm.local.bin (llama-completion / llama-cli). GPU layers when a GPU
// is present and preferred; otherwise CPU. No cloud API. Secrets are redacted.
func (l Local) Embed(ctx context.Context, text string) ([]float32, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	text = redact.String(text)
	needGPU := l.RequireGPU
	if needGPU && !l.gpuUsable() {
		return nil, ErrGPURequired
	}
	job := headroom.Job{NeedRAM: true, NeedGPU: needGPU}
	if err := headroom.Allow(job, l.Headroom, l.RAMMin, nil, nil, l.VRAMMin); err != nil {
		return nil, err
	}
	useGPU := l.PreferGPU && l.gpuUsable()
	if strings.TrimSpace(l.EmbedModelPath) == "" {
		return nil, ErrNoModel
	}
	bin := resolveEmbedBin(l.Bin)
	if bin == "" {
		return nil, ErrNoBinary
	}
	argv := embedArgs(bin, l.EmbedModelPath, text, useGPU)
	out, err := l.invoke(ctx, argv)
	if err != nil && useGPU && !needGPU && ctx.Err() == nil {
		argv = embedArgs(bin, l.EmbedModelPath, text, false)
		out, err = l.invoke(ctx, argv)
	}
	if err != nil {
		return nil, err
	}
	return parseEmbedding(out)
}

// Model names the configured embedder (path), or the chat model if unset.
func (l Local) Model() string {
	if s := strings.TrimSpace(l.EmbedModelPath); s != "" {
		return s
	}
	return strings.TrimSpace(l.ModelPath)
}

// embedSeparator is a single token that does not appear in playbook
// text. llama-embedding 9426 defaults to newline, which splits a
// playbook into many sequences and yields tens of thousands of dims
// instead of the nomic 768-d mean pool.
const embedSeparator = "<#sep#>"

func embedArgs(bin, model, text string, useGPU bool) []string {
	ngl := "0"
	if useGPU {
		ngl = "99"
	}
	// llama-embedding 9426 rejects --embedding and --no-display-prompt.
	// llama-cli --embedding is invalid on that port. Do not pass either.
	return []string{
		bin,
		"-m", model,
		"-p", text,
		"-ngl", ngl,
		"--embd-output-format", "array",
		"--pooling", "mean",
		"--embd-separator", embedSeparator,
	}
}

func parseEmbedding(s string) ([]float32, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty embedding output")
	}
	if i := strings.Index(s, "["); i >= 0 {
		if j := strings.LastIndex(s, "]"); j > i {
			var raw []float64
			if json.Unmarshal([]byte(s[i:j+1]), &raw) == nil && len(raw) > 0 {
				out := make([]float32, len(raw))
				for k, v := range raw {
					out[k] = float32(v)
				}
				return out, nil
			}
		}
	}
	var out []float32
	var tok []rune
	flush := func() {
		if len(tok) == 0 {
			return
		}
		n, err := strconv.ParseFloat(string(tok), 32)
		tok = tok[:0]
		if err != nil {
			return
		}
		out = append(out, float32(n))
	}
	for _, r := range s {
		if unicode.IsDigit(r) || r == '.' || r == '-' || r == '+' || r == 'e' || r == 'E' {
			tok = append(tok, r)
			continue
		}
		flush()
	}
	flush()
	if len(out) == 0 {
		return nil, fmt.Errorf("no embedding floats parsed")
	}
	return out, nil
}
