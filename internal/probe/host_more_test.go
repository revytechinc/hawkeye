// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package probe_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/revytechinc/hawkeye/internal/probe"
)

func TestDefaultHost_MountsAndGPU(t *testing.T) {
	dir := t.TempDir()
	mounts := filepath.Join(dir, "mounts")
	if err := os.WriteFile(mounts, []byte("/dev/ada0p2 / ufs ro 0 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	h := probe.DefaultHost{
		ReadFile: os.ReadFile,
		Stat:     os.Stat,
		Glob:     func(string) ([]string, error) { return nil, nil },
	}
	_ = h.PathExists(dir)
	_ = h.PathWritable(dir)
	_ = h.GPUPresent()
	_ = h.NetworkCarrier()
	h.ReadFile = func(path string) ([]byte, error) {
		if path == "/proc/mounts" {
			return os.ReadFile(mounts)
		}
		return nil, os.ErrNotExist
	}
	if !h.MountReadOnly("/") {
		t.Fatal("expected ro")
	}
}

func TestDefaultHost_CarrierLoopbackIgnored(t *testing.T) {
	h := probe.DefaultHost{
		ReadFile: func(path string) ([]byte, error) {
			if path == "/sys/class/net/lo/carrier" {
				return []byte("1\n"), nil
			}
			if filepath.Base(path) == "carrier" && filepath.Base(filepath.Dir(path)) == "em0" {
				return []byte("1\n"), nil
			}
			return nil, os.ErrNotExist
		},
		Stat: os.Stat,
		Glob: func(string) ([]string, error) {
			return []string{"/sys/class/net/lo/carrier", "/sys/class/net/em0/carrier"}, nil
		},
	}
	if !h.NetworkCarrier() {
		t.Fatal("em0 carrier")
	}
}

func TestLiveConstructor(t *testing.T) {
	h := probe.Live()
	if h.ReadFile == nil || h.Stat == nil {
		t.Fatal("live")
	}
}
