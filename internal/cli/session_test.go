// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/revytechinc/hawkeye/internal/apply"
	"github.com/revytechinc/hawkeye/internal/cli"
	"github.com/revytechinc/hawkeye/internal/knowledge"
	"github.com/revytechinc/hawkeye/internal/probe"
)

func sessionEnv(t *testing.T, args []string, stdin string, host probe.Host, tty bool, exec apply.Executor, editor func(string) error) cli.Env {
	t.Helper()
	kd := knowledgeDir(t)
	cfgPath, _ := auditConfig(t)
	envmap := map[string]string{
		"HAWKEYE_KNOWLEDGE_PATH": kd,
		"HAWKEYE_CONFIG":         cfgPath,
	}
	return cli.Env{
		Args:   append([]string{"hawkeye", "--config", cfgPath}, args...),
		Stdin:  bytes.NewBufferString(stdin),
		Getenv: func(k string) string { return envmap[k] },
		Host:   host,
		TTY:    tty,
		Editor: editor,
		Exec:   exec,
	}
}

func hasSessionPrompt(out string) bool {
	return strings.Contains(out, "\n> ") || strings.HasPrefix(out, "> ")
}

func countSessionPrompts(out string) int {
	// Tests feed stdin without terminal echo, so empty-line re-prompts
	// appear as "> > > " on one line.
	return strings.Count(out, "> ")
}

func assertNoMachineChrome(t *testing.T, out string) {
	t.Helper()
	for _, junk := range []string{`"Title"`, `"hits"`, "hits[]", "llm skipped", `"query":`} {
		if strings.Contains(out, junk) {
			t.Fatalf("session leaked machine chrome %q:\n%s", junk, out)
		}
	}
	trim := strings.TrimSpace(out)
	if trim != "" && json.Valid([]byte(trim)) && strings.HasPrefix(trim, "{") {
		t.Fatalf("session dumped JSON:\n%s", out)
	}
}

func TestSession_TTY_StartsAndQuits(t *testing.T) {
	ex := &apply.CountingExecutor{}
	env := sessionEnv(t, nil, "quit\n", fakeHost{ro: true, rescue: true}, true, ex, nil)
	code, out, err := runConsult(t, env)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if !strings.Contains(out, "hawkeye") {
		t.Fatalf("want banner:\n%s", out)
	}
	if !hasSessionPrompt(out) {
		t.Fatalf("want prompt:\n%s", out)
	}
	if strings.Contains(out, "Apply these steps?") {
		t.Fatalf("quit must not consult:\n%s", out)
	}
	if ex.Calls != 0 {
		t.Fatal(ex.Calls)
	}
	assertNoMachineChrome(t, out)
}

func TestSession_TTY_OneQueryThenNThenQuit(t *testing.T) {
	ex := &apply.CountingExecutor{}
	env := sessionEnv(t, nil, "zfs readonly\nn\nquit\n", fakeHost{ro: true, rescue: true}, true, ex, nil)
	code, out, err := runConsult(t, env)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if !strings.Contains(out, "ZFS readonly pool") {
		t.Fatalf("want playbook title:\n%s", out)
	}
	if !strings.Contains(out, "also:") && !strings.Contains(out, "unlock-rw") {
		t.Fatalf("want human recommendation:\n%s", out)
	}
	if !strings.Contains(out, "Apply these steps? [y/N/e]") {
		t.Fatalf("want apply prompt:\n%s", out)
	}
	if countSessionPrompts(out) < 2 {
		t.Fatalf("N must return to prompt:\n%s", out)
	}
	if ex.Calls != 0 {
		t.Fatalf("N mutated: %d", ex.Calls)
	}
	assertNoMachineChrome(t, out)
}

func TestSession_TTY_EmptyLineDoesNotConsult(t *testing.T) {
	ex := &apply.CountingExecutor{}
	env := sessionEnv(t, nil, "\n\nexit\n", fakeHost{ro: true, rescue: true}, true, ex, nil)
	code, out, err := runConsult(t, env)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if strings.Contains(out, "Apply these steps?") {
		t.Fatalf("empty line must not consult:\n%s", out)
	}
	if countSessionPrompts(out) < 3 {
		t.Fatalf("want re-prompt on empty lines:\n%s", out)
	}
}

