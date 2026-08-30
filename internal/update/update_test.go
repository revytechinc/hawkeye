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

func TestRun_RefusesWhenRootRO(t *testing.T) {
	_, err := update.Run("src", "dst", probe.Snapshot{RootRO: true, Tier: 0})
	if err == nil {
		t.Fatal("expected refuse")
	}
}

func TestRun_CopiesArtifactWhenWritable(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.sqlite")
	dst := filepath.Join(dir, "dest", "knowledge.sqlite")
	if err := os.WriteFile(src, []byte("fake-knowledge-db"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := update.Run(src, dst, probe.Snapshot{RootRO: false, Tier: 1})
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(got)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "fake-knowledge-db" {
		t.Fatalf("got %q", b)
	}
}
