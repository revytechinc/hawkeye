// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package consult_test

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/revytechinc/hawkeye/internal/consult"
	"github.com/revytechinc/hawkeye/internal/knowledge"
	"github.com/revytechinc/hawkeye/internal/probe"

	_ "modernc.org/sqlite"
)

func TestRun_EmptyEmbeddingsFTSQuietTTY(t *testing.T) {
	dir := t.TempDir()
	if err := knowledge.CreatePlaybookTestDB(filepath.Join(dir, knowledge.DBName)); err != nil {
		t.Fatal(err)
	}
	st, err := knowledge.Open([]string{dir}, true)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	r, err := consult.Run("ZFS root is read-only after boot", probe.Snapshot{Tier: 0, RootRO: true}, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Hits) < 1 {
		t.Fatal("jail empty embeddings must still FTS")
	}
	got := r.Human()
	for _, junk := range []string{
		"embeddings", "sqlite-vec", "vec_distance", `"vector"`, "QueryVec",
	} {
		if strings.Contains(got, junk) {
			t.Fatalf("TTY leaked %q:\n%s", junk, got)
		}
	}
}

func TestRun_VectorsRankHitsJSONKeepsMachineObject(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, knowledge.DBName)
	if err := knowledge.CreatePlaybookTestDB(p); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", p)
	if err != nil {
		t.Fatal(err)
	}
	if err := knowledge.InsertEmbedding(db, "playbooks", "zfs-remount-rw", "fake-test", []float32{1, 0, 0}); err != nil {
		t.Fatal(err)
	}
	if err := knowledge.InsertEmbedding(db, "playbooks", "bectl-rollback", "fake-test", []float32{0, 1, 0}); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()

	st, err := knowledge.Open([]string{dir}, true)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	st.QueryVec = []float32{1, 0, 0}
	r, err := consult.Run("boot environment", probe.Snapshot{Tier: 1}, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Hits) < 1 || r.Hits[0].Title != knowledge.RemountPlaybookTitle {
		t.Fatalf("consult must rank with vectors: %#v", r.Hits)
	}
	human := r.Human()
	if strings.Contains(human, "vec_distance") || strings.Contains(human, `"hits"`) {
		t.Fatalf("TTY leaked guts:\n%s", human)
	}
	b, err := r.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(b) {
		t.Fatal(string(b))
	}
	if !strings.Contains(string(b), `"hits"`) {
		t.Fatalf("JSON missing hits: %s", b)
	}
}
