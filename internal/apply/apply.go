// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package apply

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
)

var (
	ErrLLMMustNotExec = errors.New("llm must not execute privileged apply; operator CLI is the only mutator")
	ErrNilExecutor    = errors.New("executor is required for apply")
)

type Mode int

const (
	ModeDryRun Mode = iota
	ModeApply
)

func (m Mode) String() string {
	if m == ModeApply {
		return "apply"
	}
	return "dry-run"
}

// ResolveMode implements Hawkeye law: apply defaults to dry-run.
// --yes without --dry-run is the only way to mutate. --dry-run always wins.
func ResolveMode(dryRunFlag, yesFlag bool) Mode {
	if dryRunFlag {
		return ModeDryRun
	}
	if yesFlag {
		return ModeApply
	}
	return ModeDryRun
}

type Actor string

const (
	ActorOperator Actor = "operator"
	ActorLLM      Actor = "llm"
	ActorMCP      Actor = "mcp"
)

type Step struct {
	ID         string   `json:"id"`
	Action     string   `json:"action"`
	Argv       []string `json:"argv"`
	Privileged bool     `json:"privileged"`
}

type Plan struct {
	ID      string `json:"id"`
	Source  string `json:"source"`
	Summary string `json:"summary"`
	Steps   []Step `json:"steps"`
}

func (p Plan) Privileged() bool {
	for _, s := range p.Steps {
		if s.Privileged {
			return true
		}
	}
	return false
}

type StepResult struct {
	ID      string `json:"id"`
	Skipped bool   `json:"skipped"`
	DryRun  bool   `json:"dry_run"`
	Output  string `json:"output,omitempty"`
	Error   string `json:"error,omitempty"`
}

type Result struct {
	DryRun  bool         `json:"dry_run"`
	Applied bool         `json:"applied"`
	Steps   []StepResult `json:"steps"`
}

type Executor interface {
	Run(argv []string) (stdout string, stderr string, err error)
}

type Auditor interface {
	Record(plan Plan, mode Mode, actor Actor, result Result) error
}

type NopAuditor struct{}

func (NopAuditor) Record(Plan, Mode, Actor, Result) error { return nil }

type CountingExecutor struct {
	Calls int
	Argv  [][]string
}

func (c *CountingExecutor) Run(argv []string) (string, string, error) {
	c.Calls++
	cp := append([]string(nil), argv...)
	c.Argv = append(c.Argv, cp)
	return "ok", "", nil
}

func Execute(plan Plan, mode Mode, actor Actor, exec Executor, auditor Auditor) (Result, error) {
	if auditor == nil {
		auditor = NopAuditor{}
	}
	res := Result{DryRun: true}

	if actor == ActorLLM && plan.Privileged() && mode == ModeApply {
		_ = auditor.Record(plan, mode, actor, res)
		return res, ErrLLMMustNotExec
	}
	if actor == ActorMCP && plan.Privileged() {
		mode = ModeDryRun
	}

	if mode == ModeApply {
		if exec == nil {
			return res, ErrNilExecutor
		}
		res.DryRun = false
	}

	for _, s := range plan.Steps {
		sr := StepResult{ID: s.ID, DryRun: mode != ModeApply}
		if mode != ModeApply {
			sr.Skipped = true
			sr.Output = "dry-run: " + strings.Join(s.Argv, " ")
			res.Steps = append(res.Steps, sr)
			continue
		}
		out, errOut, err := exec.Run(s.Argv)
		sr.Output = strings.TrimSpace(strings.TrimSpace(out) + "\n" + strings.TrimSpace(errOut))
		if err != nil {
			sr.Error = err.Error()
		}
		res.Steps = append(res.Steps, sr)
	}
	if mode == ModeApply {
		res.Applied = true
		res.DryRun = false
	}
	if err := auditor.Record(plan, mode, actor, res); err != nil {
		return res, err
	}
	return res, nil
}

type SysExecutor struct{}

func needsShell(line string) bool {
	return strings.ContainsAny(line, " \t|$;&<>(){}'\"`\n") || strings.Contains(line, "=")
}

func (SysExecutor) Run(argv []string) (string, string, error) {
	if len(argv) == 0 {
		return "", "", errors.New("empty argv")
	}
	var cmd *exec.Cmd
	if len(argv) == 1 && needsShell(argv[0]) {
		cmd = exec.Command("/bin/sh", "-c", argv[0])
	} else {
		cmd = exec.Command(argv[0], argv[1:]...)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}
