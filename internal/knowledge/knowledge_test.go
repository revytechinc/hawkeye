// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package knowledge_test

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/revytechinc/hawkeye/internal/knowledge"

	_ "modernc.org/sqlite"
)

func TestSearchPaths_OrderBootThenLocalThenXDG(t *testing.T) {
	paths := knowledge.SearchPaths("/xdg/share", "/home/operator")
	if len(paths) < 3 {
		t.Fatalf("paths %v", paths)
	}
	if knowledge.RescueDir != "/boot/hawkeye" || knowledge.SystemDir != "/usr/local/share/hawkeye" {
		t.Fatalf("rescue/system constants: %q %q", knowledge.RescueDir, knowledge.SystemDir)
	}
	if paths[0] != knowledge.RescueDir {
		t.Fatalf("first path must be /boot/hawkeye, got %q", paths[0])
	}
	if paths[1] != knowledge.SystemDir {
		t.Fatalf("second path must be system share, got %q", paths[1])
	}
	if knowledge.RescueBinary != "/rescue/hawkeye" {
		t.Fatalf("rescue binary %q", knowledge.RescueBinary)
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

func TestOpen_HarvestSchemaFTS(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "knowledge.sqlite")
	db, err := sql.Open("sqlite", p)
	if err != nil {
		t.Fatal(err)
	}
	stmts := []string{
		`CREATE TABLE documents (
			rowid INTEGER PRIMARY KEY,
			id TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			category TEXT,
			body TEXT NOT NULL
		)`,
		`CREATE VIRTUAL TABLE documents_fts USING fts5(
			title, category, body,
			content='documents', content_rowid='rowid', tokenize='unicode61'
		)`,
		`CREATE TABLE playbooks (
			rowid INTEGER PRIMARY KEY,
			id TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			when_to_use TEXT NOT NULL,
			body TEXT NOT NULL
		)`,
		`CREATE VIRTUAL TABLE playbooks_fts USING fts5(
			title, when_to_use, body,
			content='playbooks', content_rowid='rowid', tokenize='unicode61'
		)`,
		`INSERT INTO documents(id, title, category, body) VALUES (
			'zfs-emergency', 'ZFS emergency cheat sheet', 'docs',
			'ZFS root mounted read-only after boot. Remount with zfs-remount-rw.'
		)`,
		`INSERT INTO documents_fts(rowid, title, category, body)
			SELECT rowid, title, category, body FROM documents`,
		`INSERT INTO playbooks(id, title, when_to_use, body) VALUES (
			'zfs-remount-rw', 'Remount ZFS root read-write',
			'Root is a ZFS dataset and is mounted read-only after boot.',
			'If the root pool is imported readonly, remount ZFS read-write. First skill is unlock-rw, not pkg.'
		)`,
		`INSERT INTO playbooks_fts(rowid, title, when_to_use, body)
			SELECT rowid, title, when_to_use, body FROM playbooks`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("%v: %s", err, s)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	st, err := knowledge.Open([]string{dir}, true)
	if err != nil {
		t.Fatalf("harvest schema (documents_fts + playbooks_fts, no knowledge_fts) must open: %v", err)
	}
	defer st.Close()
	if !st.FTS {
		t.Fatal("FTS is required")
	}
	hits, err := st.Search("zfs readonly", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) < 1 {
		t.Fatal("expected FTS hit from harvest playbooks_fts/documents_fts")
	}
	joined := ""
	for _, h := range hits {
		joined += h.Title + " " + h.Body + " "
	}
	if !strings.Contains(strings.ToLower(joined), "zfs") {
		t.Fatalf("hits %#v", hits)
	}
	hyphenHits, err := st.Search("ZFS root is read-only after boot", 10)
	if err != nil {
		t.Fatalf("hyphenated consult query must not fail MATCH: %v", err)
	}
	if len(hyphenHits) < 1 {
		t.Fatal("expected hit for consult query with hyphen")
	}
}

func TestSearch_PlaybookCommandsFromStore(t *testing.T) {
	dir := t.TempDir()
	if err := knowledge.CreatePlaybookTestDB(filepath.Join(dir, "knowledge.sqlite")); err != nil {
		t.Fatal(err)
	}
	st, err := knowledge.Open([]string{dir}, true)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	hits, err := st.Search("ZFS root is read-only after boot", 8)
	if err != nil {
		t.Fatal(err)
	}
	var remount *knowledge.Hit
	for i := range hits {
		if hits[i].Title == knowledge.RemountPlaybookTitle {
			remount = &hits[i]
			break
		}
	}
	if remount == nil {
		t.Fatalf("remount playbook missing: %#v", hits)
	}
	want := knowledge.RemountPlaybookCommands()
	if strings.Join(remount.Commands, "\n") != strings.Join(want, "\n") {
		t.Fatalf("stored commands missing from hit: %#v", remount.Commands)
	}
}
