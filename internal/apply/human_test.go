// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package apply_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/revytechinc/hawkeye/internal/apply"
)

func TestPlanHuman_OperatorProseNotJSON(t *testing.T) {
	p := apply.Plan{
		ID:      "consult-plan",
		Source:  "knowledge",
		Summary: "root is read-only; first skill is unlock-rw, not pkg",
		Steps: []apply.Step{{
			ID:         "1",
			Action:     "unlock-rw",
			Argv:       []string{"zfs", "set", "readonly=off", "<rootpool>"},
			Privileged: true,
		}},
	}
	got := p.Human()
	if strings.HasPrefix(strings.TrimSpace(got), "{") {
		t.Fatalf("plan human must not be JSON:\n%s", got)
	}
	for _, key := range []string{`"id"`, `"argv"`, `"privileged"`, `"summary"`} {
		if strings.Contains(got, key) {
			t.Fatalf("plan human leaked JSON key %s:\n%s", key, got)
		}
	}
	if !strings.Contains(got, "unlock-rw") {
		t.Fatalf("action missing:\n%s", got)
	}
	if !strings.Contains(got, "zfs set readonly=off <rootpool>") {
		t.Fatalf("command as typed missing:\n%s", got)
	}
	if !strings.Contains(got, "root is read-only") {
		t.Fatalf("summary missing:\n%s", got)
	}
}

func TestPlanHuman_RedactsSecrets(t *testing.T) {
	p := apply.Plan{
		Summary: "rotate",
		Steps: []apply.Step{{
			Action: "echo",
			Argv:   []string{"echo", "password=fake-password-for-tests-only"},
		}},
	}
	got := p.Human()
	if strings.Contains(got, "fake-password-for-tests-only") {
		t.Fatalf("secret leaked:\n%s", got)
	}
}

func TestPlanHuman_EmptyAndUnprivileged(t *testing.T) {
	empty := apply.Plan{}
	got := empty.Human()
	if !strings.Contains(got, "(empty plan)") || !strings.Contains(got, "no steps") {
		t.Fatalf("empty plan:\n%s", got)
	}
	p := apply.Plan{
		ID: "plan-1",
		Steps: []apply.Step{
			{Argv: []string{"echo", "hi"}},
			{ID: "2", Action: "diagnose"},
		},
	}
	got = p.Human()
	if !strings.Contains(got, "echo hi") {
		t.Fatalf("unprivileged argv missing:\n%s", got)
	}
	if strings.Contains(got, "privileged") {
		t.Fatalf("unprivileged marked privileged:\n%s", got)
	}
	if !strings.Contains(got, "diagnose") {
		t.Fatalf("second step missing:\n%s", got)
	}
}

func TestPlanJSON_Unchanged(t *testing.T) {
	p := samplePlan("operator")
	b, err := json.Marshal(p)
	if err != nil || !json.Valid(b) {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"argv"`) {
		t.Fatalf("machine plan JSON missing argv: %s", b)
	}
}
