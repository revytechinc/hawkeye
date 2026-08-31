// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package knowledge

import (
	"context"
	"database/sql"
	"encoding/binary"
	"math"
	"strings"

	"github.com/revytechinc/hawkeye/internal/headroom"
	"github.com/revytechinc/hawkeye/internal/redact"
)

const embeddingsDDL = `CREATE TABLE IF NOT EXISTS embeddings (
	id INTEGER PRIMARY KEY,
	target_table TEXT NOT NULL CHECK (target_table IN ('documents', 'playbooks')),
	target_id TEXT NOT NULL,
	model TEXT NOT NULL,
	dim INTEGER NOT NULL CHECK (dim > 0),
	vector BLOB,
	UNIQUE (target_table, target_id, model)
)`

// PackF32 encodes a little-endian FLOAT32 blob (sqlite-vec / hawkeye-data).
func PackF32(v []float32) []byte {
	b := make([]byte, 4*len(v))
	for i, x := range v {
		binary.LittleEndian.PutUint32(b[i*4:], math.Float32bits(x))
	}
	return b
}

// UnpackF32 decodes a little-endian FLOAT32 blob.
func UnpackF32(b []byte) []float32 {
	if len(b) < 4 || len(b)%4 != 0 {
		return nil
	}
	out := make([]float32, len(b)/4)
	for i := range out {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return out
}

// DistanceCosine is sqlite-vec vec_distance_cosine: 1 - cos similarity.
func DistanceCosine(a, b []float32) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 2
	}
	var dot, na, nb float64
	for i := range a {
		fa := float64(a[i])
		fb := float64(b[i])
		dot += fa * fb
		na += fa * fa
		nb += fb * fb
	}
	if na == 0 || nb == 0 {
		return 2
	}
	return 1 - dot/(math.Sqrt(na)*math.Sqrt(nb))
}

func (s *Store) probeVectors() {
	if s == nil || s.DB == nil {
		return
	}
	s.Vec = vecAvailable(s.DB)
	s.hasEmb = embeddingRows(s.DB) > 0
}

func vecAvailable(db *sql.DB) bool {
	if db == nil {
		return false
	}
	var ver string
	if err := db.QueryRow(`SELECT vec_version()`).Scan(&ver); err != nil {
		return false
	}
	return strings.TrimSpace(ver) != ""
}

