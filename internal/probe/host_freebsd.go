// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

//go:build freebsd

package probe

import (
	"bytes"
	"fmt"
	"strings"

	"golang.org/x/sys/unix"
)

func cstring(b []byte) string {
	if i := bytes.IndexByte(b, 0); i >= 0 {
		b = b[:i]
	}
	return string(b)
}

func liveSysctlInt(name string) (int, bool) {
	v, err := unix.SysctlUint32(name)
	if err != nil {
		return 0, false
	}
	return int(v), true
}

func liveMountTable() (string, error) {
	n, err := unix.Getfsstat(nil, unix.MNT_NOWAIT)
	if err != nil || n <= 0 {
		return "", err
	}
	buf := make([]unix.Statfs_t, n)
	n, err = unix.Getfsstat(buf, unix.MNT_NOWAIT)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for i := 0; i < n; i++ {
		st := buf[i]
		on := cstring(st.Mntonname[:])
		from := cstring(st.Mntfromname[:])
		fstype := cstring(st.Fstypename[:])
		opts := "rw"
		if st.Flags&unix.MNT_RDONLY != 0 {
			opts = "ro"
		}
		fmt.Fprintf(&b, "%s %s %s %s 0 0\n", from, on, fstype, opts)
	}
	return b.String(), nil
}

func liveStatfsReadOnly(path string) bool {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return false
	}
	return st.Flags&unix.MNT_RDONLY != 0
}
