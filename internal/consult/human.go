// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package consult

import (
	"fmt"
	"strings"

	"github.com/revytechinc/hawkeye/internal/knowledge"
	"github.com/revytechinc/hawkeye/internal/redact"
)

// Human returns the operator TTY session for a consult: the query, then
// ranked hits as title, short summary, and stored playbook text/commands.
// JSON field names are not printed. Secrets are redacted.
func (r Result) Human() string {
	var b strings.Builder
	q := strings.TrimSpace(r.Query)
	if q == "" {
		q = "(empty query)"
	}
	fmt.Fprintf(&b, "consult  %s\n", q)
	if r.FirstSkill != "" {
		fmt.Fprintf(&b, "tier %d — first skill: %s\n", r.Tier, r.FirstSkill)
	} else {
		fmt.Fprintf(&b, "tier %d\n", r.Tier)
	}
	for _, n := range r.Notes {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		fmt.Fprintf(&b, "%s\n", n)
	}
	if len(r.Hits) == 0 {
		b.WriteString("\nno knowledge hits\n")
	} else {
		b.WriteByte('\n')
		for i, h := range r.Hits {
			writeHit(&b, i+1, h)
			if i+1 < len(r.Hits) {
				b.WriteByte('\n')
			}
		}
	}
	if r.LLM != nil {
		if text := strings.TrimSpace(r.LLM.Text); text != "" {
			b.WriteString("\nhunch\n")
			b.WriteString(indentBlock(text, "   "))
			b.WriteByte('\n')
		}
	}
	return redact.String(b.String())
}

func writeHit(b *strings.Builder, n int, h knowledge.Hit) {
	title := strings.TrimSpace(h.Title)
	if title == "" {
		title = "untitled"
	}
	fmt.Fprintf(b, "%d. %s\n", n, title)
	if sum := strings.TrimSpace(h.Tags); sum != "" {
		fmt.Fprintf(b, "   %s\n", sum)
	}
	body := unwrapFences(h.Body)
	body = dropDuplicateHeading(title, body)
	body = strings.TrimSpace(body)
	if body == "" {
		return
	}
	b.WriteByte('\n')
	b.WriteString(indentBlock(body, "   "))
	b.WriteByte('\n')
}

func unwrapFences(body string) string {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	lines := strings.Split(body, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "```") || strings.HasPrefix(trim, "~~~") {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func dropDuplicateHeading(title, body string) string {
	body = strings.TrimLeft(body, "\n")
	if body == "" {
		return body
	}
	rest := body
	nl := strings.IndexByte(body, '\n')
	first := body
	if nl >= 0 {
		first = body[:nl]
		rest = body[nl+1:]
	} else {
		rest = ""
	}
	heading := strings.TrimSpace(first)
	heading = strings.TrimLeft(heading, "#")
	heading = strings.TrimSpace(heading)
	if heading != title {
		return body
	}
	return rest
}

func indentBlock(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if line == "" {
			continue
		}
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}
