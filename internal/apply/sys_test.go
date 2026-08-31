// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package apply_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/revytechinc/hawkeye/internal/apply"
)

func TestSysExecutor_Echo(t *testing.T) {
	out, errOut, err := (&apply.SysExecutor{}).Run([]string{"echo", "hawkeye-skeleton"})
	if err != nil {
		t.Fatal(err)
	}
	if errOut != "" {
		t.Fatal(errOut)
	}
	if out == "" {
		t.Fatal("empty stdout")
	}
}

func TestSysExecutor_Empty(t *testing.T) {
	if _, _, err := (&apply.SysExecutor{}).Run(nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestSysExecutor_StoredShellLine(t *testing.T) {
	out, errOut, err := (&apply.SysExecutor{}).Run([]string{"echo hawkeye-shell && echo ok"})
	if err != nil {
		t.Fatal(err, errOut)
	}
	if !strings.Contains(out, "hawkeye-shell") {
		t.Fatalf("stored playbook line must run via sh -c: %q %q", out, errOut)
	}
}

func TestSysExecutor_PersistsShellEnvAcrossSteps(t *testing.T) {
	ex := &apply.SysExecutor{}
	t.Cleanup(func() { _ = ex.Close() })
	plan := apply.Plan{
		ID:     "remount-env",
		Source: "knowledge",
		Steps: []apply.Step{
			{ID: "1", Action: "ROOTDS", Argv: []string{"ROOTDS=/export/hawkeye-rootds-fixture"}, Privileged: true},
			{ID: "2", Action: "printf", Argv: []string{`printf '%s\n' "$ROOTDS"`}, Privileged: true},
		},
	}
	res, err := apply.Execute(plan, apply.ModeApply, apply.ActorOperator, ex, apply.NopAuditor{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Applied || res.DryRun {
		t.Fatalf("%+v", res)
	}
	joined := ""
	for _, s := range res.Steps {
		joined += s.Output
	}
	if !strings.Contains(joined, "/export/hawkeye-rootds-fixture") {
		t.Fatalf("ROOTDS must persist across playbook steps in one shell session, not a new /bin/sh -c per line: %+v", res)
	}
	if strings.TrimSpace(joined) == "ok" || strings.TrimSpace(joined) == "ok\nok" {
		t.Fatal("CountingExecutor omits the assignment; this test requires a real executor")
	}
}

func TestSysExecutor_CloseAndShellFailure(t *testing.T) {
	var unused *apply.SysExecutor
	if err := unused.Close(); err != nil {
		t.Fatal(err)
	}
	ex := &apply.SysExecutor{}
	t.Cleanup(func() { _ = ex.Close() })
	out, errOut, err := ex.Run([]string{"echo hi >&2; echo out"})
	if err != nil {
		t.Fatal(err, errOut)
	}
	if !strings.Contains(out, "out") {
		t.Fatalf("%q %q", out, errOut)
	}
	_, _, err = ex.Run([]string{"false && true"})
	if err == nil {
		t.Fatal("expected shell failure")
	}
}

func TestSysExecutor_SessionExitRestarts(t *testing.T) {
	ex := &apply.SysExecutor{}
	t.Cleanup(func() { _ = ex.Close() })
	if _, _, err := ex.Run([]string{"exit 0"}); err == nil {
		t.Fatal("exit ends the session; expect read error")
	}
	out, _, err := ex.Run([]string{"echo hawkeye-session-restart"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "hawkeye-session-restart") {
		t.Fatalf("%q", out)
	}
}

func TestSysExecutor_MissingShell(t *testing.T) {
	ex := &apply.SysExecutor{Shell: filepath.Join(t.TempDir(), "no-such-sh")}
	if _, _, err := ex.Run([]string{"echo hi && true"}); err == nil {
		t.Fatal("expected start failure")
	}
}
