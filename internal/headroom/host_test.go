// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package headroom_test

import (
	"testing"

	"github.com/revytechinc/hawkeye/internal/headroom"
)

func TestLive_ReportsRAMOrZero(t *testing.T) {
	s := headroom.Live(false)
	if s.GPUPresent {
		t.Fatal("gpuPresent passed false")
	}
	_ = s.RAMFreeBytes
	_ = s.DiskFreeBytes
}
