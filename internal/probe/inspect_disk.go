// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package probe

import "fmt"

const diskFullPct = 95

func inspectDisk(r *Report, src Sources, mounts string) {
	paths := []string{"/"}
	mounted := mountedSet(mounts)
	for _, p := range []string{"/var", "/usr"} {
		if _, ok := mounted[p]; ok {
			paths = append(paths, p)
		}
	}
	for _, p := range paths {
		u, ok := diskAt(src, p)
		if !ok || u.TotalBytes <= 0 {
			continue
		}
		used := 100 * (u.TotalBytes - u.FreeBytes) / u.TotalBytes
		if u.FreeBytes == 0 || used >= diskFullPct {
			r.add("disk", fmt.Sprintf("%s is %d%% full (%d bytes free); free space before writes", p, used, u.FreeBytes))
		}
		if u.TotalInodes > 0 && u.FreeInodes == 0 {
			r.add("disk", p+" is out of inodes (0 free); remove files or grow the filesystem")
		}
	}
}

func diskAt(src Sources, path string) (DiskUse, bool) {
	if src.Disk != nil {
		return src.Disk(path)
	}
	if src.Live {
		return liveDiskUse(path)
	}
	return DiskUse{}, false
}
