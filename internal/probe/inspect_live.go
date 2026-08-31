// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package probe

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

// RescuePATH is the FreeBSD rescue-safe PATH. Never assume /usr is mounted.
const RescuePATH = "/rescue:/sbin:/bin:/usr/sbin:/usr/bin:/usr/local/sbin:/usr/local/bin"

func LiveSources() Sources {
	return Sources{
		Live:        true,
		ReadFile:    os.ReadFile,
		Stat:        os.Stat,
		MountTable:  liveMountTable,
		Ifaces:      liveIfaces,
		ZpoolList:   func() (string, error) { return liveCmd("zpool", "list", "-H", "-o", "name,health") },
		ZpoolStatus: func() (string, error) { return liveCmd("zpool", "status") },
		ZpoolGet:    func() (string, error) { return liveCmd("zpool", "get", "-H", "readonly,bootfs") },
		Routes:      liveRoutes,
		GeliStatus:  func() (string, error) { return liveCmd("geli", "status") },
		Disk:        liveDiskUse,
		LookPath:    lookRescue,
	}
}

func lookRescue(name string) string {
	if name == "" {
		return ""
	}
	if strings.Contains(name, "/") {
		if _, err := os.Stat(name); err == nil {
			return name
		}
		return ""
	}
	for _, dir := range strings.Split(RescuePATH, ":") {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func liveCmd(name string, args ...string) (string, error) {
	path := lookRescue(name)
	if path == "" {
		return "", os.ErrNotExist
	}
	cmd := exec.Command(path, args...)
	cmd.Env = append(os.Environ(), "PATH="+RescuePATH)
	out, err := cmd.Output()
	return string(out), err
}

func liveRoutes() (string, error) {
	if out, err := liveCmd("netstat", "-rn"); err == nil {
		return out, nil
	}
	return liveCmd("route", "-n", "get", "default")
}

func liveDiskUse(path string) (DiskUse, bool) {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return DiskUse{}, false
	}
	bsize := int64(st.Bsize)
	if bsize <= 0 {
		bsize = 512
	}
	return DiskUse{
		TotalBytes:  int64(st.Blocks) * bsize,
		FreeBytes:   int64(st.Bavail) * bsize,
		TotalInodes: int64(st.Files),
		FreeInodes:  int64(st.Ffree),
	}, true
}

func liveIfaces() ([]IfaceStatus, error) {
	// getifaddrs via net.Interfaces; FreeBSD refines with SIOCGIFMEDIA.
	return liveIfaceStatuses()
}
