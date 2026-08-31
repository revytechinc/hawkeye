// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package update_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/revytechinc/hawkeye/internal/probe"
	"github.com/revytechinc/hawkeye/internal/update"
)

func TestRun_MissingSourceFile(t *testing.T) {
	if _, err := update.Run("s", "", probe.Snapshot{}); err == nil {
		t.Fatal("open of missing src must fail (dest defaults; do not write live ZFS)")
	}
	if _, err := update.Run("/no/such/src", "/tmp/x", probe.Snapshot{}); err == nil {
		t.Fatal("open")
	}
}

func TestRun_DestParentIsFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.sqlite")
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	blocker := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(blocker, []byte("nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(blocker, "knowledge.sqlite")
	if _, err := update.Run(src, dest, probe.Snapshot{}); err == nil {
		t.Fatal("mkdir through a file must fail (no live ZFS remount)")
	}
}

func TestSourceFromEnvNames(t *testing.T) {
	if update.SourceEnv != "HAWKEYE_UPDATE_SOURCE" {
		t.Fatalf("SourceEnv = %q", update.SourceEnv)
	}
	if update.LegacySourceEnv != "HAWKEYE_DATA_ARTIFACT" {
		t.Fatalf("LegacySourceEnv = %q", update.LegacySourceEnv)
	}
}
