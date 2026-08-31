// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package knowledge_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/revytechinc/hawkeye/internal/knowledge"
)

func TestCanStageRescue_DanglingSymlinkSkipped(t *testing.T) {
	dir := t.TempDir()
	rescue := filepath.Join(dir, "rescue")
	if err := os.Symlink(filepath.Join(dir, "missing-target"), rescue); err != nil {
		t.Fatal(err)
	}
	if knowledge.CanStageRescue("", rescue, os.Lstat) {
		t.Fatal("dangling bastille /rescue must be skipped")
	}
	if !knowledge.CanStageRescue("/var/tmp/stage", rescue, os.Lstat) {
		t.Fatal("DESTDIR/STAGEDIR must still stage /rescue")
	}
}

func TestCanStageRescue_RealDirectoryAllowed(t *testing.T) {
	dir := t.TempDir()
	rescue := filepath.Join(dir, "rescue")
	if err := os.Mkdir(rescue, 0o755); err != nil {
		t.Fatal(err)
	}
	if !knowledge.CanStageRescue("", rescue, os.Lstat) {
		t.Fatal("real /rescue directory must be installable")
	}
}

func TestCanStageRescue_MissingSkipped(t *testing.T) {
	dir := t.TempDir()
	if knowledge.CanStageRescue("", filepath.Join(dir, "rescue"), os.Lstat) {
		t.Fatal("missing /rescue must be skipped on a live jail")
	}
	if knowledge.CanStageRescue("", "", os.Lstat) {
		t.Fatal("empty rescue dir")
	}
}

func TestCanStageBootKit_BootExistsNoHawkeye(t *testing.T) {
	dir := t.TempDir()
	boot := filepath.Join(dir, "boot")
	if err := os.Mkdir(boot, 0o755); err != nil {
		t.Fatal(err)
	}
	kit := filepath.Join(boot, "hawkeye")
	if _, err := os.Stat(kit); !os.IsNotExist(err) {
		t.Fatal("fixture must not already have /boot/hawkeye")
	}
	if !knowledge.CanStageBootKit("", kit, os.Lstat) {
		t.Fatal("/boot exists: may create /boot/hawkeye")
	}
	if !knowledge.CanStageBootKit("/var/tmp/stage", kit, os.Lstat) {
		t.Fatal("DESTDIR must stage /boot/hawkeye even without live /boot")
	}
}

func TestCanStageBootKit_MissingBootSkipped(t *testing.T) {
	dir := t.TempDir()
	kit := filepath.Join(dir, "no-boot", "hawkeye")
	if knowledge.CanStageBootKit("", kit, os.Lstat) {
		t.Fatal("missing /boot must not invent a boot filesystem")
	}
	if knowledge.CanStageBootKit("", "", os.Lstat) {
		t.Fatal("empty boot kit")
	}
}
