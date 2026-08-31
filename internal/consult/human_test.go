// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package consult_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/revytechinc/hawkeye/internal/consult"
	"github.com/revytechinc/hawkeye/internal/knowledge"
	"github.com/revytechinc/hawkeye/internal/llm"
	"github.com/revytechinc/hawkeye/internal/probe"
)

func jailOrderHits() []knowledge.Hit {
	return []knowledge.Hit{
		{Title: "List, activate, or roll back a ZFS boot environment", Tags: "Boot environments after an upgrade."},
		{Title: "Single-user versus multi-user", Tags: "Choose a runlevel."},
		{Title: "Import a ZFS pool (readonly first, then unlock)", Tags: "Pool is not imported yet."},
		{
			Title: "Remount ZFS root read-write",
			Tags:  "Root is a ZFS dataset and is mounted read-only (single-user, panic remount, zfs readonly=on, or a readonly pool import).",
			Body: `# Remount ZFS root read-write

Use this when / is ZFS and mount shows read-only.

## Commands

` + "```sh\n" + `export PATH=/rescue:/sbin:/bin:/usr/sbin:/usr/bin
mount -p
zfs set readonly=off "$ROOTDS"
mount -u -o rw /
` + "```\n",
			Rank: 4,
		},
	}
}

func TestHuman_LeadsWithActionablePlaybook(t *testing.T) {
	r := consult.Result{
		Query: "ZFS root is read-only after boot",
		Tier:  1,
		Hits:  jailOrderHits(),
		Notes: []string{"llm skipped: local llm model is not configured"},
	}
	got := r.Human()
	if strings.HasPrefix(strings.TrimSpace(got), "{") {
		t.Fatalf("human output must not be a JSON blob:\n%s", got)
	}
	for _, junk := range []string{
		`"Title"`, `"when_to_use"`, `"score"`, `"query":`, `"hits"`, `"Tags"`, `"Rank"`,
		"llm skipped", "tier ", "consult  ",
	} {
		if strings.Contains(got, junk) {
			t.Fatalf("human TTY leaked %q:\n%s", junk, got)
		}
	}
	lead := strings.TrimSpace(got)
	if !strings.HasPrefix(lead, "Remount ZFS root read-write\n") {
		t.Fatalf("must lead with the actionable playbook, not FTS order:\n%s", got)
	}
	if !strings.Contains(got, "Root is a ZFS dataset and is mounted read-only") {
		t.Fatalf("summary missing:\n%s", got)
	}
	if !strings.Contains(got, `export PATH=/rescue:/sbin:/bin:/usr/sbin:/usr/bin`) {
		t.Fatalf("playbook command missing:\n%s", got)
	}
	if !strings.Contains(got, "mount -p") {
		t.Fatalf("playbook command missing:\n%s", got)
	}
	if strings.Contains(got, "```") || strings.Contains(got, "## Commands") {
		t.Fatalf("markdown chrome leaked:\n%s", got)
	}
	also := strings.Index(got, "also:")
	if also < 0 {
		t.Fatalf("related titles missing:\n%s", got)
	}
	rest := got[also:]
	for _, title := range []string{
		"List, activate, or roll back a ZFS boot environment",
		"Single-user versus multi-user",
		"Import a ZFS pool (readonly first, then unlock)",
	} {
		if !strings.Contains(rest, title) {
			t.Fatalf("also: missing %q:\n%s", title, got)
		}
	}
	if strings.Contains(rest, "Remount ZFS root read-write") {
		t.Fatalf("lead title repeated under also:\n%s", got)
	}
}

