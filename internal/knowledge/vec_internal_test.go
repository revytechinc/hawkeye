// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package knowledge

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
	_ "modernc.org/sqlite/vec"
)

func TestProbeVectorsAndHelpersNil(t *testing.T) {
	var s *Store
	s.probeVectors()
	s = &Store{}
	s.probeVectors()
	if vecAvailable(nil) {
		t.Fatal("nil db")
	}
	if embeddingRows(nil) != 0 {
		t.Fatal("nil rows")
	}
	if embeddingRows(s.DB) != 0 {
		t.Fatal("nil store db")
	}
}

func TestLookupTargetEdges(t *testing.T) {
	if _, ok := (*Store)(nil).lookupTarget("playbooks", "x", 0); ok {
		t.Fatal("nil store")
	}
	dir := t.TempDir()
	p := filepath.Join(dir, DBName)
	if err := CreatePlaybookTestDB(p); err != nil {
		t.Fatal(err)
	}
	st, err := Open([]string{dir}, true)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if _, ok := st.lookupTarget("playbooks", "", 0); ok {
		t.Fatal("empty id")
	}
	if _, ok := st.lookupTarget("nope", "zfs-remount-rw", 0); ok {
		t.Fatal("unknown table")
	}
	if _, ok := st.lookupTarget("playbooks", "missing-id", 0); ok {
		t.Fatal("missing playbook")
	}
	if _, ok := st.lookupTarget("documents", "missing-id", 0); ok {
		t.Fatal("missing document")
	}
	if h, ok := st.lookupTarget("documents", "zfs-emergency", 0.5); !ok || h.Title == "" {
		t.Fatal("document lookup")
	}
}

func TestMergeHitsEdges(t *testing.T) {
	out := mergeHits(
		[]Hit{{Title: "", Body: "same-body"}, {Title: "", Body: ""}},
		[]Hit{{Title: "Lead", Body: "x"}, {Title: "", Body: "same-body"}},
		0,
	)
	if len(out) < 2 || out[0].Title != "Lead" {
		t.Fatalf("%#v", out)
	}
	many := []Hit{{Title: "a"}, {Title: "b"}, {Title: "c"}}
	got := mergeHits(many, nil, 1)
	if len(got) != 1 || got[0].Title != "a" {
		t.Fatalf("%#v", got)
	}
}

func TestOpenRW_MissingFTSAndOpenFail(t *testing.T) {
	if _, err := OpenRW(filepath.Join(t.TempDir(), "empty.sqlite")); err == nil {
		t.Fatal("new empty sqlite has no FTS")
	}
	garbage := filepath.Join(t.TempDir(), "bad.sqlite")
	if err := os.WriteFile(garbage, []byte("not a database"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenRW(garbage); err == nil {
		t.Fatal("garbage")
	}
	old := sqlOpen
	sqlOpen = func(string, string) (*sql.DB, error) { return nil, os.ErrPermission }
	t.Cleanup(func() { sqlOpen = old })
	if _, err := OpenRW(filepath.Join(t.TempDir(), "x.sqlite")); err == nil {
		t.Fatal("sqlOpen fail")
	}
}

func TestOpenRW_LegacyKitFillNoChunks(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, DBName)
	if err := CreateTestDB(p); err != nil {
		t.Fatal(err)
	}
	rw, err := OpenRW(p)
	if err != nil {
		t.Fatal(err)
	}
	defer rw.Close()
	if err := rw.FillEmbeddings(context.Background(), &FakeEmbedder{Name: "fake-test", Default: []float32{1}}); err != nil {
		t.Fatal(err)
	}
}

func TestQueryVectorEmptyAndSearchVecOnly(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, DBName)
	if err := CreatePlaybookTestDB(p); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", p)
	if err != nil {
		t.Fatal(err)
	}
	if err := InsertEmbedding(db, "playbooks", "zfs-remount-rw", "fake-test", []float32{1, 0, 0}); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	st, err := Open([]string{dir}, true)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	st.Embedder = &FakeEmbedder{Default: nil}
	if st.queryVector("x") != nil {
		t.Fatal("empty embed")
	}
	st.Embedder = nil
	st.QueryVec = []float32{1, 0, 0}
	hits, err := st.Search("...", 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) < 1 || hits[0].Title != RemountPlaybookTitle {
		t.Fatalf("vector-only when FTS match empty: %#v", hits)
	}
}

func TestVectorHitsGoDimMismatchAndSQLFallback(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, DBName)
	if err := CreatePlaybookTestDB(p); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", p)
	if err != nil {
		t.Fatal(err)
	}
	if err := InsertEmbedding(db, "playbooks", "zfs-remount-rw", "fake-test", []float32{1, 0, 0}); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	st, err := Open([]string{dir}, true)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	st.Vec = false
	st.hasEmb = true
	st.QueryVec = []float32{1, 2}
	hits, err := st.vectorHitsGo(st.QueryVec, 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 0 {
		t.Fatalf("dim mismatch must skip: %#v", hits)
	}
	st.DB.Close()
	if _, err := st.vectorHitsSQL([]float32{1, 0, 0}, 2); err == nil {
		t.Fatal("closed db sql")
	}
	if _, err := st.vectorHitsGo([]float32{1, 0, 0}, 2); err == nil {
		t.Fatal("closed db go")
	}
}

func TestFillEmbeddings_EmptyVecSkipped(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, DBName)
	if err := CreatePlaybookTestDB(p); err != nil {
		t.Fatal(err)
	}
	rw, err := OpenRW(p)
	if err != nil {
		t.Fatal(err)
	}
	defer rw.Close()
	if err := rw.FillEmbeddings(context.Background(), &FakeEmbedder{Name: "fake-test"}); err != nil {
		t.Fatal(err)
	}
}

func TestFillEmbeddings_EmbedError(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, DBName)
	if err := CreatePlaybookTestDB(p); err != nil {
		t.Fatal(err)
	}
	rw, err := OpenRW(p)
	if err != nil {
		t.Fatal(err)
	}
	defer rw.Close()
	if err := rw.FillEmbeddings(context.Background(), &FakeEmbedder{Name: "fake-test", Err: context.Canceled}); err == nil {
		t.Fatal("embed error")
	}
}

func TestSearchVectorsClosedDB(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, DBName)
	if err := CreatePlaybookTestDB(p); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", p)
	if err != nil {
		t.Fatal(err)
	}
	if err := InsertEmbedding(db, "playbooks", "zfs-remount-rw", "fake-test", []float32{1, 0, 0}); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	st, err := Open([]string{dir}, true)
	if err != nil {
		t.Fatal(err)
	}
	st.QueryVec = []float32{1, 0, 0}
	_ = st.DB.Close()
	if _, ok := st.searchVectors("zfs", 2); ok {
		t.Fatal("closed db must not claim vector hits")
	}
}

func TestSearchVectorsMiss(t *testing.T) {
	var s *Store
	if _, ok := s.searchVectors("x", 1); ok {
		t.Fatal("nil")
	}
	s = &Store{hasEmb: true, QueryVec: []float32{1}}
	if _, ok := s.searchVectors("x", 1); ok {
		t.Fatal("no db")
	}
}

func TestVecAvailableClosed(t *testing.T) {
	db, err := sql.Open("sqlite", "file:closedvec?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	if vecAvailable(db) {
		t.Fatal("closed")
	}
	if embeddingRows(db) != 0 {
		t.Fatal("closed rows")
	}
}
