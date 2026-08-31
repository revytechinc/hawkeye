// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package llm_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/revytechinc/hawkeye/internal/headroom"
	"github.com/revytechinc/hawkeye/internal/llm"
)

func TestLocal_EmbedNoModelSkips(t *testing.T) {
	l := llm.Local{Backend: "llama.cpp", Headroom: ampleRAM()}
	_, err := l.Embed(context.Background(), "zfs readonly")
	if !errors.Is(err, llm.ErrNoModel) {
		t.Fatalf("err=%v", err)
	}
}

func TestLocal_EmbedLowRAMSkips(t *testing.T) {
	ram := int64(1 << 40)
	l := llm.Local{
		Backend:        "llama.cpp",
		EmbedModelPath: "/models/fake-embed.gguf",
		Bin:            "/nonexistent/llama-cli",
		Headroom:       headroom.Snapshot{RAMFreeBytes: 1},
		RAMMin:         &ram,
	}
	_, err := l.Embed(context.Background(), "zfs readonly")
	if err == nil {
		t.Fatal("expected headroom refusal")
	}
	if errors.Is(err, llm.ErrNoModel) {
		t.Fatalf("wrong error: %v", err)
	}
}

func TestLocal_EmbedFakeBinaryGPUThenCPU(t *testing.T) {
	dir := t.TempDir()
	capture := filepath.Join(dir, "argv.txt")
	model := filepath.Join(dir, "fake-embed.gguf")
	if err := os.WriteFile(model, []byte("not-a-real-gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := writeFakeLlama(t, capture, "[1.0, 0.0, 0.0]")
	l := llm.Local{
		Backend:        "llama.cpp",
		Bin:            bin,
		EmbedModelPath: model,
		PreferGPU:      true,
		GPUPresent:     false,
		Headroom:       ampleRAM(),
	}
	vec, err := l.Embed(context.Background(), "password=fake-password-for-tests-only remount")
	if err != nil {
		t.Fatal(err)
	}
	if len(vec) != 3 || vec[0] != 1 {
		t.Fatalf("vec=%v", vec)
	}
	got, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	argv := string(got)
	if strings.Contains(argv, "fake-password-for-tests-only") {
		t.Fatal("secret leaked into embed argv")
	}
	if !strings.Contains(argv, "--embedding") {
		t.Fatalf("must request embeddings, not chat: %s", argv)
	}
	if !strings.Contains(argv, "-ngl") || !strings.Contains(argv, "\n0\n") {
		t.Fatalf("missing GPU must stay CPU: %s", argv)
	}
	if l.Model() != model {
		t.Fatalf("model name %q", l.Model())
	}
}

func TestLocal_EmbedNullVRAMUsesCPU(t *testing.T) {
	dir := t.TempDir()
	capture := filepath.Join(dir, "argv.txt")
	model := filepath.Join(dir, "fake-embed.gguf")
	if err := os.WriteFile(model, []byte("not-a-real-gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := writeFakeLlama(t, capture, "[1.0, 0.0]")
	l := llm.Local{
		Backend:        "llama.cpp",
		Bin:            bin,
		EmbedModelPath: model,
		PreferGPU:      true,
		GPUPresent:     true,
		Headroom:       headroom.Snapshot{RAMFreeBytes: 1 << 30, GPUPresent: true, GPUVRAMFreeBytes: nil},
	}
	vec, err := l.Embed(context.Background(), "zfs")
	if err != nil || len(vec) != 2 {
		t.Fatalf("null VRAM must embed on CPU: %v %v", vec, err)
	}
	got, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "99") {
		t.Fatalf("null VRAM must not pass -ngl 99: %s", got)
	}
}

func TestLocal_EmbedPreferGPU(t *testing.T) {
	dir := t.TempDir()
	capture := filepath.Join(dir, "argv.txt")
	model := filepath.Join(dir, "fake-embed.gguf")
	if err := os.WriteFile(model, []byte("not-a-real-gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := writeFakeLlama(t, capture, "1.0 0.0 0.0")
	l := llm.Local{
		Backend:        "llama.cpp",
		Bin:            bin,
		EmbedModelPath: model,
		PreferGPU:      true,
		GPUPresent:     true,
		Headroom:       headroom.Snapshot{RAMFreeBytes: 1 << 30, GPUPresent: true, GPUVRAMFreeBytes: vram(1 << 30)},
	}
	vec, err := l.Embed(context.Background(), "zfs")
	if err != nil {
		t.Fatal(err)
	}
	if len(vec) != 3 {
		t.Fatalf("parsed %v", vec)
	}
	got, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "99") {
		t.Fatalf("GPU then CPU must pass -ngl 99: %s", got)
	}
}

func TestLocal_EmbedNoBinary(t *testing.T) {
	l := llm.Local{
		Backend:        "llama.cpp",
		EmbedModelPath: "/models/fake-embed.gguf",
		Headroom:       ampleRAM(),
	}
	t.Setenv("PATH", t.TempDir())
	_, err := l.Embed(context.Background(), "zfs")
	if !errors.Is(err, llm.ErrNoBinary) {
		t.Fatalf("err=%v", err)
	}
}

func TestLocal_EmbedRequireGPUWhenPresent(t *testing.T) {
	dir := t.TempDir()
	model := filepath.Join(dir, "fake-embed.gguf")
	if err := os.WriteFile(model, []byte("not-a-real-gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := writeFakeLlama(t, "", "[1.0]")
	l := llm.Local{
		Bin:            bin,
		EmbedModelPath: model,
		RequireGPU:     true,
		PreferGPU:      true,
		GPUPresent:     true,
		Headroom:       headroom.Snapshot{RAMFreeBytes: 1 << 30, GPUPresent: true, GPUVRAMFreeBytes: vram(1 << 30)},
	}
	vec, err := l.Embed(context.Background(), "zfs")
	if err != nil || len(vec) != 1 {
		t.Fatalf("%v %v", vec, err)
	}
}

func TestLocal_EmbedRequireGPU(t *testing.T) {
	l := llm.Local{
		RequireGPU:     true,
		GPUPresent:     false,
		EmbedModelPath: "/models/fake-embed.gguf",
		Headroom:       ampleRAM(),
	}
	_, err := l.Embed(nil, "zfs")
	if !errors.Is(err, llm.ErrGPURequired) {
		t.Fatalf("err=%v", err)
	}
}

func TestLocal_ModelFallsBackToChatPath(t *testing.T) {
	l := llm.Local{ModelPath: "/models/chat.gguf"}
	if l.Model() != "/models/chat.gguf" {
		t.Fatal(l.Model())
	}
}

func TestLocal_EmbedEmptyOutput(t *testing.T) {
	dir := t.TempDir()
	model := filepath.Join(dir, "fake-embed.gguf")
	if err := os.WriteFile(model, []byte("not-a-real-gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := writeFakeLlama(t, "", "")
	l := llm.Local{Bin: bin, EmbedModelPath: model, Headroom: ampleRAM()}
	if _, err := l.Embed(context.Background(), "x"); err == nil {
		t.Fatal("empty embedding output")
	}
	l.Run = func(context.Context, []string) (string, error) { return "no floats here", nil }
	if _, err := l.Embed(context.Background(), "x"); err == nil {
		t.Fatal("unparsed embedding")
	}
	l.Run = func(context.Context, []string) (string, error) { return "", errors.New("boom") }
	if _, err := l.Embed(context.Background(), "x"); err == nil {
		t.Fatal("invoke error")
	}
}

func TestParseEmbedding_SurroundingText(t *testing.T) {
	dir := t.TempDir()
	model := filepath.Join(dir, "fake-embed.gguf")
	if err := os.WriteFile(model, []byte("not-a-real-gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := writeFakeLlama(t, "", "loading...\nembedding: [0.5, 0.25]\ndone")
	l := llm.Local{Bin: bin, EmbedModelPath: model, Headroom: ampleRAM()}
	vec, err := l.Embed(context.Background(), "x")
	if err != nil {
		t.Fatal(err)
	}
	if len(vec) != 2 || vec[0] != 0.5 {
		t.Fatalf("vec=%v", vec)
	}
}
