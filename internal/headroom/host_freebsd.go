// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

//go:build freebsd

package headroom

import "golang.org/x/sys/unix"

func sysctlU32(name string) uint32 {
	v, err := unix.SysctlUint32(name)
	if err != nil {
		return 0
	}
	return v
}

func liveRAM() (free, total int64) {
	page := sysctlU32("vm.stats.vm.v_page_size")
	return ramFromVMStats(
		page,
		sysctlU32("vm.stats.vm.v_free_count"),
		sysctlU32("vm.stats.vm.v_inactive_count"),
		sysctlU32("vm.stats.vm.v_cache_count"),
		sysctlU32("vm.stats.vm.v_page_count"),
	)
}
