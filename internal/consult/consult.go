// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package consult

import (
	"context"
	"encoding/json"

	"github.com/revytechinc/hawkeye/internal/knowledge"
	"github.com/revytechinc/hawkeye/internal/llm"
	"github.com/revytechinc/hawkeye/internal/probe"
	"github.com/revytechinc/hawkeye/internal/redact"
)

type Result struct {
	Query      string          `json:"query"`
	Tier       int             `json:"tier"`
	FirstSkill string          `json:"first_skill,omitempty"`
	Hits       []knowledge.Hit `json:"hits"`
	Notes      []string        `json:"notes,omitempty"`
	LLM        *llm.Response   `json:"llm,omitempty"`
}

func Run(query string, snap probe.Snapshot, store *knowledge.Store, completer llm.Completer) (Result, error) {
	query = redact.String(query)
	r := Result{
		Query:      query,
		Tier:       snap.Tier,
		FirstSkill: snap.FirstSkill(),
	}
	if snap.RootRO {
		r.Notes = append(r.Notes, "root is read-only; consult does not write; first skill is unlock-rw, not pkg")
	}
	if store != nil {
		hits, err := store.Search(query, 8)
		if err != nil {
			r.Notes = append(r.Notes, "knowledge search error: "+err.Error())
		} else {
			r.Hits = hits
		}
	} else {
		r.Notes = append(r.Notes, "knowledge store unavailable; FTS skipped")
	}
	if completer != nil && snap.Tier >= 1 {
		resp, err := completer.Complete(context.Background(), llm.Request{Prompt: query, NeedGPU: false, NeedRAM: true})
		if err == nil {
			r.LLM = &resp
		} else {
			r.Notes = append(r.Notes, "llm skipped: "+err.Error())
		}
	} else if snap.Tier == 0 {
		r.Notes = append(r.Notes, "tier 0: FTS only, LLM not required")
	}
	return r, nil
}

func (r Result) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
