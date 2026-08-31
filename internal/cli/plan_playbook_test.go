// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/revytechinc/hawkeye/internal/apply"
	"github.com/revytechinc/hawkeye/internal/cli"
	"github.com/revytechinc/hawkeye/internal/knowledge"
)

func playbookDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := knowledge.CreatePlaybookTestDB(filepath.Join(dir, "knowledge.sqlite")); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestPlan_UsesPlaybookNotEchoStub(t *testing.T) {
	dir := playbookDir(t)
	code, out, err := run(t, []string{"plan", "--json", "ZFS", "root", "is", "read-only", "after", "boot"}, "", fakeHost{ro: true, rescue: true}, map[string]string{"HAWKEYE_KNOWLEDGE_PATH": dir})
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if !json.Valid([]byte(strings.TrimSpace(out))) {
		t.Fatalf("plan --json must stay machine-shaped:\n%s", out)
	}
	var p apply.Plan
	if err := json.Unmarshal([]byte(out), &p); err != nil {
		t.Fatal(err)
	}
	joined := ""
	for _, s := range p.Steps {
		joined += strings.Join(s.Argv, " ") + "\n"
	}
	if !strings.Contains(joined, `zfs set readonly=off "$ROOTDS"`) {
		t.Fatalf("plan must use stored remount command:\n%s", out)
	}
	if strings.Contains(joined, "echo ZFS") || strings.Contains(out, `"action": "diagnose"`) {
		t.Fatalf("echo stub still present:\n%s", out)
	}
	if strings.Contains(joined, "<rootpool>") && !strings.Contains(joined, "$ROOTDS") {
		t.Fatalf("RO root must not replace the playbook with only unlock-rw:\n%s", out)
	}
}

func TestPlan_HumanShowsStoredCommands(t *testing.T) {
	dir := playbookDir(t)
	code, out, err := run(t, []string{"plan", "ZFS", "root", "is", "read-only", "after", "boot"}, "", fakeHost{usr: true, varp: true}, map[string]string{"HAWKEYE_KNOWLEDGE_PATH": dir})
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if json.Valid([]byte(strings.TrimSpace(out))) {
		t.Fatalf("default plan must stay operator prose:\n%s", out)
	}
	if !strings.Contains(out, "Remount ZFS root read-write") {
		t.Fatalf("playbook title missing:\n%s", out)
	}
	if !strings.Contains(out, `zfs set readonly=off "$ROOTDS"`) {
		t.Fatalf("stored command missing:\n%s", out)
	}
	if strings.Contains(out, "echo ZFS") {
		t.Fatalf("echo stub:\n%s", out)
	}
}

func TestPlan_WritableNoHitsNoEcho(t *testing.T) {
	code, out, err := run(t, []string{"plan", "--json", "hello"}, "", fakeHost{usr: true, varp: true}, nil)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if !json.Valid([]byte(strings.TrimSpace(out))) {
		t.Fatalf("must stay JSON:\n%s", out)
	}
	if strings.Contains(out, `"echo"`) && strings.Contains(out, "hello") {
		t.Fatalf("must not invent echo <query>:\n%s", out)
	}
}

func TestConsult_TTY_ApplyDryRunsPlaybook(t *testing.T) {
	ex := &apply.CountingExecutor{}
	dir := playbookDir(t)
	cfgPath, _ := auditConfig(t)
	env := cli.Env{
		Args:  []string{"hawkeye", "--config", cfgPath, "consult", "ZFS", "root", "is", "read-only", "after", "boot"},
		Stdin: strings.NewReader("y\n\n"),
		Getenv: func(k string) string {
			switch k {
			case "HAWKEYE_KNOWLEDGE_PATH":
				return dir
			case "HAWKEYE_CONFIG":
				return cfgPath
			default:
				return ""
			}
		},
		Host: fakeHost{ro: true, rescue: true},
		TTY:  true,
		Exec: ex,
	}
	code, out, errb := runConsult(t, env)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, errb)
	}
	if !strings.Contains(out, `dry-run: zfs set readonly=off "$ROOTDS"`) {
		t.Fatalf("TTY apply must dry-run stored playbook commands:\n%s", out)
	}
	if strings.Contains(out, "dry-run: echo ") {
		t.Fatalf("TTY apply still dry-runs echo stub:\n%s", out)
	}
	if ex.Calls != 0 {
		t.Fatalf("default N after dry-run mutated: %d", ex.Calls)
	}
}

