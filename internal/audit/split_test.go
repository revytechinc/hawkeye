// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package audit_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/revytechinc/hawkeye/internal/apply"
	"github.com/revytechinc/hawkeye/internal/audit"
)

func TestFile_EmptyPathAndTrailing(t *testing.T) {
	var f *audit.File
	if err := f.Record(apply.Plan{}, apply.ModeDryRun, apply.ActorOperator, apply.Result{}); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "a.log")
	a := &audit.File{Path: p}
	_ = a.Record(apply.Plan{ID: "a"}, apply.ModeDryRun, apply.ActorOperator, apply.Result{})
	_ = a.Record(apply.Plan{ID: "b"}, apply.ModeDryRun, apply.ActorOperator, apply.Result{})
	if err := os.WriteFile(p, append([]byte(mustRead(t, p)), []byte("not-json\n")...), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := a.ReadAll(); err == nil {
		t.Fatal("bad json line")
	}
}

func mustRead(t *testing.T, p string) []byte {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
