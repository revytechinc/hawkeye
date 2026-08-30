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
