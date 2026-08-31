// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package consult

import (
	"strconv"
	"strings"

	"github.com/revytechinc/hawkeye/internal/apply"
	"github.com/revytechinc/hawkeye/internal/knowledge"
	"github.com/revytechinc/hawkeye/internal/probe"
	"github.com/revytechinc/hawkeye/internal/redact"
)

// Plan turns the lead consult hit into apply steps using stored playbook
// commands (JSON column or fenced body). It does not invent echo <query>.
// When the root is read-only and no commands are stored, the first skill
// remains unlock-rw.
func (r Result) Plan(snap probe.Snapshot) apply.Plan {
	p := apply.Plan{
		ID:      "consult-plan",
		Source:  "knowledge",
		Summary: redact.String(r.Query),
		Steps:   []apply.Step{},
	}
	if len(r.Hits) > 0 {
		lead := r.Hits[leadIndex(r.Query, r.Hits)]
		cmds := CommandLines(lead)
		if len(cmds) > 0 {
			if title := strings.TrimSpace(lead.Title); title != "" {
				p.Summary = redact.String(title)
			}
			p.Steps = stepsFromCommands(cmds)
			return p
		}
	}
	if snap.RootRO {
		p.Summary = "root is read-only; first skill is unlock-rw, not pkg"
		p.Steps = []apply.Step{{
			ID:         "1",
			Action:     "unlock-rw",
			Argv:       []string{"zfs", "set", "readonly=off", "<rootpool>"},
			Privileged: true,
		}}
	}
	return p
}

// CommandLines returns stored playbook command lines. The JSON commands
// column wins; otherwise fenced ``` / ~~~ body lines. Prose is not argv.
func CommandLines(h knowledge.Hit) []string {
	if cmds := cleanCommandLines(h.Commands); len(cmds) > 0 {
		return cmds
	}
	raw := fencedCommands(h.Body)
	if raw == "" {
		return nil
	}
	return cleanCommandLines(strings.Split(raw, "\n"))
}

func cleanCommandLines(in []string) []string {
	var out []string
	for _, line := range in {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, redact.String(line))
	}
	return out
}

func stepsFromCommands(cmds []string) []apply.Step {
	steps := make([]apply.Step, 0, len(cmds))
	for i, line := range cmds {
		action := "command"
		if fields := strings.Fields(line); len(fields) > 0 {
			action = fields[0]
		}
		steps = append(steps, apply.Step{
			ID:         strconv.Itoa(i + 1),
			Action:     action,
			Argv:       []string{line},
			Privileged: true,
		})
	}
	return steps
}
