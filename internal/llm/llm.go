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

// Local invokes a configured llama.cpp-style binary (llama-completion,
// llama-cli, llama.cpp) with a local GGUF. llama-cli 9426 is
// conversation-only; Complete passes --single-turn --simple-io so the
// panic session does not hang on `>`. GPU layers when a GPU is present
// and VRAM is known; otherwise CPU. Missing GPU does not block CPU jobs.
type Local struct {
	Backend        string
	Bin            string
	ModelPath      string
	EmbedModelPath string
	PreferGPU      bool
	RequireGPU     bool
	GPUPresent     bool
	Headroom       headroom.Snapshot
	RAMMin         *int64
	VRAMMin        *int64
	// Run, if set, replaces exec (tests). Production uses the configured Bin.
	Run func(ctx context.Context, argv []string) (string, error)
}

func (l Local) Complete(ctx context.Context, req Request) (Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	req.Prompt = redact.String(req.Prompt)
	needGPU := req.NeedGPU || l.RequireGPU
	if needGPU && !l.gpuUsable() {
		return Response{}, ErrGPURequired
	}
	job := headroom.Job{NeedRAM: true, NeedGPU: needGPU}
	if err := headroom.Allow(job, l.Headroom, l.RAMMin, nil, nil, l.VRAMMin); err != nil {
		return Response{}, err
	}
	useGPU := l.PreferGPU && l.gpuUsable()
	if strings.TrimSpace(l.ModelPath) == "" {
		return Response{Backend: l.Backend, UsedGPU: useGPU}, ErrNoModel
	}
	bin := resolveBin(l.Bin)
	if bin == "" {
		return Response{Backend: l.Backend, UsedGPU: useGPU}, ErrNoBinary
	}
	argv := cliArgs(bin, l.ModelPath, req.Prompt, useGPU)
	out, err := l.invoke(ctx, argv)
	if err != nil && useGPU && !needGPU && ctx.Err() == nil {
		argv = cliArgs(bin, l.ModelPath, req.Prompt, false)
		out, err = l.invoke(ctx, argv)
		useGPU = false
	}
	if err != nil {
		return Response{Backend: l.Backend, UsedGPU: useGPU}, err
	}
	return Response{
		Text:    cleanCompletion(out),
		Backend: l.Backend,
		UsedGPU: useGPU,
	}, nil
}

// gpuUsable is true only when a device is present and VRAM is known.
// Product jails often have /dev/nvidia0 (gpu_present) with
// gpu_vram_free_bytes=null — llama-cli -ngl 99 fails there. CPU (-ngl 0)
// must still run. Missing GPU does not block CPU jobs.
func (l Local) gpuUsable() bool {
	if !l.GPUPresent {
		return false
	}
	return l.Headroom.GPUVRAMFreeBytes != nil
}

func resolveBin(explicit string) string {
	if strings.TrimSpace(explicit) != "" {
		return explicit
	}
	for _, name := range []string{"llama-completion", "llama-cli", "llama.cpp"} {
		if p, err := lookPath(name); err == nil && p != "" {
			return p
		}
	}
	return ""
}

// wellKnownEmbedBins are FreeBSD prefixes for llama-embedding when PATH
// is empty. llm.local.bin stays the Complete binary.
var wellKnownEmbedBins = []string{"/usr/local/bin/llama-embedding"}

func isEmbedBin(bin string) bool {
	base := strings.ToLower(filepath.Base(strings.TrimSpace(bin)))
	return base == "llama-embedding" || strings.HasPrefix(base, "llama-embedding.")
}

func resolveNamed(name string) string {
	if strings.TrimSpace(name) == "" {
		return ""
	}
	p, err := lookPath(name)
	if err != nil || p == "" {
		return ""
	}
	return p
}

// resolveEmbedBin finds llama-embedding. completeBin is llm.local.bin
// (llama-completion / llama-cli) and is not reused for Embed.
// Jail config keeps one bin for Complete; operators do not hand-edit
// JSON when llama-embedding is on PATH or beside that bin.
func resolveEmbedBin(completeBin string) string {
	completeBin = strings.TrimSpace(completeBin)
	if isEmbedBin(completeBin) {
		return completeBin
	}
	if p := resolveNamed("llama-embedding"); p != "" {
		return p
	}
	if completeBin != "" {
		sib := filepath.Join(filepath.Dir(completeBin), "llama-embedding")
		if p := resolveNamed(sib); p != "" {
			return p
		}
	}
	for _, p := range wellKnownEmbedBins {
		if got := resolveNamed(p); got != "" {
			return got
		}
	}
	return ""
}

func needsSingleTurn(bin string) bool {
	base := strings.ToLower(filepath.Base(strings.TrimSpace(bin)))
	return base == "llama-cli" || strings.HasPrefix(base, "llama-cli.")
}

func cliArgs(bin, model, prompt string, useGPU bool) []string {
	ngl := "0"
	if useGPU {
		ngl = "99"
	}
	argv := []string{
		bin,
		"-m", model,
		"-p", prompt,
		"--no-display-prompt",
		"-n", "256",
		"-ngl", ngl,
	}
	if needsSingleTurn(bin) {
		argv = append(argv, "--single-turn", "--simple-io")
	}
	return argv
}

// cleanCompletion drops trailing llama-cli chat leftovers so TTY
// consult does not show "> EOF by user" / "Exiting...".
func cleanCompletion(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	for len(lines) > 0 {
		last := strings.TrimSpace(lines[len(lines)-1])
		if last == "" || leftoverLine(last) {
			lines = lines[:len(lines)-1]
			continue
		}
		break
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func leftoverLine(s string) bool {
	s = strings.TrimSpace(s)
	switch s {
	case ">", "> EOF by user", "EOF by user", "Exiting...", "Exiting.":
		return true
	}
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "> eof") {
		return true
	}
	if strings.HasPrefix(lower, "exiting") {
		return true
	}
	return false
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
