// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package headroom

import (
	"os"
	"strconv"
	"strings"
	"syscall"
)

func Live(gpuPresent bool) Snapshot {
	s := Snapshot{GPUPresent: gpuPresent}
	if b, err := os.ReadFile("/proc/meminfo"); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			f := strings.Fields(line)
			if len(f) < 2 {
				continue
			}
			n, _ := strconv.ParseInt(f[1], 10, 64)
			n *= 1024
			switch f[0] {
			case "MemTotal:":
				s.RAMTotalBytes = n
			case "MemAvailable:":
				s.RAMFreeBytes = n
			}
		}
	}
	var st syscall.Statfs_t
	if err := syscall.Statfs("/", &st); err == nil {
		s.DiskFreeBytes = int64(st.Bavail) * int64(st.Bsize)
	}
	return s
}
