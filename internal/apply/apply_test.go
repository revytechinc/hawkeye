// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package apply_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/revytechinc/hawkeye/internal/apply"
)

func samplePlan(source string) apply.Plan {
	return apply.Plan{
		ID:      "plan-1",
		Source:  source,
		Summary: "restart sshd after sshd_config check",
		Steps: []apply.Step{{
			ID:         "1",
			Action:     "service",
			Argv:       []string{"service", "sshd", "restart"},
			Privileged: true,
		}},
	}
}

func TestResolveMode_DefaultIsDryRun(t *testing.T) {
	if m := apply.ResolveMode(false, false); m != apply.ModeDryRun {
		t.Fatalf("default mode = %s, want dry-run", m)
	}
}

func TestResolveMode_YesMutatesUnlessDryRun(t *testing.T) {
	if m := apply.ResolveMode(false, true); m != apply.ModeApply {
		t.Fatalf("--yes mode = %s, want apply", m)
	}
	if m := apply.ResolveMode(true, true); m != apply.ModeDryRun {
		t.Fatalf("--dry-run wins over --yes: got %s", m)
	}
	if m := apply.ResolveMode(true, false); m != apply.ModeDryRun {
		t.Fatalf("--dry-run mode = %s", m)
	}
}

func TestExecute_DefaultDryRunDoesNotCallExecutor(t *testing.T) {
	ex := &apply.CountingExecutor{}
	mode := apply.ResolveMode(false, false)
	res, err := apply.Execute(samplePlan("operator"), mode, apply.ActorOperator, ex, apply.NopAuditor{})
	if err != nil {
		t.Fatal(err)
	}
	if ex.Calls != 0 {
		t.Fatalf("dry-run called executor %d times", ex.Calls)
	}
	if !res.DryRun || res.Applied {
		t.Fatalf("result %+v", res)
	}
}

func TestExecute_YesCallsExecutorForOperator(t *testing.T) {
	ex := &apply.CountingExecutor{}
	res, err := apply.Execute(samplePlan("operator"), apply.ModeApply, apply.ActorOperator, ex, apply.NopAuditor{})
	if err != nil {
		t.Fatal(err)
	}
	if ex.Calls != 1 {
		t.Fatalf("calls = %d", ex.Calls)
	}
	if res.DryRun || !res.Applied {
		t.Fatalf("result %+v", res)
	}
}

func TestExecute_LLMActorNeverExecsPrivileged(t *testing.T) {
	ex := &apply.CountingExecutor{}
	_, err := apply.Execute(samplePlan("llm"), apply.ModeApply, apply.ActorLLM, ex, apply.NopAuditor{})
	if err == nil {
		t.Fatal("expected error: LLM must not exec as root")
	}
	if !strings.Contains(err.Error(), "llm") {
		t.Fatalf("error %v", err)
	}
	if ex.Calls != 0 {
		t.Fatalf("LLM actor executed %d privileged commands", ex.Calls)
	}
}

func TestExecute_MCPActorPrivilegedIsDryRunOnly(t *testing.T) {
	ex := &apply.CountingExecutor{}
	res, err := apply.Execute(samplePlan("knowledge"), apply.ModeApply, apply.ActorMCP, ex, apply.NopAuditor{})
	if err != nil {
		t.Fatal(err)
	}
	if ex.Calls != 0 {
		t.Fatal("MCP must not mutate privileged steps")
	}
	if !res.DryRun {
		t.Fatal("MCP apply must remain dry-run for privileged steps")
	}
}

func TestPlanJSONRoundTrip(t *testing.T) {
	p := samplePlan("operator")
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var got apply.Plan
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != p.ID || len(got.Steps) != 1 {
		t.Fatalf("round trip %+v", got)
	}
}
