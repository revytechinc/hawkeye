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
		s.RAMFreeBytes, s.RAMTotalBytes = parseMeminfo(string(b))
	}
	if s.RAMTotalBytes == 0 {
		s.RAMFreeBytes, s.RAMTotalBytes = liveRAM()
	}
	var st syscall.Statfs_t
	if err := syscall.Statfs("/", &st); err == nil {
		s.DiskFreeBytes = int64(st.Bavail) * int64(st.Bsize)
	}
	return s
}

func parseMeminfo(data string) (free, total int64) {
	for _, line := range strings.Split(data, "\n") {
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		n, _ := strconv.ParseInt(f[1], 10, 64)
		n *= 1024
		switch f[0] {
		case "MemTotal:":
			total = n
		case "MemAvailable:":
			free = n
		}
	}
	return free, total
}

func ramFromVMStats(page, free, inactive, cache, total uint32) (freeB, totalB int64) {
	if page == 0 {
		return 0, 0
	}
	avail := uint64(free) + uint64(inactive) + uint64(cache)
	return int64(avail * uint64(page)), int64(uint64(total) * uint64(page))
}
