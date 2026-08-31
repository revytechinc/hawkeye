// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package apply_test

import (
	"errors"
	"testing"

	"github.com/revytechinc/hawkeye/internal/apply"
)

type failExec struct{}

func (failExec) Run([]string) (string, string, error) {
	return "", "boom", errors.New("boom")
}

func TestExecute_NilExecutor(t *testing.T) {
	p := apply.Plan{ID: "p", Source: "operator", Steps: []apply.Step{{ID: "1", Argv: []string{"true"}}}}
	if _, err := apply.Execute(p, apply.ModeApply, apply.ActorOperator, nil, apply.NopAuditor{}); err == nil {
		t.Fatal("expected nil executor error")
	}
}

func TestExecute_StepErrorRecorded(t *testing.T) {
	p := apply.Plan{ID: "p", Source: "operator", Steps: []apply.Step{{ID: "1", Argv: []string{"nope"}}}}
	res, err := apply.Execute(p, apply.ModeApply, apply.ActorOperator, failExec{}, apply.NopAuditor{})
	if err == nil {
		t.Fatal("step failure must return an error")
	}
	if res.Applied {
		t.Fatalf("failed land must not set Applied: %+v", res)
	}
	if res.Steps[0].Error == "" {
		t.Fatalf("%+v", res)
	}
}

func TestExecute_AnyStepFailureIsNotApplied(t *testing.T) {
	p := apply.Plan{ID: "p", Source: "operator", Steps: []apply.Step{
		{ID: "1", Argv: []string{"echo", "ok"}},
		{ID: "2", Argv: []string{"nope"}},
	}}
	res, err := apply.Execute(p, apply.ModeApply, apply.ActorOperator, &seqFailExec{}, apply.NopAuditor{})
	if err == nil {
		t.Fatal("expected step error")
	}
	if res.DryRun {
		t.Fatal("mode apply")
	}
	if res.Applied {
		t.Fatalf("any step failure → Applied=false: %+v", res)
	}
	if res.Steps[0].Error != "" || res.Steps[1].Error == "" {
		t.Fatalf("partial results: %+v", res)
	}
}

type seqFailExec struct{ n int }

func (s *seqFailExec) Run([]string) (string, string, error) {
	s.n++
	if s.n == 2 {
		return "", "boom", errors.New("boom")
	}
	return "ok", "", nil
}

func TestExecute_LLMDryRunOK(t *testing.T) {
	p := apply.Plan{ID: "p", Source: "llm", Steps: []apply.Step{{ID: "1", Privileged: true, Argv: []string{"true"}}}}
	res, err := apply.Execute(p, apply.ModeDryRun, apply.ActorLLM, &apply.CountingExecutor{}, apply.NopAuditor{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.DryRun {
		t.Fatal(res)
	}
}

func TestModeString(t *testing.T) {
	if apply.ModeDryRun.String() != "dry-run" || apply.ModeApply.String() != "apply" {
		t.Fatal("strings")
	}
}

type failAud struct{}

func (failAud) Record(apply.Plan, apply.Mode, apply.Actor, apply.Result) error {
	return errors.New("audit")
}

func TestExecute_AuditorError(t *testing.T) {
	p := apply.Plan{ID: "p", Source: "operator"}
	if _, err := apply.Execute(p, apply.ModeDryRun, apply.ActorOperator, &apply.CountingExecutor{}, failAud{}); err == nil {
		t.Fatal("expected auditor error")
	}
}

func TestExecute_UnprivilegedPlan(t *testing.T) {
	p := apply.Plan{ID: "p", Source: "operator", Steps: []apply.Step{{ID: "1", Privileged: false, Argv: []string{"true"}}}}
	if p.Privileged() {
		t.Fatal("not privileged")
	}
}

func TestExecute_NilAuditor(t *testing.T) {
	p := apply.Plan{ID: "p", Source: "operator", Steps: []apply.Step{{ID: "1", Argv: []string{"echo", "x"}}}}
	res, err := apply.Execute(p, apply.ModeDryRun, apply.ActorOperator, &apply.CountingExecutor{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.DryRun {
		t.Fatal(res)
	}
}
