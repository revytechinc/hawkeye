// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package probe

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

// DefaultHost inspects the live machine. Tests inject Host fakes instead.
type DefaultHost struct {
	ReadFile   func(string) ([]byte, error)
	Stat       func(string) (os.FileInfo, error)
	Glob       func(string) ([]string, error)
	MountTable func() (string, error) // mount -p / getfsstat text; tests inject this
	Sysctl     func(string) (int, bool)
	// Ifaces injects a FreeBSD getifaddrs/ifconfig view. Tests use this
	// instead of faking Linux /sys/class/net/*/carrier.
	Ifaces func() ([]IfaceStatus, error)
}

func Live() DefaultHost {
	return DefaultHost{
		ReadFile:   os.ReadFile,
		Stat:       os.Stat,
		Glob:       filepath.Glob,
		MountTable: liveMountTable,
		// Sysctl left nil so SysctlInt prefers sysctl(8) then native.
	}
}

func (h DefaultHost) read(path string) string {
	if h.ReadFile == nil {
		return ""
	}
	b, err := h.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}

func (h DefaultHost) SysctlInt(name string) (int, bool) {
	if h.Sysctl != nil {
		return h.Sysctl(name)
	}
	// Prefer sysctl(8) host overlay (T010), then native syscall/libc.
	if v, ok := Sysctl8Int(name); ok {
		return v, true
	}
	return liveSysctlInt(name)
}

func (h DefaultHost) PathExists(path string) bool {
	if h.Stat == nil {
		_, err := os.Stat(path)
		return err == nil
	}
	_, err := h.Stat(path)
	return err == nil
}

func (h DefaultHost) PathWritable(path string) bool {
	if !h.PathExists(path) {
		return false
	}
	// syscall.Access does not create files; safe on a read-only root.
	err := unix.Access(path, unix.W_OK)
	return err == nil
}

func (h DefaultHost) MountReadOnly(path string) bool {
	data := h.read("/proc/mounts")
	if data == "" {
		data = h.read("/etc/mtab")
	}
	if data == "" && h.MountTable != nil {
		if b, err := h.MountTable(); err == nil {
			data = b
		}
	}
	if data != "" {
		if ro, found := MountPointReadOnly(data, path); found {
			return ro
		}
	}
	// Jail-safe: getfsstat/statfs. Never treat Access(W_OK) failure as mount RO.
	return liveStatfsReadOnly(path)
}

func (h DefaultHost) NetworkCarrier() bool {
	if h.sysfsCarrier() {
		return true
	}
	if h.Ifaces != nil {
		ifaces, err := h.Ifaces()
		if err == nil {
			return CarrierUp(ifaces)
		}
	}
	return liveNetworkCarrier()
}

func (h DefaultHost) sysfsCarrier() bool {
	g := h.Glob
	if g == nil {
		g = filepath.Glob
	}
	matches, err := g("/sys/class/net/*/carrier")
	if err != nil || len(matches) == 0 {
		return false
	}
	for _, m := range matches {
		if strings.Contains(m, "/lo/") || IsLoopbackName(sysfsIfaceName(m)) {
			continue
		}
		if strings.TrimSpace(h.read(m)) == "1" {
			return true
		}
	}
	return false
}

func sysfsIfaceName(path string) string {
	dir := filepath.Dir(path)
	return filepath.Base(dir)
}

func (h DefaultHost) GPUPresent() bool {
	// nvd(4) is NVMe disk, not a GPU. nvme(4) likewise.
	for _, p := range []string{"/dev/nvidia0", "/dev/dri/card0"} {
		if h.PathExists(p) {
			return true
		}
	}
	return false
}