func TestSession_TTY_HelpAndQuestion(t *testing.T) {
	ex := &apply.CountingExecutor{}
	env := sessionEnv(t, nil, "help\n?\nq\n", fakeHost{ro: true, rescue: true}, true, ex, nil)
	code, out, err := runConsult(t, env)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if !strings.Contains(strings.ToLower(out), "type the problem") {
		t.Fatalf("want panic-friendly help:\n%s", out)
	}
	if !strings.Contains(out, "y") || !strings.Contains(out, "e") {
		t.Fatalf("help must mention y/e:\n%s", out)
	}
	if strings.Contains(out, "Apply these steps? [y/N/e] ") {
		t.Fatalf("help must not open the apply prompt:\n%s", out)
	}
	if strings.Contains(out, "ZFS readonly pool") {
		t.Fatalf("help must not consult:\n%s", out)
	}
}

func TestSession_TTY_EOFExitsZero(t *testing.T) {
	ex := &apply.CountingExecutor{}
	env := sessionEnv(t, nil, "", fakeHost{ro: true, rescue: true}, true, ex, nil)
	code, out, err := runConsult(t, env)
	if code != 0 {
		t.Fatalf("Ctrl-D must exit 0: %d %s %s", code, out, err)
	}
	if !hasSessionPrompt(out) {
		t.Fatalf("want prompt before EOF:\n%s", out)
	}
	if ex.Calls != 0 {
		t.Fatal(ex.Calls)
	}
}

func TestSession_TTY_YesThenDefaultNReturnsToPrompt(t *testing.T) {
	ex := &apply.CountingExecutor{}
	env := sessionEnv(t, nil, "zfs readonly\ny\n\nquit\n", fakeHost{ro: true, rescue: true}, true, ex, nil)
	code, out, err := runConsult(t, env)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if !strings.Contains(out, "dry-run:") {
		t.Fatalf("y must dry-run:\n%s", out)
	}
	if !strings.Contains(out, "Apply for real? [y/N]") {
		t.Fatalf("want second confirm:\n%s", out)
	}
	if ex.Calls != 0 {
		t.Fatalf("Enter on second confirm must not land: %d", ex.Calls)
	}
	if countSessionPrompts(out) < 2 {
		t.Fatalf("must return to session:\n%s", out)
	}
}

func TestSession_TTY_EditThenNReturnsToPrompt(t *testing.T) {
	ex := &apply.CountingExecutor{}
	editor := func(path string) error {
		return os.WriteFile(path, []byte("echo hawkeye-session-edit\n"), 0o600)
	}
	env := sessionEnv(t, nil, "hello\ne\n\nquit\n", fakeHost{usr: true, varp: true}, true, ex, editor)
	code, out, err := runConsult(t, env)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if !strings.Contains(out, "echo hawkeye-session-edit") {
		t.Fatalf("want edited plan:\n%s", out)
	}
	if ex.Calls != 0 {
		t.Fatalf("edit then N mutated: %d", ex.Calls)
	}
	if countSessionPrompts(out) < 2 {
		t.Fatalf("must return to session:\n%s", out)
	}
}

func TestSession_TTY_PositionalFirstQueryThenStay(t *testing.T) {
	ex := &apply.CountingExecutor{}
	env := sessionEnv(t, []string{"ZFS", "root", "is", "read-only", "after", "boot"}, "n\nquit\n", fakeHost{ro: true, rescue: true}, true, ex, nil)
	code, out, err := runConsult(t, env)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if !strings.Contains(out, "ZFS readonly pool") {
		t.Fatalf("positional args are the first query:\n%s", out)
	}
	if !strings.Contains(out, "Apply these steps? [y/N/e]") {
		t.Fatalf("want apply prompt:\n%s", out)
	}
	if countSessionPrompts(out) < 2 {
		t.Fatalf("must stay in session after first query:\n%s", out)
	}
	if ex.Calls != 0 {
		t.Fatal(ex.Calls)
	}
	assertNoMachineChrome(t, out)
}

