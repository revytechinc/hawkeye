// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package probe

import "strings"

// MountPointReadOnly parses mount -p, fstab, /proc/mounts, or /etc/mtab.
// Each line is: fs mountpoint type options [dump pass]
func MountPointReadOnly(table, mountpoint string) (ro bool, found bool) {
	for _, line := range strings.Split(table, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Fields(line)
		if len(f) < 4 {
			continue
		}
		if f[1] != mountpoint {
			continue
		}
		found = true
		for _, o := range strings.Split(f[3], ",") {
			if o == "ro" {
				return true, true
			}
		}
		return false, true
	}
	return false, false
}
