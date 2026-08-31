// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package knowledge_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/revytechinc/hawkeye/internal/headroom"
	"github.com/revytechinc/hawkeye/internal/knowledge"
	"github.com/revytechinc/hawkeye/internal/redact"

	_ "modernc.org/sqlite"
	_ "modernc.org/sqlite/vec"
)

func TestPackF32_RoundTripAndCosine(t *testing.T) {
	in := []float32{1, 0, 0}
	got := knowledge.UnpackF32(knowledge.PackF32(in))
	if len(got) != 3 || got[0] != 1 || got[1] != 0 || got[2] != 0 {
		t.Fatalf("round-trip %v", got)
	}
	if d := knowledge.DistanceCosine(in, []float32{1, 0, 0}); d > 1e-6 {
		t.Fatalf("identical vectors: %v", d)
	}
	if d := knowledge.DistanceCosine(in, []float32{0, 1, 0}); d < 0.9 {
		t.Fatalf("orthogonal should be near 1, got %v", d)
	}
}

func TestSearch_EmptyEmbeddingsIsFTSOnly(t *testing.T) {
	dir := t.TempDir()
	if err := knowledge.CreatePlaybookTestDB(filepath.Join(dir, knowledge.DBName)); err != nil {
		t.Fatal(err)
	}
	st, err := knowledge.Open([]string{dir}, true)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if !st.FTS {
		t.Fatal("FTS is required when embeddings are empty")
	}
	hits, err := st.Search("boot environment", 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) < 1 {
		t.Fatal("empty embeddings must still FTS")
	}
	if hits[0].Title != "List, activate, or roll back a ZFS boot environment" {
		t.Fatalf("FTS-only order, got %q", hits[0].Title)
	}
}

