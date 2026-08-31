// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package consult

import (
	"strings"
	"unicode"

	"github.com/revytechinc/hawkeye/internal/knowledge"
	"github.com/revytechinc/hawkeye/internal/redact"
)

const (
	indent = "  "
	wrapAt = 72
)

// Human returns the operator TTY view: the most actionable playbook for
// the query (title, stored summary, stored commands), then related titles.
// Query/tier/rank/JSON keys and "llm skipped" are omitted. Secrets are redacted.
func (r Result) Human() string {
	var b strings.Builder
	if len(r.Hits) == 0 {
		b.WriteString("no knowledge hits\n")
	} else {
		lead := leadIndex(r.Query, r.Hits)
		writeLead(&b, r.Hits[lead])
		var also []string
		for i, h := range r.Hits {
			if i == lead {
				continue
			}
			if t := strings.TrimSpace(h.Title); t != "" {
				also = append(also, t)
			}
		}
		if len(also) > 0 {
			b.WriteString("\nalso:\n")
			for _, t := range also {
				b.WriteString(indent)
				b.WriteString(t)
				b.WriteByte('\n')
			}
		}
	}
	if r.LLM != nil {
		if text := strings.TrimSpace(r.LLM.Text); text != "" {
			if b.Len() > 0 && !strings.HasSuffix(b.String(), "\n\n") {
				b.WriteByte('\n')
			}
			b.WriteString(indentBlock(text, indent))
			b.WriteByte('\n')
		}
	}
	return redact.String(b.String())
}

func writeLead(b *strings.Builder, h knowledge.Hit) {
	title := strings.TrimSpace(h.Title)
	if title == "" {
		title = "untitled"
	}
	b.WriteString(title)
	b.WriteByte('\n')
	if sum := proseSummary(h.Tags); sum != "" {
		b.WriteString(indentBlock(wrapWords(sum, wrapAt), indent))
		b.WriteByte('\n')
	}
	body := storedSteps(title, h.Body)
	if body == "" {
		return
	}
	b.WriteByte('\n')
	b.WriteString(indentBlock(body, indent))
	b.WriteByte('\n')
}

func proseSummary(tags string) string {
	s := strings.TrimSpace(tags)
	if len(s) < 24 {
		return ""
	}
	if strings.ContainsAny(s, ".,;:") || strings.Contains(s, " is ") || strings.Contains(s, " when ") {
		return s
	}
	return ""
}

func storedSteps(title, body string) string {
	if cmds := fencedCommands(body); cmds != "" {
		return cmds
	}
	body = unwrapFences(body)
	body = dropDuplicateHeading(title, body)
	return strings.TrimSpace(body)
}

func fencedCommands(body string) string {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	var out []string
	inFence := false
	found := false
	for _, line := range strings.Split(body, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "```") || strings.HasPrefix(trim, "~~~") {
			inFence = !inFence
			if inFence {
				found = true
			}
			continue
		}
		if inFence {
			out = append(out, line)
		}
	}
	if !found {
		return ""
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
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
	nl := strings.IndexByte(body, '\n')
	first := body
	rest := ""
	if nl >= 0 {
		first = body[:nl]
		rest = body[nl+1:]
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

func wrapWords(s string, width int) string {
	if width <= 0 {
		return s
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return ""
	}
	var lines []string
	var cur strings.Builder
	for _, w := range words {
		if cur.Len() == 0 {
			cur.WriteString(w)
			continue
		}
		if cur.Len()+1+len(w) > width {
			lines = append(lines, cur.String())
			cur.Reset()
			cur.WriteString(w)
			continue
		}
		cur.WriteByte(' ')
		cur.WriteString(w)
	}
	if cur.Len() > 0 {
		lines = append(lines, cur.String())
	}
	return strings.Join(lines, "\n")
}

func leadIndex(query string, hits []knowledge.Hit) int {
	if len(hits) == 0 {
		return 0
	}
	q := tokens(query)
	best := 0
	bestScore := actionScore(q, hits[0])
	for i := 1; i < len(hits); i++ {
		s := actionScore(q, hits[i])
		if s > bestScore {
			bestScore = s
			best = i
		}
	}
	return best
}

func actionScore(qtoks []string, h knowledge.Hit) int {
	title := tokens(h.Title)
	tags := tokens(h.Tags)
	score := 3*overlap(qtoks, title) + 2*overlap(qtoks, tags)
	if fencedCommands(h.Body) != "" {
		score += 5
	}
	return score
}

func overlap(q, hay []string) int {
	n := 0
	for _, t := range q {
		for _, h := range hay {
			if h == t || (len(t) >= 4 && strings.Contains(h, t)) {
				n++
				break
			}
		}
	}
	return n
}

var stop = map[string]bool{
	"a": true, "an": true, "the": true, "is": true, "are": true, "was": true,
	"be": true, "to": true, "of": true, "for": true, "and": true, "or": true,
	"vs": true, "versus": true, "then": true, "after": true, "with": true,
	"from": true, "on": true, "in": true, "at": true, "as": true, "by": true,
}

func tokens(s string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(raw string) {
		for _, w := range splitWords(raw) {
			if stop[w] || len(w) < 2 || seen[w] {
				continue
			}
			seen[w] = true
			out = append(out, w)
		}
	}
	add(s)
	add(strings.ReplaceAll(s, "-", ""))
	return out
}

func splitWords(s string) []string {
	s = strings.ToLower(s)
	var out []string
	var tok []rune
	flush := func() {
		if len(tok) == 0 {
			return
		}
		out = append(out, string(tok))
		tok = tok[:0]
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			tok = append(tok, r)
			continue
		}
		flush()
	}
	flush()
	return out
}
