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
	bin := writeJailEmbeddingStub(t, dir, capture, "[1.0, 0.0, 0.0]")
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
	if strings.Contains(argv, "--embedding") || strings.Contains(argv, "--no-display-prompt") {
		t.Fatalf("llama-embedding 9426 rejects chat flags: %s", argv)
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
	bin := writeJailEmbeddingStub(t, dir, capture, "[1.0, 0.0]")
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
	if nglIs(got, "99") {
		t.Fatalf("null VRAM must not pass -ngl 99: %s", got)
	}
	if !nglIs(got, "0") {
		t.Fatalf("null VRAM must pass -ngl 0: %s", got)
	}
}

func TestLocal_EmbedPreferGPU(t *testing.T) {
	dir := t.TempDir()
	capture := filepath.Join(dir, "argv.txt")
	model := filepath.Join(dir, "fake-embed.gguf")
	if err := os.WriteFile(model, []byte("not-a-real-gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := writeJailEmbeddingStub(t, dir, capture, "1.0 0.0 0.0")
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
	if !nglIs(got, "99") {
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
	bin := writeJailEmbeddingStub(t, dir, "", "[1.0]")
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
	bin := writeFakeLlamaNamed(t, "llama-embedding", "", "")
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
	bin := writeFakeLlamaNamed(t, "llama-embedding", "", "loading...\nembedding: [0.5, 0.25]\ndone")
	l := llm.Local{Bin: bin, EmbedModelPath: model, Headroom: ampleRAM()}
	vec, err := l.Embed(context.Background(), "x")
	if err != nil {
		t.Fatal(err)
	}
	if len(vec) != 2 || vec[0] != 0.5 {
		t.Fatalf("vec=%v", vec)
	}
}

func TestLocal_EmbedLlamaEmbeddingRejectsChatFlags(t *testing.T) {
	dir := t.TempDir()
	capture := filepath.Join(dir, "argv.txt")
	model := filepath.Join(dir, "fake-embed.gguf")
	if err := os.WriteFile(model, []byte("not-a-real-gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := writeJailEmbeddingStub(t, dir, capture, "[1.0, 0.0, 0.0]")
	l := llm.Local{
		Backend:        "llama.cpp",
		Bin:            bin,
		EmbedModelPath: model,
		Headroom:       ampleRAM(),
	}
	vec, err := l.Embed(context.Background(), "root filesystem is read-only")
	if err != nil {
		t.Fatalf("llama-embedding 9426 rejects --embedding and --no-display-prompt: %v", err)
	}
	if len(vec) != 3 || vec[0] != 1 {
		t.Fatalf("vec=%v", vec)
	}
	got, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	argv := string(got)
	if strings.Contains(argv, "--no-display-prompt") {
		t.Fatalf("must not pass --no-display-prompt to llama-embedding: %s", argv)
	}
	if strings.Contains(argv, "--embedding") {
		t.Fatalf("llama-embedding does not want --embedding: %s", argv)
	}
	if !argvFlagIs(argv, "--pooling", "mean") {
		t.Fatalf("must pass --pooling mean: %s", argv)
	}
	if !argvFlagIs(argv, "--embd-separator", "<#sep#>") {
		t.Fatalf("newline embd-separator splits playbooks; must pass <#sep#>: %s", argv)
	}
	if !argvFlagIs(argv, "--embd-output-format", "array") {
		t.Fatalf("must pass --embd-output-format array: %s", argv)
	}
}

func TestLocal_EmbedPrefersLlamaEmbeddingNotCompletion(t *testing.T) {
	root := t.TempDir()
	compCap := filepath.Join(root, "completion-argv.txt")
	embCap := filepath.Join(root, "embedding-argv.txt")
	model := filepath.Join(root, "nomic-embed-text-v1.5.Q8_0.gguf")
	if err := os.WriteFile(model, []byte("not-a-real-gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	compDir := filepath.Join(root, "complete")
	embDir := filepath.Join(root, "embed")
	comp := writeFakeAt(t, compDir, "llama-completion", compCap, "SHOULD-NOT-RUN")
	_ = writeJailEmbeddingStub(t, embDir, embCap, "[0.25, 0.5]")
	t.Setenv("PATH", embDir)
	l := llm.Local{
		Backend:        "llama.cpp",
		Bin:            comp,
		EmbedModelPath: model,
		Headroom:       ampleRAM(),
	}
	vec, err := l.Embed(context.Background(), "root filesystem is read-only")
	if err != nil {
		t.Fatal(err)
	}
	if len(vec) != 2 || vec[0] != 0.25 {
		t.Fatalf("vec=%v", vec)
	}
	if b, err := os.ReadFile(compCap); err == nil && len(b) > 0 {
		t.Fatalf("llama-completion must not be used for Embed when llama-embedding exists: %s", b)
	}
	got, err := os.ReadFile(embCap)
	if err != nil {
		t.Fatal(err)
	}
	argv := string(got)
	if filepath.Base(strings.TrimSpace(strings.Split(argv, "\n")[0])) == "llama-completion" {
		t.Fatalf("Embed argv started with llama-completion: %s", argv)
	}
	if strings.Contains(argv, "--no-display-prompt") || strings.Contains(argv, "--embedding") {
		t.Fatalf("chat flags on llama-embedding: %s", argv)
	}
}

func TestLocal_EmbedSiblingLlamaEmbeddingWhenPATHEmpty(t *testing.T) {
	root := t.TempDir()
	compCap := filepath.Join(root, "completion-argv.txt")
	embCap := filepath.Join(root, "embedding-argv.txt")
	model := filepath.Join(root, "fake-embed.gguf")
	if err := os.WriteFile(model, []byte("not-a-real-gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	binDir := filepath.Join(root, "bin")
	comp := writeFakeAt(t, binDir, "llama-completion", compCap, "SHOULD-NOT-RUN")
	_ = writeJailEmbeddingStub(t, binDir, embCap, "[1.0]")
	t.Setenv("PATH", t.TempDir())
	l := llm.Local{
		Bin:            comp,
		EmbedModelPath: model,
		Headroom:       ampleRAM(),
	}
	vec, err := l.Embed(context.Background(), "zfs")
	if err != nil {
		t.Fatal(err)
	}
	if len(vec) != 1 || vec[0] != 1 {
		t.Fatalf("vec=%v", vec)
	}
	if b, err := os.ReadFile(compCap); err == nil && len(b) > 0 {
		t.Fatalf("sibling llama-embedding must win over llama-completion: %s", b)
	}
}

func TestLocal_EmbedDoesNotReuseLlamaCLI(t *testing.T) {
	dir := t.TempDir()
	capture := filepath.Join(dir, "argv.txt")
	model := filepath.Join(dir, "fake-embed.gguf")
	if err := os.WriteFile(model, []byte("not-a-real-gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := writeFakeLlama(t, capture, "[1.0]")
	t.Setenv("PATH", t.TempDir())
	l := llm.Local{Bin: bin, EmbedModelPath: model, Headroom: ampleRAM()}
	_, err := l.Embed(context.Background(), "zfs")
	if !errors.Is(err, llm.ErrNoBinary) {
		t.Fatalf("llama-cli 9426 cannot embed; missing llama-embedding must skip: %v", err)
	}
	if b, err := os.ReadFile(capture); err == nil && len(b) > 0 {
		t.Fatalf("must not exec llama-cli for Embed: %s", b)
	}
}

func TestLocal_EmbedParsesSmallArray(t *testing.T) {
	dir := t.TempDir()
	model := filepath.Join(dir, "fake-embed.gguf")
	if err := os.WriteFile(model, []byte("not-a-real-gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := writeJailEmbeddingStub(t, dir, "", "[0.1, 0.2, 0.3, 0.4]")
	l := llm.Local{Bin: bin, EmbedModelPath: model, Headroom: ampleRAM()}
	vec, err := l.Embed(context.Background(), "zfs")
	if err != nil {
		t.Fatal(err)
	}
	if len(vec) != 4 || vec[3] != float32(0.4) {
		t.Fatalf("parsed %v", vec)
	}
}

func TestLocal_EmbedParsesDim768Array(t *testing.T) {
	dir := t.TempDir()
	model := filepath.Join(dir, "fake-embed.gguf")
	if err := os.WriteFile(model, []byte("not-a-real-gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	raw := make([]byte, 0, 8*768)
	raw = append(raw, '[')
	for i := 0; i < 768; i++ {
		if i > 0 {
			raw = append(raw, ',', ' ')
		}
		if i == 0 {
			raw = append(raw, '1')
		} else {
			raw = append(raw, '0')
		}
	}
	raw = append(raw, ']')
	l := llm.Local{
		Bin:            filepath.Join(dir, "llama-embedding"),
		EmbedModelPath: model,
		Headroom:       ampleRAM(),
		Run: func(_ context.Context, argv []string) (string, error) {
			if filepath.Base(argv[0]) != "llama-embedding" {
				t.Fatalf("argv[0]=%q", argv[0])
			}
			for _, a := range argv {
				if a == "--no-display-prompt" || a == "--embedding" {
					t.Fatalf("chat flag %q", a)
				}
			}
			return string(raw), nil
		},
	}
	vec, err := l.Embed(context.Background(), "root filesystem is read-only")
	if err != nil {
		t.Fatal(err)
	}
	if len(vec) != 768 || vec[0] != 1 {
		t.Fatalf("nomic dim must parse as 768, got %d", len(vec))
	}
}

func TestLocal_CompleteStillUsesConfiguredBin(t *testing.T) {
	root := t.TempDir()
	compCap := filepath.Join(root, "completion-argv.txt")
	embCap := filepath.Join(root, "embedding-argv.txt")
	model := filepath.Join(root, "fake.gguf")
	if err := os.WriteFile(model, []byte("not-a-real-gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	comp := writeFakeAt(t, filepath.Join(root, "complete"), "llama-completion", compCap, "from-completion")
	_ = writeJailEmbeddingStub(t, filepath.Join(root, "embed"), embCap, "[1.0]")
	t.Setenv("PATH", filepath.Join(root, "embed"))
	l := llm.Local{Bin: comp, ModelPath: model, Headroom: ampleRAM()}
	resp, err := l.Complete(context.Background(), llm.Request{Prompt: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "from-completion" {
		t.Fatalf("Complete must keep llm.local.bin: %q", resp.Text)
	}
	if b, err := os.ReadFile(embCap); err == nil && len(b) > 0 {
		t.Fatalf("Complete must not exec llama-embedding: %s", b)
	}
}

func argvFlagIs(capture, flag, val string) bool {
	lines := strings.Split(capture, "\n")
	for i, line := range lines {
		if line == flag && i+1 < len(lines) && lines[i+1] == val {
			return true
		}
	}
	return false
}
