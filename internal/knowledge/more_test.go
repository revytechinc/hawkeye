// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package knowledge_test

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/revytechinc/hawkeye/internal/knowledge"
	_ "modernc.org/sqlite"
)

func TestSearchPaths_HomeFallback(t *testing.T) {
	paths := knowledge.SearchPaths("", "/home/operator")
	found := false
	for _, p := range paths {
		if p == "/home/operator/.local/share/hawkeye" {
			found = true
		}
	}
	if !found {
		t.Fatal(paths)
	}
}

func TestOpen_FilePathAndNoImmutableWhenWritable(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "knowledge.sqlite")
	if err := knowledge.CreateTestDB(db); err != nil {
		t.Fatal(err)
	}
	st, err := knowledge.Open([]string{db}, false)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if !st.ReadOnly {
		t.Fatal("consult must still be read-only")
	}
	if st.Immutable {
		t.Fatal("writable root should not set immutable=1")
	}
	if _, err := st.Search("readonly", 0); err != nil {
		t.Fatal(err)
	}
}

func TestOpen_MissingFTS(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "knowledge.sqlite")
	db, err := sql.Open("sqlite", p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE not_fts (x TEXT)`); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	_, err = knowledge.Open([]string{dir}, true)
	if err == nil {
		t.Fatal("expected FTS missing")
	}
}

func TestSearchAndCloseNil(t *testing.T) {
	var st *knowledge.Store
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Search("x", 1); err == nil {
		t.Fatal("expected error")
	}
}