func embeddingRows(db *sql.DB) int {
	if db == nil || !tableExists(db, "embeddings") {
		return 0
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM embeddings WHERE vector IS NOT NULL`).Scan(&n); err != nil {
		return 0
	}
	return n
}

func (s *Store) useVectors() bool {
	if s == nil || s.DB == nil || !s.hasEmb {
		return false
	}
	if len(s.QueryVec) == 0 && s.Embedder == nil {
		return false
	}
	if err := headroom.Allow(headroom.Job{NeedRAM: true}, s.Headroom, s.RAMMin, nil, nil, nil); err != nil {
		return false
	}
	return true
}

func (s *Store) queryVector(query string) []float32 {
	if len(s.QueryVec) > 0 {
		return s.QueryVec
	}
	if s.Embedder == nil {
		return nil
	}
	vec, err := s.Embedder.Embed(context.Background(), redact.String(query))
	if err != nil || len(vec) == 0 {
		return nil
	}
	return vec
}

func (s *Store) searchVectors(query string, limit int) ([]Hit, bool) {
	if !s.useVectors() {
		return nil, false
	}
	qvec := s.queryVector(query)
	if len(qvec) == 0 {
		return nil, false
	}
	hits, err := s.vectorHits(qvec, limit)
	if err != nil || len(hits) == 0 {
		return nil, false
	}
	return hits, true
}

func (s *Store) vectorHits(qvec []float32, limit int) ([]Hit, error) {
	if s.Vec {
		hits, err := s.vectorHitsSQL(qvec, limit)
		if err == nil && len(hits) > 0 {
			return hits, nil
		}
	}
	return s.vectorHitsGo(qvec, limit)
}

func (s *Store) vectorHitsSQL(qvec []float32, limit int) ([]Hit, error) {
	blob := PackF32(qvec)
	rows, err := s.DB.Query(
		`SELECT target_table, target_id, vec_distance_cosine(vector, ?) AS dist
		 FROM embeddings
		 WHERE vector IS NOT NULL AND dim = ?
		 ORDER BY dist ASC
		 LIMIT ?`,
		blob, len(qvec), limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanVectorRows(rows)
}

func (s *Store) vectorHitsGo(qvec []float32, limit int) ([]Hit, error) {
	rows, err := s.DB.Query(`SELECT target_table, target_id, dim, vector FROM embeddings WHERE vector IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type scored struct {
		table, id string
		dist      float64
	}
	var all []scored
	for rows.Next() {
		var table, id string
		var dim int
		var blob []byte
		if err := rows.Scan(&table, &id, &dim, &blob); err != nil {
			return nil, err
		}
		vec := UnpackF32(blob)
		if len(vec) != len(qvec) || dim != len(qvec) {
			continue
		}
		all = append(all, scored{table: table, id: id, dist: DistanceCosine(qvec, vec)})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := 0; i < len(all); i++ {
		best := i
		for j := i + 1; j < len(all); j++ {
			if all[j].dist < all[best].dist {
				best = j
			}
		}
		all[i], all[best] = all[best], all[i]
	}
	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}
	var hits []Hit
	for _, sc := range all {
		if h, ok := s.lookupTarget(sc.table, sc.id, sc.dist); ok {
			hits = append(hits, h)
		}
	}
	return hits, nil
}

func (s *Store) scanVectorRows(rows *sql.Rows) ([]Hit, error) {
	var hits []Hit
	for rows.Next() {
		var table, id string
		var dist float64
		if err := rows.Scan(&table, &id, &dist); err != nil {
			return nil, err
		}
		if h, ok := s.lookupTarget(table, id, dist); ok {
			hits = append(hits, h)
		}
	}
	return hits, rows.Err()
}

func (s *Store) lookupTarget(table, id string, rank float64) (Hit, bool) {
	if s == nil || s.DB == nil || id == "" {
		return Hit{}, false
	}
	switch table {
	case "playbooks":
		var h Hit
		err := s.DB.QueryRow(`SELECT title, body, when_to_use FROM playbooks WHERE id = ?`, id).Scan(&h.Title, &h.Body, &h.Tags)
		if err != nil {
			return Hit{}, false
		}
		h.Rank = rank
		return h, true
	case "documents":
		var h Hit
		err := s.DB.QueryRow(`SELECT title, body, COALESCE(category, '') FROM documents WHERE id = ?`, id).Scan(&h.Title, &h.Body, &h.Tags)
		if err != nil {
			return Hit{}, false
		}
		h.Rank = rank
		return h, true
	default:
		return Hit{}, false
	}
}

func mergeHits(fts, vec []Hit, limit int) []Hit {
	seen := map[string]bool{}
	out := make([]Hit, 0, limit)
	add := func(h Hit) {
		key := strings.ToLower(strings.TrimSpace(h.Title))
		if key == "" {
			key = strings.ToLower(strings.TrimSpace(h.Body))
		}
		if key == "" || seen[key] {
			return
		}
		seen[key] = true
		out = append(out, h)
	}
	for _, h := range vec {
		add(h)
	}
	for _, h := range fts {
		add(h)
	}
	if limit <= 0 {
		limit = 10
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

// InsertEmbedding writes a tiny FLOAT32 row (tests / runtime fill).
func InsertEmbedding(db *sql.DB, table, id, model string, vec []float32) error {
	if db == nil {
		return ErrNotFound
	}
	if _, err := db.Exec(embeddingsDDL); err != nil {
		return err
	}
	_, err := db.Exec(
		`INSERT OR REPLACE INTO embeddings(target_table, target_id, model, dim, vector) VALUES (?, ?, ?, ?, ?)`,
		table, id, model, len(vec), PackF32(vec),
	)
	return err
}
