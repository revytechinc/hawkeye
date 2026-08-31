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
	"time"

	"github.com/revytechinc/hawkeye/internal/headroom"
	"github.com/revytechinc/hawkeye/internal/llm"
)

func ampleRAM() headroom.Snapshot {
	return headroom.Snapshot{RAMFreeBytes: 1 << 30, GPUPresent: false}
}

func writeFakeLlama(t *testing.T, capture, canned string) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "llama-cli")
	script := "#!/bin/sh\n"
	if capture != "" {
		script += "for a in \"$@\"; do printf '%s\\n' \"$a\" >> \"" + capture + "\"; done\n"
	}
	script += "printf '%s\\n' '" + canned + "'\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin
}

func writeFakeLlamaNamed(t *testing.T, name, capture, canned string) string {
	t.Helper()
	return writeFakeAt(t, t.TempDir(), name, capture, canned)
}

func writeFakeAt(t *testing.T, dir, name, capture, canned string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, name)
	script := "#!/bin/sh\n"
	if capture != "" {
		script += "for a in \"$@\"; do printf '%s\\n' \"$a\" >> \"" + capture + "\"; done\n"
	}
	script += "printf '%s\\n' '" + canned + "'\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin
}

// writeJailEmbeddingStub mimics llama-embedding 9426: rejects --embedding
// and --no-display-prompt, prints a FLOAT32 array otherwise.
func writeJailEmbeddingStub(t *testing.T, dir, capture, canned string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "llama-embedding")
	script := "#!/bin/sh\n"
	if capture != "" {
		script += "for a in \"$@\"; do printf '%s\\n' \"$a\" >> \"" + capture + "\"; done\n"
	}
	script += "for a in \"$@\"; do\n"
	script += "  if [ \"$a\" = \"--embedding\" ] || [ \"$a\" = \"--no-display-prompt\" ]; then\n"
	script += "    echo 'error: invalid argument: '" + "\"$a\" >&2\n"
	script += "    exit 1\n"
	script += "  fi\n"
	script += "done\n"
	script += "printf '%s\\n' '" + canned + "'\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin
}

// writeChatOnlyLlamaCLI hangs like llama-cli 9426 conversation mode unless
// --single-turn is present. A hang is a test failure (panic session must not wait).
func writeChatOnlyLlamaCLI(t *testing.T, canned string) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "llama-cli")
	script := "#!/bin/sh\nsingle=0\nfor a in \"$@\"; do\n  if [ \"$a\" = \"--single-turn\" ]; then single=1; fi\ndone\nif [ \"$single\" -eq 0 ]; then exec cat >/dev/null; fi\nprintf '%s\\n' '" + canned + "'\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin
}

func vram(n int64) *int64 { return &n }

func nglIs(capture []byte, n string) bool {
	lines := strings.Split(string(capture), "\n")
	for i, line := range lines {
		if line == "-ngl" && i+1 < len(lines) && lines[i+1] == n {
			return true
		}
	}
	return false
}