func TestHuman_JSONKeepsFTSOrder(t *testing.T) {
	r := consult.Result{Query: "ZFS root is read-only after boot", Hits: jailOrderHits()}
	b, err := r.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(b) {
		t.Fatalf("invalid JSON: %s", b)
	}
	var decoded struct {
		Hits []struct {
			Title string `json:"title"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Hits) < 4 || decoded.Hits[0].Title != "List, activate, or roll back a ZFS boot environment" {
		t.Fatalf("--json must keep FTS order: %#v", decoded.Hits)
	}
	if decoded.Hits[3].Title != "Remount ZFS root read-write" {
		t.Fatalf("remount should stay 4th in JSON: %#v", decoded.Hits)
	}
}

func TestHuman_RedactsSecrets(t *testing.T) {
	r := consult.Result{
		Query: "unlock pool",
		Hits: []knowledge.Hit{{
			Title: "Load keys",
			Body:  "password=fake-password-for-tests-only\n",
		}},
	}
	got := r.Human()
	if strings.Contains(got, "fake-password-for-tests-only") {
		t.Fatalf("secret leaked:\n%s", got)
	}
	if !strings.Contains(got, "[REDACTED]") {
		t.Fatalf("expected redaction marker:\n%s", got)
	}
}

func TestHuman_EmptyHitsNotJSON(t *testing.T) {
	r, err := consult.Run("hello", probe.Snapshot{Tier: 0, RootRO: true}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := r.Human()
	if strings.HasPrefix(strings.TrimSpace(got), "{") {
		t.Fatalf("must not be JSON:\n%s", got)
	}
	if strings.Contains(got, `"query":`) || strings.Contains(got, "llm skipped") {
		t.Fatalf("empty-hit TTY leaked machine chrome:\n%s", got)
	}
}

func TestHuman_IncludesLLMTextWithoutJSONKeys(t *testing.T) {
	r := consult.Result{
		Query: "zpool degraded",
		Tier:  2,
		LLM:   &llm.Response{Text: "check zpool status then replace the faulted vdev", Backend: "llama.cpp"},
	}
	got := r.Human()
	if !strings.Contains(got, "check zpool status then replace the faulted vdev") {
		t.Fatalf("llm text missing:\n%s", got)
	}
	if strings.Contains(got, `"text"`) || strings.Contains(got, `"backend"`) {
		t.Fatalf("llm JSON keys leaked:\n%s", got)
	}
}

func TestHuman_UntitledHit(t *testing.T) {
	r := consult.Result{
		Query: "   ",
		Tier:  2,
		Hits:  []knowledge.Hit{{Title: "", Body: "plain stored body", Tags: ""}},
		Notes: []string{"", "llm skipped: local llm model is not configured"},
	}
	got := r.Human()
	if !strings.Contains(got, "untitled") {
		t.Fatalf("untitled fallback missing:\n%s", got)
	}
	if !strings.Contains(got, "plain stored body") {
		t.Fatalf("body missing:\n%s", got)
	}
	if strings.Contains(got, "llm skipped") || strings.Contains(got, "(empty query)") {
		t.Fatalf("chrome leaked:\n%s", got)
	}
}

func TestHuman_TildeFenceAndNonMatchingHeading(t *testing.T) {
	r := consult.Result{
		Query: "ufs remount",
		Hits: []knowledge.Hit{{
			Title: "Remount UFS",
			Body:  "# Other heading\n\n~~~\nmount -u -o rw /\n~~~\n",
		}},
	}
	got := r.Human()
	if !strings.Contains(got, "mount -u -o rw /") {
		t.Fatalf("tilde-fenced command missing:\n%s", got)
	}
	if strings.Contains(got, "~~~") {
		t.Fatalf("tilde fence leaked:\n%s", got)
	}
}

func TestHuman_EmptyBodyAndEmptyLLMText(t *testing.T) {
	r := consult.Result{
		Query: "x",
		Hits:  []knowledge.Hit{{Title: "Bare title", Body: ""}},
		LLM:   &llm.Response{Text: "   ", Backend: "none"},
	}
	got := r.Human()
	if !strings.Contains(got, "Bare title") {
		t.Fatalf("title missing:\n%s", got)
	}
	if strings.Contains(got, "hunch") {
		t.Fatalf("empty llm text should not print hunch:\n%s", got)
	}
}

func TestHuman_ProseWhenAndKeywordTags(t *testing.T) {
	r := consult.Result{
		Query: "geli attach",
		Hits: []knowledge.Hit{{
			Title: "Attach a geli provider",
			Tags:  "Use this when the provider is locked at the console",
			Body:  "geli attach -k /boot/keys/root.key da0",
		}},
	}
	got := r.Human()
	if !strings.Contains(got, "Use this when the provider is locked") {
		t.Fatalf("when-to-use prose missing:\n%s", got)
	}
	r.Hits[0].Tags = "zfs rescue tier0 boot environment kit extra tokens"
	got = r.Human()
	if strings.Contains(got, "zfs rescue tier0") {
		t.Fatalf("keyword tags must not print as a summary:\n%s", got)
	}
}

func TestHuman_TitleOnlyBody(t *testing.T) {
	r := consult.Result{
		Query: "x",
		Hits:  []knowledge.Hit{{Title: "Only title", Body: "# Only title"}},
	}
	got := r.Human()
	if !strings.Contains(got, "Only title") {
		t.Fatalf("title missing:\n%s", got)
	}
}
