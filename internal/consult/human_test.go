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

func TestHuman_OperatorSessionNotJSON(t *testing.T) {
	r := consult.Result{
		Query:      "ZFS root is read-only after boot",
		Tier:       0,
		FirstSkill: "unlock-rw",
		Hits: []knowledge.Hit{{
			Title: "Remount ZFS root read-write",
			Tags:  "Root is a ZFS dataset and is mounted read-only after boot.",
			Body: `# Remount ZFS root read-write

Use this when / is ZFS and mount shows read-only.

## Commands

` + "```sh\n" + `export PATH=/rescue:/sbin:/bin
zfs set readonly=off "$ROOTDS"
mount -u -o rw /
` + "```\n",
			Rank: -1.25,
		}},
		Notes: []string{"root is read-only; consult does not write; first skill is unlock-rw, not pkg"},
	}
	got := r.Human()
	if strings.TrimSpace(got) == "" {
		t.Fatal("empty human output")
	}
	if strings.HasPrefix(strings.TrimSpace(got), "{") {
		t.Fatalf("human output must not be a JSON blob:\n%s", got)
	}
	for _, key := range []string{`"Title"`, `"when_to_use"`, `"score"`, `"query":`, `"hits"`} {
		if strings.Contains(got, key) {
			t.Fatalf("human output leaked JSON key %s:\n%s", key, got)
		}
	}
	if !strings.Contains(got, "ZFS root is read-only after boot") {
		t.Fatalf("query missing:\n%s", got)
	}
	if !strings.Contains(got, "Remount ZFS root read-write") {
		t.Fatalf("title missing:\n%s", got)
	}
	if !strings.Contains(got, "Root is a ZFS dataset and is mounted read-only after boot.") {
		t.Fatalf("summary missing:\n%s", got)
	}
	if !strings.Contains(got, `export PATH=/rescue:/sbin:/bin`) {
		t.Fatalf("playbook command missing:\n%s", got)
	}
	if !strings.Contains(got, `zfs set readonly=off "$ROOTDS"`) {
		t.Fatalf("playbook command missing:\n%s", got)
	}
	if strings.Contains(got, "```") {
		t.Fatalf("fences should be unwrapped so commands look typed:\n%s", got)
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

func TestHuman_EmptyHitsStillShowsQuery(t *testing.T) {
	r, err := consult.Run("hello", probe.Snapshot{Tier: 0, RootRO: true}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := r.Human()
	if !strings.Contains(got, "hello") {
		t.Fatalf("query missing:\n%s", got)
	}
	if strings.HasPrefix(strings.TrimSpace(got), "{") {
		t.Fatalf("must not be JSON:\n%s", got)
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

func TestHuman_EmptyQueryAndUntitledHit(t *testing.T) {
	r := consult.Result{
		Query: "   ",
		Tier:  2,
		Hits:  []knowledge.Hit{{Title: "", Body: "plain stored body", Tags: ""}},
		Notes: []string{"", "knowledge store unavailable; FTS skipped"},
	}
	got := r.Human()
	if !strings.Contains(got, "(empty query)") {
		t.Fatalf("empty query chrome missing:\n%s", got)
	}
	if !strings.Contains(got, "untitled") {
		t.Fatalf("untitled fallback missing:\n%s", got)
	}
	if !strings.Contains(got, "plain stored body") {
		t.Fatalf("body missing:\n%s", got)
	}
	if !strings.Contains(got, "knowledge store unavailable") {
		t.Fatalf("note missing:\n%s", got)
	}
}

func TestHuman_TildeFenceAndNonMatchingHeading(t *testing.T) {
	r := consult.Result{
		Query: "ufs",
		Hits: []knowledge.Hit{{
			Title: "Remount UFS",
			Body:  "# Other heading\n\n~~~\nmount -u -o rw /\n~~~\n",
		}},
	}
	got := r.Human()
	if !strings.Contains(got, "Other heading") {
		t.Fatalf("non-matching heading must stay:\n%s", got)
	}
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

func TestJSON_StillMachineObject(t *testing.T) {
	r := consult.Result{Query: "zfs", Hits: []knowledge.Hit{{Title: "Remount ZFS root read-write"}}}
	b, err := r.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(b) {
		t.Fatalf("invalid JSON: %s", b)
	}
	if !strings.Contains(string(b), `"query"`) {
		t.Fatalf("JSON missing query: %s", b)
	}
}
