// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package knowledge

import (
	"context"
	"fmt"
	"strings"

	"github.com/revytechinc/hawkeye/internal/headroom"
	"github.com/revytechinc/hawkeye/internal/redact"
)

// Embedder turns redacted text into a FLOAT32 vector. No cloud API.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	Model() string
}

// FakeEmbedder is a test fixture. Keys in ByText match as substrings.
type FakeEmbedder struct {
	Name     string
	ByText   map[string][]float32
	Default  []float32
	Err      error
	LastText string
}

func (f *FakeEmbedder) Model() string {
	if f == nil {
		return ""
	}
	return f.Name
}

func (f *FakeEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	if f == nil {
		return nil, ErrNotFound
	}
	f.LastText = text
	if f.Err != nil {
		return nil, f.Err
	}
	lower := strings.ToLower(text)
	for k, v := range f.ByText {
		if k != "" && strings.Contains(lower, strings.ToLower(k)) {
			return append([]float32(nil), v...), nil
		}
	}
	return append([]float32(nil), f.Default...), nil
}

// OpenRW opens a writable kit for runtime embedding of existing chunks.
// Consult never uses this; diagnose Open stays mode=ro.
func OpenRW(path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, ErrNotFound
	}
	db, err := sqlOpen("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	st := &Store{Path: path, DSN: path, ReadOnly: false, Immutable: false, DB: db}
	if err := st.verifyFTS(); err != nil {
		_ = db.Close()
		return nil, err
	}
	st.FTS = true
	if _, err := db.Exec(embeddingsDDL); err != nil {
		_ = db.Close()
		return nil, err
	}
	st.probeVectors()
	return st, nil
}

// FillEmbeddings embeds playbook and document chunks into embeddings.
// Refuses a read-only consult handle. Skips when RAM is below RAMMin.
func (s *Store) FillEmbeddings(ctx context.Context, emb Embedder) error {
	if s == nil || s.DB == nil {
		return ErrNotFound
	}
	if s.ReadOnly {
		return fmt.Errorf("embeddings fill refused: store is read-only")
	}
	if emb == nil {
		return fmt.Errorf("local embedder is not configured")
	}
	if err := headroom.Allow(headroom.Job{NeedRAM: true}, s.Headroom, s.RAMMin, nil, nil, nil); err != nil {
		return err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if _, err := s.DB.Exec(embeddingsDDL); err != nil {
		return err
	}
	model := strings.TrimSpace(emb.Model())
	if model == "" {
		model = "local"
	}
	chunks, err := s.chunkTexts()
	if err != nil {
		return err
	}
	for _, c := range chunks {
		text := redact.String(c.text)
		if strings.TrimSpace(text) == "" {
			continue
		}
		vec, err := emb.Embed(ctx, text)
		if err != nil {
			return err
		}
		if len(vec) == 0 {
			continue
		}
		if err := InsertEmbedding(s.DB, c.table, c.id, model, vec); err != nil {
			return err
		}
	}
	s.probeVectors()
	return nil
}

type chunk struct {
	table, id, text string
}

func (s *Store) chunkTexts() ([]chunk, error) {
	var out []chunk
	if tableExists(s.DB, "playbooks") {
		rows, err := s.DB.Query(`SELECT id, title, COALESCE(when_to_use, ''), body FROM playbooks`)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var id, title, when, body string
			if err := rows.Scan(&id, &title, &when, &body); err != nil {
				_ = rows.Close()
				return nil, err
			}
			out = append(out, chunk{table: "playbooks", id: id, text: title + "\n" + when + "\n" + body})
		}
		err = rows.Err()
		_ = rows.Close()
		if err != nil {
			return nil, err
		}
	}
	if tableExists(s.DB, "documents") {
		rows, err := s.DB.Query(`SELECT id, title, body FROM documents`)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var id, title, body string
			if err := rows.Scan(&id, &title, &body); err != nil {
				_ = rows.Close()
				return nil, err
			}
			out = append(out, chunk{table: "documents", id: id, text: title + "\n" + body})
		}
		err = rows.Err()
		_ = rows.Close()
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}
