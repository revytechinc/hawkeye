// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/revytechinc/hawkeye/internal/apply"
	"github.com/revytechinc/hawkeye/internal/audit"
	"github.com/revytechinc/hawkeye/internal/config"
	"github.com/revytechinc/hawkeye/internal/redact"
)

const (
	promptApplySteps = "Apply these steps? [y/N/e] "
	promptApplyReal  = "Apply for real? [y/N] "
	msgNothing       = "nothing applied"
)

type applyChoice int

const (
	choiceNo applyChoice = iota
	choiceYes
	choiceEdit
	choiceInvalid
)

func parseApplyChoice(s string) applyChoice {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "y", "yes":
		return choiceYes
	case "e", "edit":
		return choiceEdit
	case "n", "no", "":
		return choiceNo
	default:
		return choiceInvalid
	}
}

func readerIsTTY(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	st, err := f.Stat()
	if err != nil {
		return false
	}
	return st.Mode()&os.ModeCharDevice != 0
}

func planCommands(p apply.Plan) []string {
	var out []string
	for _, s := range p.Steps {
		if len(s.Argv) == 0 {
			continue
		}
		out = append(out, strings.Join(s.Argv, " "))
	}
	return out
}

func promptConsultApply(env Env, fs flagset, cfg config.Config, plan apply.Plan) int {
	return promptConsultApplyReader(env, fs, cfg, plan, bufio.NewReader(env.Stdin))
}

func promptConsultApplyReader(env Env, fs flagset, cfg config.Config, plan apply.Plan, in *bufio.Reader) int {
	for {
		fmt.Fprint(env.Stdout, promptApplySteps)
		line, err := readPromptLine(in)
		if err != nil {
			fmt.Fprintln(env.Stdout, msgNothing)
			return 0
		}
		switch parseApplyChoice(line) {
		case choiceNo:
			fmt.Fprintln(env.Stdout, msgNothing)
			return 0
		case choiceYes:
			return confirmAndApply(env, fs, cfg, plan, in)
		case choiceEdit:
			edited, ok := editPlan(env, plan)
			if !ok {
				fmt.Fprintln(env.Stdout, msgNothing)
				return 0
			}
			plan = edited
			fmt.Fprintln(env.Stdout, "Edited plan")
			for _, cmd := range planCommands(plan) {
				fmt.Fprintf(env.Stdout, "  %s\n", cmd)
			}
			fmt.Fprintln(env.Stdout)
			return confirmAndApply(env, fs, cfg, plan, in)
		default:
			// invalid key: re-prompt
		}
	}
}

func confirmAndApply(env Env, fs flagset, cfg config.Config, plan apply.Plan, in *bufio.Reader) int {
	if fs.yes && !fs.dryRun {
		return printApply(env, cfg, plan, apply.ModeApply)
	}
	if code := printApply(env, cfg, plan, apply.ModeDryRun); code != 0 {
		return code
	}
	if fs.dryRun {
		return 0
	}
	for {
		fmt.Fprint(env.Stdout, promptApplyReal)
		line, err := readPromptLine(in)
		if err != nil {
			fmt.Fprintln(env.Stdout, msgNothing)
			return 0
		}
		switch parseApplyChoice(line) {
		case choiceYes:
			return printApply(env, cfg, plan, apply.ModeApply)
		case choiceNo:
			fmt.Fprintln(env.Stdout, msgNothing)
			return 0
		default:
			// e and other keys re-prompt on the land gate
		}
	}
}

