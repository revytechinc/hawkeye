// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package consult_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/revytechinc/hawkeye/internal/consult"
	"github.com/revytechinc/hawkeye/internal/knowledge"
	"github.com/revytechinc/hawkeye/internal/probe"
)

func TestRun_Tier0UsesFTSNoWrite(t *testing.T) {
	dir := t.TempDir()
	if err := knowledge.CreateTestDB(filepath.Join(dir, "knowledge.sqlite")); err != nil {
		t.Fatal(err)
	}
	st, err := knowledge.Open([]string{dir}, true)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	snap := probe.Snapshot{RootRO: true, Tier: 0}
	r, err := consult.Run("zfs readonly", snap, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.FirstSkill != "unlock-rw" {
		t.Fatalf("first skill %q", r.FirstSkill)
	}
	if len(r.Hits) < 1 {
		t.Fatal("expected FTS hits")
	}
	if r.LLM != nil {
		t.Fatal("tier 0 must not require LLM")
	}
	joined := strings.Join(r.Notes, " ")
	if !strings.Contains(joined, "unlock-rw") {
		t.Fatalf("notes %v", r.Notes)
	}
}

func TestRun_RedactsQuery(t *testing.T) {
	snap := probe.Snapshot{Tier: 0, RootRO: true}
	r, err := consult.Run("password=fake-password-for-tests-only", snap, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(r.Query, "fake-password-for-tests-only") {
		t.Fatalf("query leaked: %q", r.Query)
	}
}