func TestSearch_VectorsRerankWhenPresent(t *testing.T) {
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
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	st, err := knowledge.Open([]string{dir}, true)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if !st.Vec {
		t.Fatal("sqlite-vec must load so consult can rank FLOAT32 blobs")
	}

	fts, err := st.Search("boot environment", 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(fts) < 1 || fts[0].Title != "List, activate, or roll back a ZFS boot environment" {
		t.Fatalf("control FTS order: %#v", fts)
	}

	st.QueryVec = []float32{0.99, 0.01, 0}
	ranked, err := st.Search("boot environment", 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(ranked) < 1 {
		t.Fatal("expected vector-ranked hits")
	}
	if ranked[0].Title != knowledge.RemountPlaybookTitle {
		t.Fatalf("sqlite-vec must promote remount over FTS decoy, got %#v", ranked)
	}
	if !strings.Contains(strings.Join(ranked[0].Commands, "\n"), `zfs set readonly=off`) {
		t.Fatalf("vector hit must still attach stored commands: %#v", ranked[0].Commands)
	}
}

func TestSearch_LowRAMSkipsEmbeddings(t *testing.T) {
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
	min := int64(256 << 20)
	st.RAMMin = &min
	st.Headroom = headroom.Snapshot{RAMFreeBytes: 1024}
	st.QueryVec = []float32{1, 0, 0}
	hits, err := st.Search("boot environment", 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) < 1 {
		t.Fatal("low RAM must fall back to FTS, not error")
	}
	if hits[0].Title != "List, activate, or roll back a ZFS boot environment" {
		t.Fatalf("low RAM must keep FTS order, got %q", hits[0].Title)
	}
}

func TestSearch_EmbedderRedactsBeforeModel(t *testing.T) {
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
	_ = db.Close()

	st, err := knowledge.Open([]string{dir}, true)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	fake := &knowledge.FakeEmbedder{
		Name:    "fake-test",
		Default: []float32{1, 0, 0},
	}
	st.Embedder = fake
	secret := "password=fake-password-for-tests-only zfs readonly"
	if _, err := st.Search(secret, 4); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(fake.LastText, "fake-password-for-tests-only") {
		t.Fatalf("secret leaked into embedder: %q", fake.LastText)
	}
	if !strings.Contains(redact.String(secret), "[REDACTED]") {
		t.Fatal("fixture must still redact")
	}
}

func TestSearch_NoEmbedderLeavesFTSWhenRowsExist(t *testing.T) {
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
	_ = db.Close()
	st, err := knowledge.Open([]string{dir}, true)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	hits, err := st.Search("boot environment", 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) < 1 || hits[0].Title != "List, activate, or roll back a ZFS boot environment" {
		t.Fatalf("without embedder, FTS only: %#v", hits)
	}
}

func TestVecSQL_DistanceMatchesExtension(t *testing.T) {
	db, err := sql.Open("sqlite", "file:vecmatch?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var ver string
	if err := db.QueryRow(`SELECT vec_version()`).Scan(&ver); err != nil {
		t.Fatalf("sqlite-vec must register: %v", err)
	}
	if !strings.HasPrefix(ver, "v") {
		t.Fatalf("vec_version %q", ver)
	}
	a := knowledge.PackF32([]float32{1, 0, 0})
	b := knowledge.PackF32([]float32{0.99, 0.01, 0})
	var dist float64
	if err := db.QueryRow(`SELECT vec_distance_cosine(?, ?)`, a, b).Scan(&dist); err != nil {
		t.Fatal(err)
	}
	want := knowledge.DistanceCosine([]float32{1, 0, 0}, []float32{0.99, 0.01, 0})
	if diff := dist - want; diff > 1e-5 || diff < -1e-5 {
		t.Fatalf("go cosine %v vs sqlite-vec %v", want, dist)
	}
}

func TestOpenRW_FillThenROSearch(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, knowledge.DBName)
	if err := knowledge.CreatePlaybookTestDB(p); err != nil {
		t.Fatal(err)
	}
	rw, err := knowledge.OpenRW(p)
	if err != nil {
		t.Fatal(err)
	}
	fake := &knowledge.FakeEmbedder{
		Name: "fake-test",
		ByText: map[string][]float32{
			"remount": {1, 0, 0},
			"bectl":   {0, 1, 0},
			"cheat":   {0, 0, 1},
		},
		Default: []float32{0, 0, 1},
	}
	if err := rw.FillEmbeddings(context.Background(), fake); err != nil {
		t.Fatal(err)
	}
	if err := rw.Close(); err != nil {
		t.Fatal(err)
	}

	ro, err := knowledge.Open([]string{dir}, true)
	if err != nil {
		t.Fatal(err)
	}
	defer ro.Close()
	if err := ro.FillEmbeddings(context.Background(), fake); err == nil {
		t.Fatal("consult RO open must refuse embedding writes")
	}
	ro.QueryVec = []float32{1, 0, 0}
	hits, err := ro.Search("boot environment", 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) < 1 || hits[0].Title != knowledge.RemountPlaybookTitle {
		t.Fatalf("runtime-filled vectors must rank remount: %#v", hits)
	}
}

func TestFakeEmbedder_ErrorAndEmpty(t *testing.T) {
	f := &knowledge.FakeEmbedder{Err: context.Canceled, Default: []float32{1}}
	if _, err := f.Embed(context.Background(), "x"); err == nil {
		t.Fatal("expected embed error")
	}
	if f.Model() != "" {
		t.Fatal(f.Model())
	}
	f.Err = nil
	f.Name = "fake-test"
	got, err := f.Embed(context.Background(), "x")
	if err != nil || len(got) != 1 {
		t.Fatalf("%v %v", got, err)
	}
	var none *knowledge.FakeEmbedder
	if none.Model() != "" {
		t.Fatal("nil model")
	}
	if _, err := none.Embed(context.Background(), "x"); err == nil {
		t.Fatal("nil embedder")
	}
}

func TestPackF32_OddBlobAndMismatchedCosine(t *testing.T) {
	if knowledge.UnpackF32([]byte{1, 2, 3}) != nil {
		t.Fatal("odd length")
	}
	if knowledge.UnpackF32(nil) != nil {
		t.Fatal("empty")
	}
	if knowledge.DistanceCosine([]float32{1}, []float32{1, 2}) != 2 {
		t.Fatal("mismatch")
	}
	if knowledge.DistanceCosine(nil, nil) != 2 {
		t.Fatal("empty cosine")
	}
	if knowledge.DistanceCosine([]float32{0, 0}, []float32{0, 0}) != 2 {
		t.Fatal("zero vectors")
	}
}

func TestOpenRW_EmptyPathAndFillRefuse(t *testing.T) {
	if _, err := knowledge.OpenRW(""); err == nil {
		t.Fatal("empty path")
	}
	if err := knowledge.InsertEmbedding(nil, "playbooks", "x", "m", []float32{1}); err == nil {
		t.Fatal("nil db")
	}
	var st *knowledge.Store
	if err := st.FillEmbeddings(context.Background(), &knowledge.FakeEmbedder{}); err == nil {
		t.Fatal("nil store")
	}
	dir := t.TempDir()
	p := filepath.Join(dir, knowledge.DBName)
	if err := knowledge.CreatePlaybookTestDB(p); err != nil {
		t.Fatal(err)
	}
	rw, err := knowledge.OpenRW(p)
	if err != nil {
		t.Fatal(err)
	}
	defer rw.Close()
	if err := rw.FillEmbeddings(context.Background(), nil); err == nil {
		t.Fatal("nil embedder")
	}
	min := int64(1 << 40)
	rw.RAMMin = &min
	rw.Headroom = headroom.Snapshot{RAMFreeBytes: 1}
	if err := rw.FillEmbeddings(context.Background(), &knowledge.FakeEmbedder{Name: "fake-test", Default: []float32{1, 0, 0}}); err == nil {
		t.Fatal("low RAM fill")
	}
}

func TestSearch_EmbedderErrorFallsBackToFTS(t *testing.T) {
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
	_ = db.Close()
	st, err := knowledge.Open([]string{dir}, true)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	st.Embedder = &knowledge.FakeEmbedder{Err: context.Canceled}
	hits, err := st.Search("boot environment", 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) < 1 || hits[0].Title != "List, activate, or roll back a ZFS boot environment" {
		t.Fatalf("embed error must FTS: %#v", hits)
	}
}

func TestSearch_GoCosineWhenVecFlagOff(t *testing.T) {
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
	if err := knowledge.InsertEmbedding(db, "documents", "zfs-emergency", "fake-test", []float32{0, 0, 1}); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	st, err := knowledge.Open([]string{dir}, true)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	st.Vec = false
	st.QueryVec = []float32{1, 0, 0}
	hits, err := st.Search("boot environment", 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) < 1 || hits[0].Title != knowledge.RemountPlaybookTitle {
		t.Fatalf("in-process cosine must still rank remount: %#v", hits)
	}
}

func TestFillEmbeddings_RedactsChunkSecrets(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, knowledge.DBName)
	if err := knowledge.CreatePlaybookTestDB(p); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO documents(id, title, category, body) VALUES ('secret-doc', 'Keys', 'docs', 'password=fake-password-for-tests-only')`); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	rw, err := knowledge.OpenRW(p)
	if err != nil {
		t.Fatal(err)
	}
	defer rw.Close()
	fake := &knowledge.FakeEmbedder{Name: "fake-test", Default: []float32{0, 0, 1}}
	if err := rw.FillEmbeddings(nil, fake); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(fake.LastText, "fake-password-for-tests-only") {
		t.Fatalf("secret leaked into fill embedder: %q", fake.LastText)
	}
}