func readPromptLine(in *bufio.Reader) (string, error) {
	line, err := in.ReadString('\n')
	if err != nil && len(strings.TrimSpace(line)) == 0 {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func printApply(env Env, cfg config.Config, plan apply.Plan, mode apply.Mode) int {
	res, err := executePlan(env, cfg, plan, mode)
	if err != nil {
		fmt.Fprintln(env.Stderr, err)
		return 1
	}
	for _, s := range res.Steps {
		if s.Output != "" {
			fmt.Fprintln(env.Stdout, s.Output)
		}
		if s.Error != "" {
			fmt.Fprintln(env.Stdout, s.Error)
		}
	}
	if res.Applied {
		fmt.Fprintln(env.Stdout, "applied")
	}
	return 0
}

func executePlan(env Env, cfg config.Config, p apply.Plan, mode apply.Mode) (apply.Result, error) {
	return executePlanActor(env, cfg, p, mode, apply.ActorOperator)
}

func executePlanActor(env Env, cfg config.Config, p apply.Plan, mode apply.Mode, actor apply.Actor) (apply.Result, error) {
	auditor, err := applyAuditor(cfg, mode)
	if err != nil {
		return apply.Result{}, err
	}
	ex := env.Exec
	if ex == nil {
		ex = apply.SysExecutor{}
	}
	return apply.Execute(p, mode, actor, ex, auditor)
}

// mcpApply is the same gate as CLI apply: ResolveMode (default dry-run,
// --yes to land, --dry-run wins), SysExecutor, and the configured auditor.
// Privileged plans stay dry-run under ActorMCP even when yes is set.
func mcpApply(env Env, cfg config.Config, p apply.Plan, yes bool) (any, error) {
	mode := apply.ResolveMode(!yes, yes)
	return executePlanActor(env, cfg, p, mode, apply.ActorMCP)
}

func applyAuditor(cfg config.Config, mode apply.Mode) (apply.Auditor, error) {
	if cfg.AuditLog == "" {
		return apply.NopAuditor{}, nil
	}
	dir := filepath.Dir(cfg.AuditLog)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		if mode == apply.ModeApply {
			return nil, fmt.Errorf("hawkeye apply: audit log %s: %w", cfg.AuditLog, err)
		}
		return apply.NopAuditor{}, nil
	}
	return &audit.File{Path: cfg.AuditLog}, nil
}

func editorArgv(getenv func(string) string, path string) []string {
	name := ""
	if getenv != nil {
		name = getenv("VISUAL")
		if name == "" {
			name = getenv("EDITOR")
		}
	}
	if strings.TrimSpace(name) == "" {
		name = "vi"
	}
	fields := strings.Fields(name)
	return append(fields, path)
}

func defaultEdit(env Env) func(string) error {
	return func(path string) error {
		argv := editorArgv(env.Getenv, path)
		cmd := exec.Command(argv[0], argv[1:]...)
		if f, ok := env.Stdin.(*os.File); ok {
			cmd.Stdin = f
		} else {
			cmd.Stdin = os.Stdin
		}
		if f, ok := env.Stdout.(*os.File); ok {
			cmd.Stdout = f
		} else {
			cmd.Stdout = os.Stdout
		}
		if f, ok := env.Stderr.(*os.File); ok {
			cmd.Stderr = f
		} else {
			cmd.Stderr = os.Stderr
		}
		return cmd.Run()
	}
}

func editPlan(env Env, plan apply.Plan) (apply.Plan, bool) {
	dir := ""
	if env.Getenv != nil {
		dir = env.Getenv("TMPDIR")
	}
	f, err := os.CreateTemp(dir, "hawkeye-plan-*.json")
	if err != nil {
		return apply.Plan{}, false
	}
	name := f.Name()
	_ = f.Close()
	defer func() { _ = os.Remove(name) }()

	raw, err := json.MarshalIndent(redactPlan(plan), "", "  ")
	if err != nil {
		return apply.Plan{}, false
	}
	if err := os.WriteFile(name, raw, 0o600); err != nil {
		return apply.Plan{}, false
	}

	edit := env.Editor
	if edit == nil {
		edit = defaultEdit(env)
	}
	if err := edit(name); err != nil {
		return apply.Plan{}, false
	}
	got, err := os.ReadFile(name)
	if err != nil {
		return apply.Plan{}, false
	}
	return parseEditedPlan(got, plan)
}

func redactPlan(p apply.Plan) apply.Plan {
	p.ID = redact.String(p.ID)
	p.Source = redact.String(p.Source)
	p.Summary = redact.String(p.Summary)
	steps := make([]apply.Step, len(p.Steps))
	for i, s := range p.Steps {
		s.ID = redact.String(s.ID)
		s.Action = redact.String(s.Action)
		if len(s.Argv) > 0 {
			argv := make([]string, len(s.Argv))
			for j, a := range s.Argv {
				argv[j] = redact.String(a)
			}
			s.Argv = argv
		}
		steps[i] = s
	}
	p.Steps = steps
	return p
}

func parseEditedPlan(raw []byte, fallback apply.Plan) (apply.Plan, bool) {
	s := strings.TrimSpace(string(raw))
	if s == "" {
		return apply.Plan{}, false
	}
	if strings.HasPrefix(s, "{") {
		var p apply.Plan
		if err := json.Unmarshal([]byte(s), &p); err != nil {
			return apply.Plan{}, false
		}
		if len(p.Steps) == 0 {
			return apply.Plan{}, false
		}
		return redactPlan(p), true
	}
	p := fallback
	p.Steps = nil
	n := 0
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(redact.String(line))
		if len(fields) == 0 {
			continue
		}
		n++
		p.Steps = append(p.Steps, apply.Step{
			ID:         strconv.Itoa(n),
			Action:     "command",
			Argv:       fields,
			Privileged: fallback.Privileged(),
		})
	}
	if len(p.Steps) == 0 {
		return apply.Plan{}, false
	}
	return redactPlan(p), true
}
