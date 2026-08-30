// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package headroom

import (
	"runtime"
	"testing"
)

func TestParseMeminfo(t *testing.T) {
	free, total := parseMeminfo("MemTotal:       16384 kB\nMemAvailable:    4096 kB\n")
	if total != 16384*1024 || free != 4096*1024 {
		t.Fatalf("free=%d total=%d", free, total)
	}
}

func TestRAMFromVMStats(t *testing.T) {
	// Live hawkeye jail sample: page=4096, free+inactive should be >> 256MiB.
	free, total := ramFromVMStats(4096, 16587311, 3715478, 0, 32781768)
	if total == 0 || free == 0 {
		t.Fatalf("free=%d total=%d", free, total)
	}
	if free < 256<<20 {
		t.Fatalf("free %d too small", free)
	}
}

func TestLive_FreeBSDRAMFromSysctl(t *testing.T) {
	if runtime.GOOS != "freebsd" {
		t.Skip("sysctl vm.stats")
	}
	s := Live(false)
	if s.RAMFreeBytes == 0 || s.RAMTotalBytes == 0 {
		t.Fatalf("expected sysctl RAM, got %+v", s)
	}
}
