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
	"strings"
	"unicode"

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
	legacyFTS bool
	docsFTS   bool
	playFTS   bool
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

func tableExists(db *sql.DB, name string) bool {
	var got string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, name).Scan(&got)
	return err == nil && got == name
}

func (s *Store) verifyFTS() error {
	s.legacyFTS = tableExists(s.DB, "knowledge_fts")
	s.docsFTS = tableExists(s.DB, "documents_fts")
	s.playFTS = tableExists(s.DB, "playbooks_fts")
	if s.legacyFTS || s.docsFTS || s.playFTS {
		return nil
	}
	return fmt.Errorf("fts table missing: %w", sql.ErrNoRows)
}

func (s *Store) Close() error {
	if s != nil && s.DB != nil {
		return s.DB.Close()
	}
	return nil
}

// ftsMatchQuery turns a natural-language consult string into an FTS5 MATCH
// expression. Hyphens are NOT operators in FTS5 ("read-only" => no such column).
func ftsMatchQuery(q string) string {
	var b strings.Builder
	var tok []rune
	flush := func() {
		if len(tok) == 0 {
			return
		}
		s := string(tok)
		tok = tok[:0]
		switch strings.ToUpper(s) {
		case "AND", "OR", "NOT", "NEAR":
			return
		}
		if b.Len() > 0 {
			b.WriteString(" OR ")
		}
		b.WriteByte('"')
		b.WriteString(s)
		b.WriteByte('"')
	}
	for _, r := range q {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			tok = append(tok, r)
			continue
		}
		flush()
	}
	flush()
	return b.String()
}

func (s *Store) Search(query string, limit int) ([]Hit, error) {
	if s == nil || s.DB == nil {
		return nil, ErrNotFound
	}
	if limit <= 0 {
		limit = 10
	}
	match := ftsMatchQuery(query)
	if match == "" {
		match = strings.TrimSpace(query)
	}
	if match == "" {
		return nil, nil
	}
	if s.legacyFTS {
		return s.searchTable(`SELECT title, body, tags, 0 FROM knowledge_fts WHERE knowledge_fts MATCH ? LIMIT ?`, match, limit)
	}
	var parts []string
	var args []any
	if s.playFTS {
		parts = append(parts, `SELECT title, body, when_to_use AS tags, bm25(playbooks_fts) AS rank, 0 AS pri FROM playbooks_fts WHERE playbooks_fts MATCH ?`)
		args = append(args, match)
	}
	if s.docsFTS {
		parts = append(parts, `SELECT title, body, category AS tags, bm25(documents_fts) AS rank, 1 AS pri FROM documents_fts WHERE documents_fts MATCH ?`)
		args = append(args, match)
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("fts table missing")
	}
	q := `SELECT title, body, tags, rank FROM (` + strings.Join(parts, " UNION ALL ") + `) ORDER BY pri ASC, rank ASC LIMIT ?`
	args = append(args, limit)
	return s.searchTable(q, args...)
}

func (s *Store) searchTable(q string, args ...any) ([]Hit, error) {
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var hits []Hit
	for rows.Next() {
		var h Hit
		if err := rows.Scan(&h.Title, &h.Body, &h.Tags, &h.Rank); err != nil {
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
