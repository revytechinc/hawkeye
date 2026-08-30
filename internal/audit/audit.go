// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package audit

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/revytechinc/hawkeye/internal/apply"
)

type Event struct {
	Time   string       `json:"time"`
	Actor  apply.Actor  `json:"actor"`
	Mode   string       `json:"mode"`
	PlanID string       `json:"plan_id"`
	Source string       `json:"source"`
	Result apply.Result `json:"result"`
}

type File struct {
	Path string
	Now  func() time.Time
	mu   sync.Mutex
}

func (f *File) Record(plan apply.Plan, mode apply.Mode, actor apply.Actor, result apply.Result) error {
	if f == nil || f.Path == "" {
		return nil
	}
	now := time.Now().UTC()
	if f.Now != nil {
		now = f.Now().UTC()
	}
	ev := Event{
		Time:   now.Format(time.RFC3339),
		Actor:  actor,
		Mode:   mode.String(),
		PlanID: plan.ID,
		Source: plan.Source,
		Result: result,
	}
	b, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	fh, err := os.OpenFile(f.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer fh.Close()
	_, err = fh.Write(append(b, '\n'))
	return err
}

func (f *File) ReadAll() ([]Event, error) {
	b, err := os.ReadFile(f.Path)
	if err != nil {
		return nil, err
	}
	var out []Event
	for _, line := range splitLines(string(b)) {
		if line == "" {
			continue
		}
		var ev Event
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			return nil, err
		}
		out = append(out, ev)
	}
	return out, nil
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
