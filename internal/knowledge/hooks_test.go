// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package knowledge

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
)

func TestFileDSN_AbsFallback(t *testing.T) {
	old := absPath
	absPath = func(string) (string, error) { return "", errors.New("abs fail") }
	t.Cleanup(func() { absPath = old })
	dsn := fileDSN("relative.sqlite", true)
	if dsn == "" {
		t.Fatal("empty dsn")
	}
}

func TestSafeIdentAndTableColumnExists(t *testing.T) {
	if safeIdent("") || safeIdent("play books") || safeIdent("foo;bar") {
		t.Fatal("unsafe identifiers must be rejected")
	}
	if !safeIdent("playbooks") || !safeIdent("commands") {
		t.Fatal("safe identifiers")
	}
	if tableColumnExists(nil, "playbooks", "commands") {
		t.Fatal("nil db")
	}
	if tableColumnExists(nil, "playbooks;drop", "commands") {
		t.Fatal("unsafe table")
	}
	dir := t.TempDir()
	if err := CreatePlaybookTestDB(filepath.Join(dir, DBName)); err != nil {
		t.Fatal(err)
	}
	st, err := Open([]string{dir}, true)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if !tableColumnExists(st.DB, "playbooks", "commands") {
		t.Fatal("commands column")
	}
	if tableColumnExists(st.DB, "playbooks", "nope") {
		t.Fatal("missing column")
	}
	if tableColumnExists(st.DB, "missing_table", "x") {
		t.Fatal("missing table")
	}
}

func TestOpen_sqlOpenFailsContinues(t *testing.T) {
	dir := t.TempDir()
	if err := CreateTestDB(filepath.Join(dir, DBName)); err != nil {
		t.Fatal(err)
	}
	old := sqlOpen
	sqlOpen = func(string, string) (*sql.DB, error) { return nil, errors.New("open fail") }
	t.Cleanup(func() { sqlOpen = old })
	_, err := Open([]string{dir}, true)
	if err == nil {
		t.Fatal("expected error")
	}
}
