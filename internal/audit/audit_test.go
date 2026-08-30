// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package audit_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/revytechinc/hawkeye/internal/apply"
	"github.com/revytechinc/hawkeye/internal/audit"
)

func TestFile_RecordsApply(t *testing.T) {
	p := filepath.Join(t.TempDir(), "audit.log")
	a := &audit.File{Path: p, Now: func() time.Time { return time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC) }}
	plan := apply.Plan{ID: "p1", Source: "operator"}
	res, err := apply.Execute(plan, apply.ModeDryRun, apply.ActorOperator, &apply.CountingExecutor{}, a)
	if err != nil {
		t.Fatal(err)
	}
	if !res.DryRun {
		t.Fatal("expected dry-run")
	}
	evs, err := a.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 1 || evs[0].PlanID != "p1" {
		t.Fatalf("%+v", evs)
	}
}
