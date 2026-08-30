// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package knowledge_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/revytechinc/hawkeye/internal/knowledge"
)

func TestSearchPaths_OrderBootThenLocalThenXDG(t *testing.T) {
	paths := knowledge.SearchPaths("/xdg/share", "/home/operator")
	if len(paths) < 3 {
		t.Fatalf("paths %v", paths)
	}
	if paths[0] != "/boot/hawkeye" {
		t.Fatalf("first path must be /boot/hawkeye, got %q", paths[0])
	}
	if paths[1] != "/usr/local/share/hawkeye" {
		t.Fatalf("second path must be system share, got %q", paths[1])
	}
	joined := strings.Join(paths, "|")
	if !strings.Contains(joined, "/xdg/share/hawkeye") {
		t.Fatalf("missing XDG path: %v", paths)
	}
}

func TestOpen_ReadOnlyImmutableWhenRootRO(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "knowledge.sqlite")
	if err := knowledge.CreateTestDB(dbPath); err != nil {
		t.Fatal(err)
	}
	st, err := knowledge.Open([]string{dir}, true)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if !st.ReadOnly || !st.Immutable {
		t.Fatalf("expected RO+immutable, got %+v", st)
	}
	if !st.FTS {
		t.Fatal("FTS is required")
	}
	if !strings.Contains(st.DSN, "mode=ro") {
		t.Fatalf("dsn %q missing mode=ro", st.DSN)
	}
	if !strings.Contains(st.DSN, "immutable=1") {
		t.Fatalf("dsn %q missing immutable=1", st.DSN)
	}
	_, err = st.DB.Exec(`INSERT INTO knowledge_fts(title, body, tags) VALUES ('x','y','z')`)
	if err == nil {
		t.Fatal("RO open must refuse writes")
	}
}

func TestOpen_ConsultSearchNeedsNoWrite(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "knowledge.sqlite")
	if err := knowledge.CreateTestDB(dbPath); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	st, err := knowledge.Open([]string{dir}, true)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	hits, err := st.Search("zfs readonly", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) < 1 {
		t.Fatal("expected FTS hit")
	}
	after, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before) {
		t.Fatalf("consult wrote files: before=%d after=%d", len(before), len(after))
	}
}

func TestOpen_MissingReturnsNotFound(t *testing.T) {
	_, err := knowledge.Open([]string{t.TempDir()}, true)
	if !errors.Is(err, knowledge.ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
}
