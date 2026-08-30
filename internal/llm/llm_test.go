// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package llm_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/revytechinc/hawkeye/internal/headroom"
	"github.com/revytechinc/hawkeye/internal/llm"
)

func TestLocal_MissingGPUDoesNotBlockCPUJob(t *testing.T) {
	ram := int64(1)
	l := llm.Local{
		Backend:    "llama.cpp",
		PreferGPU:  true,
		RequireGPU: false,
		GPUPresent: false,
		ModelPath:  "/nonexistent/model.gguf",
		Headroom:   headroom.Snapshot{RAMFreeBytes: 1 << 30, GPUPresent: false},
		RAMMin:     &ram,
	}
	_, err := l.Complete(context.Background(), llm.Request{Prompt: "hello", NeedGPU: false, NeedRAM: true})
	if err != nil && !errors.Is(err, llm.ErrNoModel) {
		t.Fatalf("CPU job blocked: %v", err)
	}
}

func TestLocal_RequireGPUFailsWithoutDevice(t *testing.T) {
	l := llm.Local{RequireGPU: true, GPUPresent: false, Headroom: headroom.Snapshot{RAMFreeBytes: 1 << 30}}
	_, err := l.Complete(context.Background(), llm.Request{NeedGPU: true})
	if !errors.Is(err, llm.ErrGPURequired) {
		t.Fatalf("err=%v", err)
	}
}

func TestLocal_RedactsPrompt(t *testing.T) {
	l := llm.Local{
		Backend:   "llama.cpp",
		ModelPath: "/models/fake.gguf",
		Headroom:  headroom.Snapshot{RAMFreeBytes: 1 << 30},
	}
	resp, err := l.Complete(context.Background(), llm.Request{Prompt: "password=fake-password-for-tests-only"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(resp.Text, "fake-password-for-tests-only") {
		t.Fatal("secret leaked into LLM response path")
	}
}
