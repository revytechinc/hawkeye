// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package llm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/revytechinc/hawkeye/internal/headroom"
	"github.com/revytechinc/hawkeye/internal/redact"
)

var (
	ErrNoModel     = errors.New("local llm model is not configured")
	ErrNoBinary    = errors.New("local llm binary is not configured")
	ErrGPURequired = errors.New("gpu required for this job but none is present")
)

// lookPath is exec.LookPath. Tests empty PATH so llama-cli is not found.
var lookPath = exec.LookPath

type Request struct {
	Prompt  string
	NeedGPU bool
	NeedRAM bool
}

type Response struct {
	Text    string `json:"text"`
	Backend string `json:"backend"`
	UsedGPU bool   `json:"used_gpu"`
}

type Completer interface {
	Complete(ctx context.Context, req Request) (Response, error)
}

// Local invokes a configured llama.cpp-style binary (llama-cli / llama.cpp)
// with a local GGUF. GPU layers are enabled when a GPU is present and
// preferred; otherwise the job stays on CPU. Missing GPU does not block
// CPU jobs. Consumption-based headroom still uses Allow().
type Local struct {
	Backend    string
	Bin        string
	ModelPath  string
	PreferGPU  bool
	RequireGPU bool
	GPUPresent bool
	Headroom   headroom.Snapshot
	RAMMin     *int64
	VRAMMin    *int64
	// Run, if set, replaces exec (tests). Production uses the configured Bin.
	Run func(ctx context.Context, argv []string) (string, error)
}

func (l Local) Complete(ctx context.Context, req Request) (Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	req.Prompt = redact.String(req.Prompt)
	needGPU := req.NeedGPU || l.RequireGPU
	if needGPU && !l.GPUPresent {
		return Response{}, ErrGPURequired
	}
	job := headroom.Job{NeedRAM: true, NeedGPU: needGPU}
	if err := headroom.Allow(job, l.Headroom, l.RAMMin, nil, nil, l.VRAMMin); err != nil {
		return Response{}, err
	}
	useGPU := l.PreferGPU && l.GPUPresent
	if strings.TrimSpace(l.ModelPath) == "" {
		return Response{Backend: l.Backend, UsedGPU: useGPU}, ErrNoModel
	}
	bin := resolveBin(l.Bin)
	if bin == "" {
		return Response{Backend: l.Backend, UsedGPU: useGPU}, ErrNoBinary
	}
	argv := cliArgs(bin, l.ModelPath, req.Prompt, useGPU)
	out, err := l.invoke(ctx, argv)
	if err != nil {
		return Response{Backend: l.Backend, UsedGPU: useGPU}, err
	}
	return Response{
		Text:    strings.TrimSpace(out),
		Backend: l.Backend,
		UsedGPU: useGPU,
	}, nil
}

func resolveBin(explicit string) string {
	if strings.TrimSpace(explicit) != "" {
		return explicit
	}
	for _, name := range []string{"llama-cli", "llama.cpp"} {
		if p, err := lookPath(name); err == nil && p != "" {
			return p
		}
	}
	return ""
}

func cliArgs(bin, model, prompt string, useGPU bool) []string {
	ngl := "0"
	if useGPU {
		ngl = "99"
	}
	return []string{
		bin,
		"-m", model,
		"-p", prompt,
		"--no-display-prompt",
		"-n", "256",
		"-ngl", ngl,
	}
}

func (l Local) invoke(ctx context.Context, argv []string) (string, error) {
	if l.Run != nil {
		return l.Run(ctx, argv)
	}
	return runLocal(ctx, argv)
}

func runLocal(ctx context.Context, argv []string) (string, error) {
	if len(argv) == 0 {
		return "", ErrNoBinary
	}
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// Do not return stderr (paths / operator guts) to the TTY.
		return "", fmt.Errorf("local llm %s: %w", filepath.Base(argv[0]), err)
	}
	return stdout.String(), nil
}

type None struct{}

func (None) Complete(context.Context, Request) (Response, error) {
	return Response{}, ErrNoModel
}
