// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package probe

import (
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"strings"
)

// DefaultHost inspects the live machine. Tests inject Host fakes instead.
type DefaultHost struct {
	ReadFile func(string) ([]byte, error)
	Stat     func(string) (os.FileInfo, error)
	Glob     func(string) ([]string, error)
}

func Live() DefaultHost {
	return DefaultHost{
		ReadFile: os.ReadFile,
		Stat:     os.Stat,
		Glob:     filepath.Glob,
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
	// File-based only so consult never shells out. CLI may overlay values.
	_ = name
	return 0, false
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
	if data == "" {
		return !h.PathWritable(path)
	}
	for _, line := range strings.Split(data, "\n") {
		f := strings.Fields(line)
		if len(f) < 4 {
			continue
		}
		if f[1] == path {
			for _, o := range strings.Split(f[3], ",") {
				if o == "ro" {
					return true
				}
			}
		}
	}
	return false
}

func (h DefaultHost) NetworkCarrier() bool {
	g := h.Glob
	if g == nil {
		g = filepath.Glob
	}
	matches, _ := g("/sys/class/net/*/carrier")
	for _, m := range matches {
		if strings.Contains(m, "/lo/") {
			continue
		}
		if strings.TrimSpace(h.read(m)) == "1" {
			return true
		}
	}
	return false
}

func (h DefaultHost) GPUPresent() bool {
	for _, p := range []string{"/dev/nvidia0", "/dev/dri/card0", "/dev/nvd0"} {
		if h.PathExists(p) {
			return true
		}
	}
	return false
}
