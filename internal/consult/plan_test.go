// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package consult_test

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/revytechinc/hawkeye/internal/apply"
	"github.com/revytechinc/hawkeye/internal/consult"
	"github.com/revytechinc/hawkeye/internal/knowledge"
	"github.com/revytechinc/hawkeye/internal/probe"
)

func TestPlan_UsesLeadPlaybookStoredCommands(t *testing.T) {
	r := consult.Result{
		Query: "ZFS root is read-only after boot",
		Hits:  jailOrderHits(),
	}
	p := r.Plan(probe.Snapshot{Tier: 0, RootRO: true})
	if p.Source != "knowledge" {
		t.Fatalf("source %q", p.Source)
	}
	if p.Summary != "Remount ZFS root read-write" {
		t.Fatalf("summary must be the lead playbook title, got %q", p.Summary)
	}
	got := stepLines(p)
	want := []string{
		"export PATH=/rescue:/sbin:/bin:/usr/sbin:/usr/bin",
		"mount -p",
		`zfs set readonly=off "$ROOTDS"`,
		"mount -u -o rw /",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("steps must be stored playbook commands, not a stub:\n%q", got)
	}
	for _, s := range p.Steps {
		if !s.Privileged {
			t.Fatalf("playbook apply steps are privileged: %#v", s)
		}
		if len(s.Argv) != 1 || s.Argv[0] == "" {
			t.Fatalf("argv must be the stored line: %#v", s)
		}
	}
	for _, junk := range []string{"echo ZFS", "echo <query>", "<rootpool>"} {
		joined := strings.Join(got, " ")
		if strings.Contains(joined, junk) {
			t.Fatalf("invented command %q in plan: %v", junk, got)
		}
	}
}

func TestPlan_JSONIsMachineShaped(t *testing.T) {
	r := consult.Result{Query: "ZFS root is read-only after boot", Hits: jailOrderHits()}
	p := r.Plan(probe.Snapshot{})
	b, err := json.Marshal(p)
	if err != nil || !json.Valid(b) {
		t.Fatal(err)
	}
	raw := string(b)
	for _, key := range []string{`"id"`, `"source"`, `"summary"`, `"steps"`, `"argv"`} {
		if !strings.Contains(raw, key) {
			t.Fatalf("plan JSON missing %s: %s", key, raw)
		}
	}
	if strings.Contains(raw, `"echo"`) && strings.Contains(raw, r.Query) {
		t.Fatalf("JSON must not be an echo stub: %s", raw)
	}
	if !strings.Contains(raw, `zfs set readonly=off`) {
		t.Fatalf("JSON missing stored command: %s", raw)
	}
}

func TestPlan_PrefersHitCommandsOverProse(t *testing.T) {
	r := consult.Result{
		Query: "ufs remount",
		Hits: []knowledge.Hit{{
			Title:    "Remount UFS",
			Body:     "This is prose. Do not exec this sentence.",
			Commands: []string{"mount -u -o rw /"},
		}},
	}
	p := r.Plan(probe.Snapshot{})
	if len(p.Steps) != 1 || p.Steps[0].Argv[0] != "mount -u -o rw /" {
		t.Fatalf("must use stored commands JSON, not prose: %#v", p.Steps)
	}
}

func TestPlan_FencedBodyWhenCommandsEmpty(t *testing.T) {
	r := consult.Result{
		Query: "ufs remount",
		Hits: []knowledge.Hit{{
			Title: "Remount UFS",
			Body:  "# Remount UFS\n\n~~~\nmount -u -o rw /\n~~~\n",
		}},
	}
	p := r.Plan(probe.Snapshot{})
	if len(p.Steps) != 1 || p.Steps[0].Argv[0] != "mount -u -o rw /" {
		t.Fatalf("must parse fenced body commands: %#v", p.Steps)
	}
}

func TestPlan_ProseBodyIsNotACommand(t *testing.T) {
	r := consult.Result{
		Query: "zfs readonly",
		Hits: []knowledge.Hit{{
			Title: "ZFS readonly pool",
			Body:  "If the root pool is imported readonly, first skill is unlock-rw, not pkg.",
		}},
	}
	p := r.Plan(probe.Snapshot{RootRO: false})
	if len(p.Steps) != 0 {
		t.Fatalf("prose must not become argv: %#v", p.Steps)
	}
	for _, s := range p.Steps {
		if len(s.Argv) > 0 && s.Argv[0] == "If" {
			t.Fatal("invented command from prose")
		}
	}
}

func TestPlan_NoHitsWritableDoesNotEcho(t *testing.T) {
	r := consult.Result{Query: "hello"}
	p := r.Plan(probe.Snapshot{Tier: 2})
	if len(p.Steps) != 0 {
		t.Fatalf("no playbook: do not invent echo: %#v", p.Steps)
	}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"steps":[]`) && !strings.Contains(string(b), `"steps": []`) {
		t.Fatalf("empty plan must still be machine-shaped: %s", b)
	}
	if strings.Contains(string(b), "echo") {
		t.Fatalf("echo stub leaked: %s", b)
	}
}

func TestPlan_NoHitsRootROUnlockRW(t *testing.T) {
	r := consult.Result{Query: "pkg install foo"}
	p := r.Plan(probe.Snapshot{RootRO: true, RescuePresent: true})
	if len(p.Steps) != 1 || p.Steps[0].Action != "unlock-rw" {
		t.Fatalf("missing knowledge on RO root keeps first skill unlock-rw: %#v", p.Steps)
	}
}

func TestPlan_RedactsSecretsInCommands(t *testing.T) {
	r := consult.Result{
		Query: "load keys",
		Hits: []knowledge.Hit{{
			Title:    "Load keys",
			Commands: []string{"echo password=fake-password-for-tests-only"},
		}},
	}
	p := r.Plan(probe.Snapshot{})
	raw, _ := json.Marshal(p)
	if strings.Contains(string(raw), "fake-password-for-tests-only") {
		t.Fatalf("secret in plan JSON: %s", raw)
	}
}

func TestCommandLines_SkipsCommentsAndEmpty(t *testing.T) {
	got := consult.CommandLines(knowledge.Hit{
		Commands: []string{"", "# note", "mount -p", "  "},
	})
	if len(got) != 1 || got[0] != "mount -p" {
		t.Fatalf("%q", got)
	}
}

func TestPlan_FromKnowledgeStore(t *testing.T) {
	dir := t.TempDir()
	if err := knowledge.CreatePlaybookTestDB(filepath.Join(dir, "knowledge.sqlite")); err != nil {
		t.Fatal(err)
	}
	st, err := knowledge.Open([]string{dir}, true)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	snap := probe.Snapshot{RootRO: true, Tier: 0, RescuePresent: true}
	res, err := consult.Run("ZFS root is read-only after boot", snap, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Hits) < 1 {
		t.Fatal("expected hits")
	}
	p := res.Plan(snap)
	got := strings.Join(stepLines(p), "\n")
	if !strings.Contains(got, `zfs set readonly=off "$ROOTDS"`) {
		t.Fatalf("store-backed plan missing remount command: %s hits=%#v", got, res.Hits)
	}
	if strings.Contains(got, "echo ") || strings.Contains(got, "<rootpool>") {
		t.Fatalf("store-backed plan used stub: %s", got)
	}
}

func stepLines(p apply.Plan) []string {
	var out []string
	for _, s := range p.Steps {
		out = append(out, strings.Join(s.Argv, " "))
	}
	return out
}
