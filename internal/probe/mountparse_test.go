// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package probe_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/revytechinc/hawkeye/internal/probe"
)

func TestMountPointReadOnly_MountPFreeBSD(t *testing.T) {
	rw := "zroot/bastille/jails/hawkeye/root /\t\tzfs\trw,noatime,nfsv4acls \t0 0\n"
	ro, found := probe.MountPointReadOnly(rw, "/")
	if !found {
		t.Fatal("expected to find /")
	}
	if ro {
		t.Fatal("mount -p rw,noatime must not be read-only")
	}

	readonly := "zroot/ROOT/default / zfs ro,noatime 0 0\n"
	ro, found = probe.MountPointReadOnly(readonly, "/")
	if !found || !ro {
		t.Fatalf("expected ro found=%v ro=%v", found, ro)
	}

	proc := "/dev/ada0p2 / ufs ro 0 0\n"
	ro, found = probe.MountPointReadOnly(proc, "/")
	if !found || !ro {
		t.Fatal("proc mounts ro")
	}
}

func TestDefaultHost_MountPReadWriteNotAccess(t *testing.T) {
	h := probe.DefaultHost{
		ReadFile: func(string) ([]byte, error) { return nil, os.ErrNotExist },
		Stat:     os.Stat,
		MountTable: func() (string, error) {
			return "zroot/bastille/jails/hawkeye/root / zfs rw,noatime,nfsv4acls 0 0\n", nil
		},
	}
	if h.MountReadOnly("/") {
		t.Fatal("fake mount -p rw must not report RO")
	}
	s := probe.Probe(h)
	if s.RootRO {
		t.Fatalf("Access(/) must not force RootRO when mount -p is rw: %+v writable=%v", s, h.PathWritable("/"))
	}
	if s.FirstSkill() == "unlock-rw" {
		t.Fatal("unlock-rw on rw mount")
	}
}

func TestDefaultHost_NVDIsNotGPU(t *testing.T) {
	dir := t.TempDir()
	nvd := filepath.Join(dir, "nvd0")
	if err := os.WriteFile(nvd, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	h := probe.DefaultHost{
		Stat: func(path string) (os.FileInfo, error) {
			switch path {
			case "/dev/nvd0", "/dev/nvme0":
				return os.Stat(nvd)
			default:
				return nil, os.ErrNotExist
			}
		},
	}
	if h.GPUPresent() {
		t.Fatal("NVMe nvd0/nvme0 must not count as GPU")
	}
}

func TestDefaultHost_NvidiaIsGPU(t *testing.T) {
	dir := t.TempDir()
	dev := filepath.Join(dir, "nvidia0")
	if err := os.WriteFile(dev, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	h := probe.DefaultHost{
		Stat: func(path string) (os.FileInfo, error) {
			if path == "/dev/nvidia0" {
				return os.Stat(dev)
			}
			return nil, os.ErrNotExist
		},
	}
	if !h.GPUPresent() {
		t.Fatal("nvidia0 is a GPU")
	}
}

func TestLive_RootROFollowsMountNotAccess(t *testing.T) {
	if runtime.GOOS != "freebsd" {
		t.Skip("freebsd mount/statfs")
	}
	h := probe.Live()
	s := probe.Probe(h)
	if h.MountReadOnly("/") {
		if !s.RootRO {
			t.Fatal("mount ro should set RootRO")
		}
		return
	}
	if s.RootRO {
		t.Fatalf("rw mount must not set RootRO because Access(/) failed: %+v writable=%v", s, h.PathWritable("/"))
	}
	if s.FirstSkill() == "unlock-rw" {
		t.Fatal("first skill unlock-rw on rw root")
	}
}