func TestSession_NonTTY_NoArgs_NoHang(t *testing.T) {
	code, out, err := run(t, []string{}, "", fakeHost{}, nil)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if hasSessionPrompt(out) {
		t.Fatalf("non-TTY must not enter REPL:\n%s", out)
	}
	if !strings.Contains(strings.ToLower(out), "terminal") {
		t.Fatalf("must say run hawkeye on a terminal:\n%s", out)
	}
}

func TestSession_JSON_NeverEntersREPL(t *testing.T) {
	ex := &apply.CountingExecutor{}
	env := sessionEnv(t, []string{"--json"}, "zfs readonly\nquit\n", fakeHost{ro: true, rescue: true}, true, ex, nil)
	code, out, err := runConsult(t, env)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if hasSessionPrompt(out) {
		t.Fatalf("--json must not enter REPL:\n%s", out)
	}
	if strings.Contains(out, "Apply these steps?") {
		t.Fatalf("--json must not prompt:\n%s", out)
	}

	env = sessionEnv(t, []string{"--json", "zfs", "readonly"}, "y\ny\n", fakeHost{ro: true, rescue: true}, true, ex, nil)
	code, out, err = runConsult(t, env)
	if code != 0 {
		t.Fatalf("json query %d %s %s", code, out, err)
	}
	if hasSessionPrompt(out) || strings.Contains(out, "Apply these steps?") {
		t.Fatalf("--json query must be one-shot machine JSON:\n%s", out)
	}
	if !json.Valid([]byte(strings.TrimSpace(out))) || !strings.Contains(out, `"query"`) {
		t.Fatalf("--json positional must dump consult JSON:\n%s", out)
	}
	if ex.Calls != 0 {
		t.Fatal(ex.Calls)
	}
}

func TestSession_KnownSubcommandsStillWork(t *testing.T) {
	code, out, err := run(t, []string{"--help"}, "", fakeHost{}, nil)
	if code != 0 || !strings.Contains(out, "consult") {
		t.Fatalf("help %d %s %s", code, out, err)
	}
	code, out, err = run(t, []string{"--version"}, "", fakeHost{}, nil)
	if code != 0 || !strings.Contains(out, "0.1.0") {
		t.Fatalf("version %d %s %s", code, out, err)
	}
	code, out, err = run(t, []string{"doctor"}, "", fakeHost{usr: true, varp: true}, nil)
	if !strings.Contains(strings.ToLower(out), "doctor") {
		t.Fatalf("doctor %d %s %s", code, out, err)
	}
	if hasSessionPrompt(out) {
		t.Fatalf("doctor must not enter REPL:\n%s", out)
	}

	dir := t.TempDir()
	if err := knowledge.CreateTestDB(filepath.Join(dir, "knowledge.sqlite")); err != nil {
		t.Fatal(err)
	}
	code, out, err = run(t, []string{"consult", "zfs", "readonly"}, "", fakeHost{ro: true, rescue: true}, map[string]string{"HAWKEYE_KNOWLEDGE_PATH": dir})
	if code != 0 {
		t.Fatalf("consult %d %s %s", code, out, err)
	}
	if hasSessionPrompt(out) {
		t.Fatalf("consult subcommand is one-shot, not the REPL:\n%s", out)
	}
	if !strings.Contains(out, "ZFS readonly pool") {
		t.Fatalf("consult still human:\n%s", out)
	}
}

func TestSession_RedactsSecrets(t *testing.T) {
	ex := &apply.CountingExecutor{}
	env := sessionEnv(t, nil, "password=fake-password-for-tests-only\nn\nquit\n", fakeHost{usr: true, varp: true}, true, ex, nil)
	code, out, err := runConsult(t, env)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if strings.Contains(out, "fake-password-for-tests-only") {
		t.Fatalf("secret leaked:\n%s", out)
	}
}

func TestSession_MCPUnchanged(t *testing.T) {
	in := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}` + "\n"
	code, out, err := run(t, []string{"mcp", "--stdio"}, in, fakeHost{usr: true, varp: true}, nil)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if hasSessionPrompt(out) {
		t.Fatalf("MCP must not enter REPL:\n%s", out)
	}
	if !strings.Contains(out, "hawkeye") {
		t.Fatal(out)
	}
}
