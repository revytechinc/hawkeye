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
	if err != nil {
		t.Fatal(err)
	}
	if res.Steps[0].Error == "" {
		t.Fatalf("%+v", res)
	}
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
