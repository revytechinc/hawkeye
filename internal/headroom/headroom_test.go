// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package headroom_test

import (
	"testing"

	"github.com/revytechinc/hawkeye/internal/headroom"
)

func TestAllow_MissingGPUDoesNotBlockCPUJob(t *testing.T) {
	snap := headroom.Snapshot{RAMFreeBytes: 1 << 30, GPUPresent: false}
	ram := int64(256 << 20)
	job := headroom.Job{NeedRAM: true, NeedGPU: false}
	if err := headroom.Allow(job, snap, &ram, nil, nil, nil); err != nil {
		t.Fatalf("CPU/RAM job blocked without GPU: %v", err)
	}
}

func TestAllow_ExhaustedRAMBlocks(t *testing.T) {
	snap := headroom.Snapshot{RAMFreeBytes: 1024}
	ram := int64(256 << 20)
	job := headroom.Job{NeedRAM: true}
	if err := headroom.Allow(job, snap, &ram, nil, nil, nil); err == nil {
		t.Fatal("expected RAM headroom failure")
	}
}

func TestAllow_GPUJobFailsWhenRequiredVRAMExhausted(t *testing.T) {
	zero := int64(0)
	min := int64(1 << 30)
	snap := headroom.Snapshot{GPUPresent: true, GPUVRAMFreeBytes: &zero, RAMFreeBytes: 1 << 30}
	job := headroom.Job{NeedGPU: true, NeedRAM: true}
	if err := headroom.Allow(job, snap, nil, nil, nil, &min); err == nil {
		t.Fatal("expected VRAM headroom failure")
	}
}