func TestLocal_LlamaCLIGetsSingleTurn(t *testing.T) {
	dir := t.TempDir()
	capture := filepath.Join(dir, "argv.txt")
	model := filepath.Join(dir, "fake.gguf")
	if err := os.WriteFile(model, []byte("not-a-real-gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := writeFakeLlamaNamed(t, "llama-cli", capture, "one-shot")
	l := llm.Local{Backend: "llama.cpp", Bin: bin, ModelPath: model, Headroom: ampleRAM()}
	resp, err := l.Complete(context.Background(), llm.Request{Prompt: "root is read-only"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "one-shot" {
		t.Fatalf("text=%q", resp.Text)
	}
	got, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	argv := string(got)
	if !strings.Contains(argv, "--single-turn") {
		t.Fatalf("llama-cli 9426 is conversation-only; must pass --single-turn: %s", argv)
	}
	if !strings.Contains(argv, "--simple-io") {
		t.Fatalf("llama-cli one-shot must pass --simple-io: %s", argv)
	}
}

func TestLocal_LlamaCompletionIsOneShotWithoutSingleTurn(t *testing.T) {
	dir := t.TempDir()
	capture := filepath.Join(dir, "argv.txt")
	model := filepath.Join(dir, "fake.gguf")
	if err := os.WriteFile(model, []byte("not-a-real-gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := writeFakeLlamaNamed(t, "llama-completion", capture, "from-completion")
	l := llm.Local{Backend: "llama.cpp", Bin: bin, ModelPath: model, Headroom: ampleRAM()}
	resp, err := l.Complete(context.Background(), llm.Request{Prompt: "root is read-only"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "from-completion" {
		t.Fatalf("text=%q", resp.Text)
	}
	got, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	argv := string(got)
	if strings.Contains(argv, "--single-turn") || strings.Contains(argv, "--simple-io") {
		t.Fatalf("llama-completion already one-shots; do not add chat flags: %s", argv)
	}
}

func TestLocal_LookPathPrefersLlamaCompletion(t *testing.T) {
	dir := t.TempDir()
	model := filepath.Join(dir, "fake.gguf")
	if err := os.WriteFile(model, []byte("not-a-real-gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	cli := filepath.Join(dir, "llama-cli")
	comp := filepath.Join(dir, "llama-completion")
	if err := os.WriteFile(cli, []byte("#!/bin/sh\nprintf 'from-cli\\n'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(comp, []byte("#!/bin/sh\nprintf 'from-completion\\n'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	l := llm.Local{Backend: "llama.cpp", ModelPath: model, Headroom: ampleRAM()}
	resp, err := l.Complete(context.Background(), llm.Request{Prompt: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "from-completion" {
		t.Fatalf("PATH must prefer llama-completion over llama-cli: %q", resp.Text)
	}
}

func TestLocal_LlamaCLIWithoutSingleTurnIsHangFailure(t *testing.T) {
	model := filepath.Join(t.TempDir(), "fake.gguf")
	if err := os.WriteFile(model, []byte("not-a-real-gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := writeChatOnlyLlamaCLI(t, "one-shot complete")
	ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel()
	l := llm.Local{Backend: "llama.cpp", Bin: bin, ModelPath: model, Headroom: ampleRAM()}
	resp, err := l.Complete(ctx, llm.Request{Prompt: "root is read-only"})
	if err != nil {
		t.Fatalf("conversation-mode hang is a product failure; --single-turn must one-shot: %v", err)
	}
	if resp.Text != "one-shot complete" {
		t.Fatalf("text=%q", resp.Text)
	}
}

func TestLocal_StripsLlamaChatLeftovers(t *testing.T) {
	l := llm.Local{
		Backend:   "llama.cpp",
		Bin:       "/usr/local/bin/llama-cli",
		ModelPath: "/models/fake.gguf",
		Headroom:  ampleRAM(),
		Run: func(_ context.Context, argv []string) (string, error) {
			return "remount the ZFS root read-write\n> EOF by user\nExiting...\n", nil
		},
	}
	resp, err := l.Complete(context.Background(), llm.Request{Prompt: "root is read-only"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(resp.Text, "EOF by user") || strings.Contains(resp.Text, "Exiting") || strings.Contains(resp.Text, ">") {
		t.Fatalf("TTY must not show llama chat leftovers: %q", resp.Text)
	}
	if resp.Text != "remount the ZFS root read-write" {
		t.Fatalf("text=%q", resp.Text)
	}
}

func TestLocal_JailLikeGPUNullVRAMUsesCPUNotSkip(t *testing.T) {
	// Product jail: gpu_present (nvidia0) but gpu_vram_free_bytes=null.
	// prefer_gpu=true must not pass -ngl 99 (llama-cli fails; consult skips).
	dir := t.TempDir()
	capture := filepath.Join(dir, "argv.txt")
	model := filepath.Join(dir, "fake.gguf")
	if err := os.WriteFile(model, []byte("not-a-real-gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := writeFakeLlama(t, capture, "cpu complete: remount the ZFS root")
	ram := int64(256 * 1024 * 1024)
	l := llm.Local{
		Backend:    "llama.cpp",
		Bin:        bin,
		ModelPath:  model,
		PreferGPU:  true,
		RequireGPU: false,
		GPUPresent: true,
		Headroom: headroom.Snapshot{
			RAMFreeBytes:     129 << 30,
			RAMTotalBytes:    129 << 30,
			GPUPresent:       true,
			GPUVRAMFreeBytes: nil,
		},
		RAMMin:  &ram,
		VRAMMin: nil,
	}
	resp, err := l.Complete(context.Background(), llm.Request{Prompt: "root is read-only", NeedGPU: false, NeedRAM: true})
	if err != nil {
		t.Fatalf("null VRAM must fall back to CPU, not skip: %v", err)
	}
	if resp.UsedGPU {
		t.Fatal("null VRAM is not usable GPU")
	}
	if resp.Text != "cpu complete: remount the ZFS root" {
		t.Fatalf("text=%q", resp.Text)
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

func TestLocal_GPUInvokeFailFallsBackToCPU(t *testing.T) {
	model := filepath.Join(t.TempDir(), "fake.gguf")
	if err := os.WriteFile(model, []byte("not-a-real-gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	var ngl []string
	l := llm.Local{
		Backend:    "llama.cpp",
		Bin:        "/usr/local/bin/llama-cli",
		ModelPath:  model,
		PreferGPU:  true,
		RequireGPU: false,
		GPUPresent: true,
		Headroom: headroom.Snapshot{
			RAMFreeBytes:     1 << 30,
			GPUPresent:       true,
			GPUVRAMFreeBytes: vram(1 << 30),
		},
		Run: func(_ context.Context, argv []string) (string, error) {
			for i, a := range argv {
				if a == "-ngl" && i+1 < len(argv) {
					ngl = append(ngl, argv[i+1])
				}
			}
			if len(ngl) > 0 && ngl[len(ngl)-1] == "99" {
				return "", errors.New("cuda error: no usable VRAM")
			}
			return "cpu fallback complete", nil
		},
	}
	resp, err := l.Complete(context.Background(), llm.Request{Prompt: "root is read-only", NeedGPU: false, NeedRAM: true})
	if err != nil {
		t.Fatalf("GPU fail must fall back to CPU, not skip: %v", err)
	}
	if resp.UsedGPU {
		t.Fatal("fallback must not mark UsedGPU")
	}
	if resp.Text != "cpu fallback complete" {
		t.Fatalf("text=%q", resp.Text)
	}
	if len(ngl) < 2 || ngl[0] != "99" || ngl[1] != "0" {
		t.Fatalf("must try -ngl 99 then 0: %v", ngl)
	}
}

func TestLocal_JailLikeGPUNullVRAMNoModelSkips(t *testing.T) {
	// Live product jail: /dev/nvidia0 present, gpu_vram_free_bytes null,
	// llama.cpp backend, empty model_path, prefer_gpu, no llama-cli, no GGUF.
	ram := int64(256 * 1024 * 1024)
	l := llm.Local{
		Backend:    "llama.cpp",
		PreferGPU:  true,
		RequireGPU: false,
		GPUPresent: true,
		Headroom: headroom.Snapshot{
			RAMFreeBytes:     129 << 30,
			RAMTotalBytes:    129 << 30,
			GPUPresent:       true,
			GPUVRAMFreeBytes: nil,
		},
		RAMMin:  &ram,
		VRAMMin: nil,
	}
	resp, err := l.Complete(context.Background(), llm.Request{Prompt: "zpool degraded", NeedGPU: false, NeedRAM: true})
	if !errors.Is(err, llm.ErrNoModel) {
		t.Fatalf("no model must skip, not block on null VRAM: %v", err)
	}
	if strings.Contains(resp.Text, "skeleton") || strings.Contains(resp.Text, "llm skipped") {
		t.Fatalf("no-model must not invent guts: %q", resp.Text)
	}
}

func TestLocal_NoModelSkips(t *testing.T) {
	ram := int64(1)
	l := llm.Local{
		Backend:  "llama.cpp",
		Headroom: ampleRAM(),
		RAMMin:   &ram,
	}
	resp, err := l.Complete(context.Background(), llm.Request{Prompt: "hello", NeedRAM: true})
	if !errors.Is(err, llm.ErrNoModel) {
		t.Fatalf("err=%v", err)
	}
	if strings.Contains(resp.Text, "skeleton") {
		t.Fatalf("no-model path must not invent skeleton text: %q", resp.Text)
	}
}

func TestLocal_FakeBinaryCapturesPromptAndReturnsCanned(t *testing.T) {
	dir := t.TempDir()
	capture := filepath.Join(dir, "argv.txt")
	model := filepath.Join(dir, "fake.gguf")
	if err := os.WriteFile(model, []byte("not-a-real-gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	canned := "canned local completion: check zpool status"
	bin := writeFakeLlama(t, capture, canned)
	ram := int64(1)
	l := llm.Local{
		Backend:   "llama.cpp",
		Bin:       bin,
		ModelPath: model,
		PreferGPU: true,
		Headroom:  ampleRAM(),
		RAMMin:    &ram,
	}
	resp, err := l.Complete(context.Background(), llm.Request{
		Prompt:  "password=fake-password-for-tests-only zpool degraded",
		NeedRAM: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != canned {
		t.Fatalf("text=%q want canned", resp.Text)
	}
	if strings.Contains(resp.Text, "skeleton") {
		t.Fatal("must invoke the binary, not the skeleton")
	}
	if strings.Contains(resp.Text, "fake-password-for-tests-only") {
		t.Fatal("secret leaked into LLM response")
	}
	got, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	argv := string(got)
	if strings.Contains(argv, "fake-password-for-tests-only") {
		t.Fatal("secret leaked into llama-cli argv")
	}
	if !strings.Contains(argv, "-m") || !strings.Contains(argv, model) {
		t.Fatalf("binary must receive model path: %s", argv)
	}
	if !strings.Contains(argv, "-p") {
		t.Fatalf("binary must receive prompt flag: %s", argv)
	}
	if !strings.Contains(argv, "-ngl") {
		t.Fatalf("GPU-then-CPU must pass -ngl: %s", argv)
	}
}

func TestLocal_MissingGPUDoesNotBlockCPUJob(t *testing.T) {
	dir := t.TempDir()
	capture := filepath.Join(dir, "argv.txt")
	model := filepath.Join(dir, "fake.gguf")
	if err := os.WriteFile(model, []byte("not-a-real-gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := writeFakeLlama(t, capture, "cpu completion")
	ram := int64(1)
	l := llm.Local{
		Backend:    "llama.cpp",
		Bin:        bin,
		PreferGPU:  true,
		RequireGPU: false,
		GPUPresent: false,
		ModelPath:  model,
		Headroom:   ampleRAM(),
		RAMMin:     &ram,
	}
	resp, err := l.Complete(context.Background(), llm.Request{Prompt: "hello", NeedGPU: false, NeedRAM: true})
	if err != nil {
		t.Fatalf("CPU job blocked: %v", err)
	}
	if resp.UsedGPU {
		t.Fatal("missing GPU must not mark UsedGPU")
	}
	got, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	if !nglIs(got, "0") {
		t.Fatalf("CPU fallback must pass -ngl 0: %s", got)
	}
}

func TestLocal_RequireGPUFailsWithoutDevice(t *testing.T) {
	l := llm.Local{RequireGPU: true, GPUPresent: false, Headroom: ampleRAM(), ModelPath: "/models/fake.gguf"}
	_, err := l.Complete(context.Background(), llm.Request{NeedGPU: true})
	if !errors.Is(err, llm.ErrGPURequired) {
		t.Fatalf("err=%v", err)
	}
}

func TestLocal_RedactsPrompt(t *testing.T) {
	dir := t.TempDir()
	capture := filepath.Join(dir, "argv.txt")
	model := filepath.Join(dir, "fake.gguf")
	if err := os.WriteFile(model, []byte("not-a-real-gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := writeFakeLlama(t, capture, "ok")
	l := llm.Local{
		Backend:   "llama.cpp",
		Bin:       bin,
		ModelPath: model,
		Headroom:  ampleRAM(),
	}
	resp, err := l.Complete(context.Background(), llm.Request{Prompt: "password=fake-password-for-tests-only"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(resp.Text, "fake-password-for-tests-only") {
		t.Fatal("secret leaked into LLM response path")
	}
	got, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "fake-password-for-tests-only") {
		t.Fatal("secret leaked into backend argv")
	}
}

func TestLocal_PreferGPUPassesLayersWhenPresent(t *testing.T) {
	dir := t.TempDir()
	capture := filepath.Join(dir, "argv.txt")
	model := filepath.Join(dir, "fake.gguf")
	if err := os.WriteFile(model, []byte("not-a-real-gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := writeFakeLlama(t, capture, "gpu completion")
	l := llm.Local{
		Backend:    "llama.cpp",
		Bin:        bin,
		ModelPath:  model,
		PreferGPU:  true,
		GPUPresent: true,
		Headroom:   headroom.Snapshot{RAMFreeBytes: 1 << 30, GPUPresent: true, GPUVRAMFreeBytes: vram(1 << 30)},
	}
	resp, err := l.Complete(context.Background(), llm.Request{Prompt: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.UsedGPU {
		t.Fatal("GPU present and preferred must set UsedGPU")
	}
	got, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	if !nglIs(got, "99") {
		t.Fatalf("GPU path must pass -ngl 99: %s", got)
	}
}

func TestLocal_NoBinary(t *testing.T) {
	l := llm.Local{
		Backend:   "llama.cpp",
		ModelPath: "/models/fake.gguf",
		Headroom:  ampleRAM(),
	}
	t.Setenv("PATH", t.TempDir())
	_, err := l.Complete(context.Background(), llm.Request{Prompt: "hello"})
	if !errors.Is(err, llm.ErrNoBinary) {
		t.Fatalf("err=%v", err)
	}
}

func TestLocal_LookPathFindsLlamaCLI(t *testing.T) {
	dir := t.TempDir()
	model := filepath.Join(dir, "fake.gguf")
	if err := os.WriteFile(model, []byte("not-a-real-gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := writeFakeLlama(t, "", "from-path")
	t.Setenv("PATH", filepath.Dir(bin))
	l := llm.Local{
		Backend:   "llama.cpp",
		ModelPath: model,
		Headroom:  ampleRAM(),
	}
	resp, err := l.Complete(context.Background(), llm.Request{Prompt: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "from-path" {
		t.Fatalf("text=%q", resp.Text)
	}
}

func TestLocal_RunHookAndExecError(t *testing.T) {
	l := llm.Local{
		Backend:   "llama.cpp",
		Bin:       "/nonexistent/llama-cli",
		ModelPath: "/models/fake.gguf",
		Headroom:  ampleRAM(),
		Run: func(ctx context.Context, argv []string) (string, error) {
			if len(argv) < 3 || argv[1] != "-m" {
				t.Fatalf("argv %v", argv)
			}
			return "hooked", nil
		},
	}
	resp, err := l.Complete(context.Background(), llm.Request{Prompt: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "hooked" {
		t.Fatal(resp.Text)
	}
	l.Run = nil
	_, err = l.Complete(context.Background(), llm.Request{Prompt: "hello"})
	if err == nil {
		t.Fatal("missing binary must fail")
	}
}

func TestLocal_HeadroomRefuse(t *testing.T) {
	ram := int64(1 << 40)
	l := llm.Local{
		Backend:   "llama.cpp",
		Bin:       "/nonexistent/llama-cli",
		ModelPath: "/models/fake.gguf",
		Headroom:  headroom.Snapshot{RAMFreeBytes: 1},
		RAMMin:    &ram,
	}
	_, err := l.Complete(context.Background(), llm.Request{Prompt: "hello", NeedRAM: true})
	if err == nil {
		t.Fatal("expected headroom refusal before exec")
	}
	if errors.Is(err, llm.ErrNoModel) || errors.Is(err, llm.ErrGPURequired) {
		t.Fatalf("wrong error: %v", err)
	}
}
