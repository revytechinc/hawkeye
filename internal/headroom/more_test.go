// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package headroom_test

import (
	"testing"

	"github.com/revytechinc/hawkeye/internal/headroom"
)

func TestAllow_CPUAndDisk(t *testing.T) {
	cpu := 10.0
	disk := int64(100)
	snap := headroom.Snapshot{CPUPct: 50, DiskFreeBytes: 1, RAMFreeBytes: 1 << 30}
	if err := headroom.Allow(headroom.Job{NeedCPU: true}, snap, nil, &cpu, nil, nil); err == nil {
		t.Fatal("cpu")
	}
	if err := headroom.Allow(headroom.Job{NeedDisk: true}, snap, nil, nil, &disk, nil); err == nil {
		t.Fatal("disk")
	}
}

func TestAllow_GPUPresentVRAMOK(t *testing.T) {
	free := int64(1 << 30)
	min := int64(1)
	snap := headroom.Snapshot{GPUPresent: true, GPUVRAMFreeBytes: &free, RAMFreeBytes: 1 << 30}
	if err := headroom.Allow(headroom.Job{NeedGPU: true}, snap, nil, nil, nil, &min); err != nil {
		t.Fatal(err)
	}
}
