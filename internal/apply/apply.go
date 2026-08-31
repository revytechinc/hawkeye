// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package apply

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrLLMMustNotExec = errors.New("llm must not execute privileged apply; operator CLI is the only mutator")
	ErrNilExecutor    = errors.New("executor is required for apply")
	ErrStepFailed     = errors.New("one or more apply steps failed")
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
	failed := false

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
			failed = true
		}
		res.Steps = append(res.Steps, sr)
	}
	if mode == ModeApply {
		res.Applied = !failed
		res.DryRun = false
	}
	if err := auditor.Record(plan, mode, actor, res); err != nil {
		return res, err
	}
	if failed {
		return res, ErrStepFailed
	}
	return res, nil
}

// SysExecutor runs argv. Stored playbook lines (single-element argv that
// needs a shell) share one /bin/sh session so ROOTDS= and export PATH
// persist into later zfs set "$ROOTDS" steps. Non-shell argv is one-shot.
type SysExecutor struct {
	// Shell is the session interpreter. Empty means /bin/sh.
	Shell  string
	mu     sync.Mutex
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	errBuf bytes.Buffer
}

func needsShell(line string) bool {
	return strings.ContainsAny(line, " \t|$;&<>(){}'\"`\n") || strings.Contains(line, "=")
}

func (s *SysExecutor) Run(argv []string) (string, string, error) {
	if len(argv) == 0 {
		return "", "", errors.New("empty argv")
	}
	if len(argv) == 1 && needsShell(argv[0]) {
		return s.runShellLine(argv[0])
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func (s *SysExecutor) Close() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.resetLocked()
}

func (s *SysExecutor) runShellLine(line string) (string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureLocked(); err != nil {
		return "", "", err
	}
	token := fmt.Sprintf("HAWKEYE_DONE_%d_%d", os.Getpid(), time.Now().UnixNano())
	beforeErr := s.errBuf.Len()
	script := line + "\n" + "printf '%s %d\\n' '" + token + "' $?\n"
	if _, err := io.WriteString(s.stdin, script); err != nil {
		_ = s.resetLocked()
		return "", "", err
	}
	var out strings.Builder
	for {
		rec, err := s.stdout.ReadString('\n')
		if err != nil {
			_ = s.resetLocked()
			return out.String(), s.errSince(beforeErr), err
		}
		if strings.HasPrefix(rec, token+" ") {
			codeStr := strings.TrimSpace(strings.TrimPrefix(rec, token+" "))
			code, _ := strconv.Atoi(codeStr)
			stderr := s.errSince(beforeErr)
			if code != 0 {
				return out.String(), stderr, fmt.Errorf("exit status %d", code)
			}
			return out.String(), stderr, nil
		}
		out.WriteString(rec)
	}
}

func (s *SysExecutor) errSince(n int) string {
	b := s.errBuf.Bytes()
	if n >= len(b) {
		return ""
	}
	return string(b[n:])
}

func (s *SysExecutor) ensureLocked() error {
	if s.cmd != nil && s.cmd.Process != nil && s.stdin != nil && s.stdout != nil {
		return nil
	}
	_ = s.resetLocked()
	shell := "/bin/sh"
	if s.Shell != "" {
		shell = s.Shell
	}
	cmd := exec.Command(shell)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return err
	}
	cmd.Stderr = &s.errBuf
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return err
	}
	s.cmd = cmd
	s.stdin = stdin
	s.stdout = bufio.NewReader(stdout)
	return nil
}

func (s *SysExecutor) resetLocked() error {
	var err error
	if s.stdin != nil {
		_ = s.stdin.Close()
		s.stdin = nil
	}
	if s.cmd != nil {
		err = s.cmd.Wait()
		s.cmd = nil
	}
	s.stdout = nil
	return err
}
