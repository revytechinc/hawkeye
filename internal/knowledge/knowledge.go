// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package knowledge

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

var ErrNotFound = errors.New("knowledge database not found")

// Test hooks. Production uses sql.Open and filepath.Abs.
var (
	sqlOpen = sql.Open
	absPath = filepath.Abs
)

const DBName = "knowledge.sqlite"

type Store struct {
	Path      string
	DSN       string
	Immutable bool
	ReadOnly  bool
	FTS       bool
	DB        *sql.DB
}

type Hit struct {
	Title string
	Body  string
	Tags  string
	Rank  float64
}

func SearchPaths(xdgDataHome, home string) []string {
	out := []string{"/boot/hawkeye", "/usr/local/share/hawkeye"}
	if xdgDataHome != "" {
		out = append(out, filepath.Join(xdgDataHome, "hawkeye"))
	} else if home != "" {
		out = append(out, filepath.Join(home, ".local", "share", "hawkeye"))
	}
	return out
}

func fileDSN(path string, immutable bool) string {
	abs, err := absPath(path)
	if err != nil {
		abs = path
	}
	u := url.URL{Scheme: "file", Path: filepath.ToSlash(abs)}
	q := url.Values{}
	q.Set("mode", "ro")
	if immutable {
		q.Set("immutable", "1")
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func Open(paths []string, rootRO bool) (*Store, error) {
	var last error
	for _, dir := range paths {
		p := dir
		if filepath.Base(dir) != DBName {
			p = filepath.Join(dir, DBName)
		}
		if _, err := os.Stat(p); err != nil {
			last = err
			continue
		}
		dsn := fileDSN(p, rootRO)
		db, err := sqlOpen("sqlite", dsn)
		if err != nil {
			last = err
			continue
		}
		if err := db.Ping(); err != nil {
			_ = db.Close()
			last = err
			continue
		}
		st := &Store{Path: p, DSN: dsn, Immutable: rootRO, ReadOnly: true, DB: db}
		if err := st.verifyFTS(); err != nil {
			_ = db.Close()
			last = err
			continue
		}
		st.FTS = true
		return st, nil
	}
	if last == nil {
		last = ErrNotFound
	}
	return nil, fmt.Errorf("%w: %v", ErrNotFound, last)
}

func (s *Store) verifyFTS() error {
	var name string
	err := s.DB.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='knowledge_fts'`).Scan(&name)
	if err != nil {
		return fmt.Errorf("fts table missing: %w", err)
	}
	return nil
}

func (s *Store) Close() error {
	if s != nil && s.DB != nil {
		return s.DB.Close()
	}
	return nil
}

func (s *Store) Search(query string, limit int) ([]Hit, error) {
	if s == nil || s.DB == nil {
		return nil, ErrNotFound
	}
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.DB.Query(
		`SELECT title, body, tags FROM knowledge_fts WHERE knowledge_fts MATCH ? LIMIT ?`,
		query, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var hits []Hit
	for rows.Next() {
		var h Hit
		if err := rows.Scan(&h.Title, &h.Body, &h.Tags); err != nil {
			return nil, err
		}
		hits = append(hits, h)
	}
	return hits, rows.Err()
}

func CreateTestDB(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE VIRTUAL TABLE knowledge_fts USING fts5(title, body, tags)`); err != nil {
		return err
	}
	_, err = db.Exec(
		`INSERT INTO knowledge_fts(title, body, tags) VALUES (?, ?, ?)`,
		"ZFS readonly pool",
		"If the root pool is imported readonly, first skill is unlock-rw, not pkg.",
		"zfs rescue tier0",
	)
	return err
}