func TestConsult_TTY_YesLandsPlaybookCommands(t *testing.T) {
	ex := &apply.CountingExecutor{}
	dir := playbookDir(t)
	cfgPath, auditPath := auditConfig(t)
	env := cli.Env{
		Args:  []string{"hawkeye", "--config", cfgPath, "--yes", "consult", "ZFS", "root", "is", "read-only", "after", "boot"},
		Stdin: strings.NewReader("y\n"),
		Getenv: func(k string) string {
			switch k {
			case "HAWKEYE_KNOWLEDGE_PATH":
				return dir
			case "HAWKEYE_CONFIG":
				return cfgPath
			default:
				return ""
			}
		},
		Host: fakeHost{usr: true, varp: true, carrier: true},
		TTY:  true,
		Exec: ex,
	}
	code, out, errb := runConsult(t, env)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, errb)
	}
	want := knowledge.RemountPlaybookCommands()
	if ex.Calls != len(want) {
		t.Fatalf("land calls=%d want %d out=%s", ex.Calls, len(want), out)
	}
	got := make([]string, 0, len(ex.Argv))
	for _, a := range ex.Argv {
		got = append(got, strings.Join(a, " "))
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("landed argv %q want %q", got, want)
	}
	b, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"apply"`) {
		t.Fatalf("apply must be audited: %s", b)
	}
}

// TestConsult_TTY_YesAppliesPrintedPlaybookArgv is the lie-catcher:
// TTY prints a playbook, then y must exec THAT playbook's stored argv,
// not makePlan's echo <query> stub and not zfs set readonly=off <rootpool>.
func TestConsult_TTY_YesAppliesPrintedPlaybookArgv(t *testing.T) {
	ex := &apply.CountingExecutor{}
	dir := playbookDir(t)
	cfgPath, _ := auditConfig(t)
	env := cli.Env{
		Args:  []string{"hawkeye", "--config", cfgPath, "--yes", "consult", "ZFS", "root", "is", "read-only", "after", "boot"},
		Stdin: strings.NewReader("y\n"),
		Getenv: func(k string) string {
			switch k {
			case "HAWKEYE_KNOWLEDGE_PATH":
				return dir
			case "HAWKEYE_CONFIG":
				return cfgPath
			default:
				return ""
			}
		},
		Host: fakeHost{ro: true, rescue: true, usr: true, varp: true},
		TTY:  true,
		Exec: ex,
	}
	code, out, errb := runConsult(t, env)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, errb)
	}
	printed := printedPlaybookCommands(out)
	if len(printed) == 0 {
		t.Fatalf("TTY must print the playbook commands before the prompt:\n%s", out)
	}
	want := knowledge.RemountPlaybookCommands()
	if strings.Join(printed, "\n") != strings.Join(want, "\n") {
		t.Fatalf("printed playbook %q want stored %q\nout=%s", printed, want, out)
	}
	got := make([]string, 0, len(ex.Argv))
	for _, a := range ex.Argv {
		got = append(got, strings.Join(a, " "))
	}
	if strings.Join(got, "\n") != strings.Join(printed, "\n") {
		t.Fatalf("y applied argv %q, not the printed playbook %q", got, printed)
	}
	joined := strings.Join(got, "\n")
	for _, junk := range []string{"echo ZFS", "echo <query>", "<rootpool>"} {
		if strings.Contains(joined, junk) {
			t.Fatalf("y applied stub %q: %q", junk, got)
		}
	}
}

func printedPlaybookCommands(out string) []string {
	cut := strings.Index(out, "Apply these steps?")
	if cut < 0 {
		return nil
	}
	var cmds []string
	for _, line := range strings.Split(out[:cut], "\n") {
		if !strings.HasPrefix(line, "  ") {
			continue
		}
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		switch fields[0] {
		case "export", "mount", "zfs", "echo", "sysctl", "service", "kenv":
			cmds = append(cmds, line)
		}
	}
	return cmds
}
