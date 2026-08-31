// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePath_UserWinsWhenSystemExists(t *testing.T) {
	t.Cleanup(func() { systemDir = "/usr/local/etc/cloudbsd/hawkeye" })
	sys := t.TempDir()
	systemDir = sys
	userHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", userHome)
	sysp := filepath.Join(sys, "config.json")
	userp := filepath.Join(userHome, "hawkeye", "config.json")
	if err := os.MkdirAll(filepath.Dir(userp), 0o755); err != nil {
		t.Fatal(err)
	}
	b, err := InitJSON()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sysp, b, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(userp, b, 0o600); err != nil {
		t.Fatal(err)
	}
	got := ResolvePath("")
	if got != userp {
		t.Fatalf("user config must win when both exist: got %s want %s", got, userp)
	}
}

func TestResolvePath_SampleOnlyDoesNotSelectSample(t *testing.T) {
	t.Cleanup(func() { systemDir = "/usr/local/etc/cloudbsd/hawkeye" })
	sys := t.TempDir()
	systemDir = sys
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	b, err := InitJSON()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sys, "config.json.sample"), b, 0o644); err != nil {
		t.Fatal(err)
	}
	got := ResolvePath("")
	want := filepath.Join(sys, "config.json")
	if got != want {
		t.Fatalf("sample must not be treated as the live config: got %s want %s", got, want)
	}
	if err := CheckFile(got); err != nil {
		t.Fatalf("sample-only system dir must check as defaults: %v", err)
	}
}

func TestResolvePath_SystemWhenUserMissing(t *testing.T) {
	t.Cleanup(func() { systemDir = "/usr/local/etc/cloudbsd/hawkeye" })
	sys := t.TempDir()
	systemDir = sys
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	sysp := filepath.Join(sys, "config.json")
	b, err := InitJSON()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sysp, b, 0o644); err != nil {
		t.Fatal(err)
	}
	got := ResolvePath("")
	if got != sysp {
		t.Fatalf("system config when user missing: got %s want %s", got, sysp)
	}
}
