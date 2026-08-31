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
	"github.com/revytechinc/hawkeye/internal/config"
	"github.com/revytechinc/hawkeye/internal/knowledge"
	"github.com/revytechinc/hawkeye/internal/probe"
)

func runConsult(t *testing.T, env cli.Env) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	env.Stdout = &out
	env.Stderr = &errb
	if env.Getenv == nil {
		env.Getenv = func(string) string { return "" }
	}
	if env.Host == nil {
		env.Host = fakeHost{ro: true, rescue: true}
	}
	code := cli.RunEnv(env)
	return code, out.String(), errb.String()
}

func knowledgeDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := knowledge.CreateTestDB(filepath.Join(dir, "knowledge.sqlite")); err != nil {
		t.Fatal(err)
	}
	return dir
}

func consultPlaybookEnv(t *testing.T, args []string, stdin string, host probe.Host, tty bool, exec apply.Executor, editor func(string) error) cli.Env {
	t.Helper()
	kd := playbookDir(t)
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

func auditConfig(t *testing.T) (cfgPath, auditPath string) {
	t.Helper()
	dir := t.TempDir()
	auditPath = filepath.Join(dir, "audit.log")
	b, err := config.InitJSON()
	if err != nil {
		t.Fatal(err)
	}
	var c config.Config
	if err := json.Unmarshal(b, &c); err != nil {
		t.Fatal(err)
	}
	c.AuditLog = auditPath
	raw, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	cfgPath = filepath.Join(dir, "config.json")
	if err := os.WriteFile(cfgPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return cfgPath, auditPath
}

func consultEnv(t *testing.T, args []string, stdin string, host probe.Host, tty bool, exec apply.Executor, editor func(string) error) cli.Env {
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

func TestConsult_TTY_YesShowsDryRunDefaultNDoesNotMutate(t *testing.T) {
	ex := &apply.CountingExecutor{}
	env := consultEnv(t, []string{"consult", "zfs", "readonly"}, "y\n\n", fakeHost{ro: true, rescue: true}, true, ex, nil)
	code, out, err := runConsult(t, env)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if strings.Contains(out, `"Title"`) || strings.Contains(out, `"hits"`) {
		t.Fatalf("TTY must be operator text, not hits JSON: %s", out)
	}
	if !strings.Contains(out, "ZFS readonly pool") {
		t.Fatalf("want playbook title: %s", out)
	}
	if !strings.Contains(out, "Apply these steps? [y/N/e]") {
		t.Fatalf("want prompt: %s", out)
	}
	if !strings.Contains(out, "dry-run:") {
		t.Fatalf("want dry-run: %s", out)
	}
	if !strings.Contains(out, "Apply for real? [y/N]") {
		t.Fatalf("want second confirm: %s", out)
	}
	if !strings.Contains(out, "nothing applied") {
		t.Fatalf("default N: %s", out)
	}
	if ex.Calls != 0 {
		t.Fatalf("default N mutated: calls=%d", ex.Calls)
	}
}

func TestConsult_TTY_N_NothingApplied(t *testing.T) {
	ex := &apply.CountingExecutor{}
	env := consultEnv(t, []string{"consult", "zfs"}, "n\n", fakeHost{ro: true, rescue: true}, true, ex, nil)
	code, out, err := runConsult(t, env)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if !strings.Contains(out, "nothing applied") {
		t.Fatal(out)
	}
	if strings.Contains(out, "dry-run:") {
		t.Fatalf("n must not apply: %s", out)
	}
	if ex.Calls != 0 {
		t.Fatal(ex.Calls)
	}
}

func TestConsult_TTY_EOF_NothingApplied(t *testing.T) {
	ex := &apply.CountingExecutor{}
	env := consultEnv(t, []string{"consult", "zfs"}, "", fakeHost{ro: true, rescue: true}, true, ex, nil)
	code, out, err := runConsult(t, env)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if !strings.Contains(out, "nothing applied") {
		t.Fatal(out)
	}
	if ex.Calls != 0 {
		t.Fatal(ex.Calls)
	}
}

func TestConsult_TTY_InvalidReprompt(t *testing.T) {
	ex := &apply.CountingExecutor{}
	env := consultEnv(t, []string{"consult", "zfs"}, "maybe\nn\n", fakeHost{ro: true, rescue: true}, true, ex, nil)
	code, out, err := runConsult(t, env)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if strings.Count(out, "Apply these steps? [y/N/e]") < 2 {
		t.Fatalf("want re-prompt: %s", out)
	}
	if ex.Calls != 0 {
		t.Fatal(ex.Calls)
	}
}

func TestConsult_NonTTY_NoPrompt(t *testing.T) {
	ex := &apply.CountingExecutor{}
	env := consultEnv(t, []string{"consult", "zfs", "readonly"}, "y\ny\n", fakeHost{ro: true, rescue: true}, false, ex, nil)
	code, out, err := runConsult(t, env)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if strings.Contains(out, "Apply these steps?") {
		t.Fatalf("non-TTY must not prompt: %s", out)
	}
	trim := strings.TrimSpace(out)
	if strings.HasPrefix(trim, "{") {
		t.Fatalf("non-TTY without --json is operator text, not JSON: %s", out)
	}
	if ex.Calls != 0 {
		t.Fatal(ex.Calls)
	}
}

func TestConsult_JSON_NoPromptEvenOnTTY(t *testing.T) {
	ex := &apply.CountingExecutor{}
	env := consultEnv(t, []string{"--json", "consult", "zfs", "readonly"}, "y\ny\n", fakeHost{ro: true, rescue: true}, true, ex, nil)
	code, out, err := runConsult(t, env)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if strings.Contains(out, "Apply these steps?") {
		t.Fatalf("--json must not prompt: %s", out)
	}
	if !strings.Contains(out, `"query"`) {
		t.Fatalf("--json must stay machine-shaped: %s", out)
	}
	if ex.Calls != 0 {
		t.Fatal(ex.Calls)
	}

	ex = &apply.CountingExecutor{}
	env = consultEnv(t, []string{"consult", "zfs"}, "y\ny\n", fakeHost{ro: true, rescue: true}, true, ex, nil)
	prev := env.Getenv
	env.Getenv = func(k string) string {
		if k == "HAWKEYE_JSON" {
			return "1"
		}
		return prev(k)
	}
	code, out, err = runConsult(t, env)
	if code != 0 {
		t.Fatalf("HAWKEYE_JSON %d %s %s", code, out, err)
	}
	if strings.Contains(out, "Apply these steps?") {
		t.Fatalf("HAWKEYE_JSON must not prompt: %s", out)
	}
}

func TestConsult_TTY_YesThenYesLandsAndAudits(t *testing.T) {
	ex := &apply.CountingExecutor{}
	kd := playbookDir(t)
	cfgPath, auditPath := auditConfig(t)
	env := cli.Env{
		Args:  []string{"hawkeye", "--config", cfgPath, "consult", "ZFS", "root", "is", "read-only", "after", "boot"},
		Stdin: bytes.NewBufferString("y\ny\n"),
		Getenv: func(k string) string {
			switch k {
			case "HAWKEYE_KNOWLEDGE_PATH":
				return kd
			case "HAWKEYE_CONFIG":
				return cfgPath
			default:
				return ""
			}
		},
		Host: fakeHost{usr: true, varp: true},
		TTY:  true,
		Exec: ex,
	}
	code, out, err := runConsult(t, env)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if !strings.Contains(out, "dry-run:") {
		t.Fatalf("must show dry-run before land: %s", out)
	}
	if !strings.Contains(out, `dry-run: zfs set readonly=off "$ROOTDS"`) {
		t.Fatalf("dry-run must be playbook commands: %s", out)
	}
	if !strings.Contains(out, "Apply for real? [y/N]") {
		t.Fatalf("second confirm: %s", out)
	}
	if ex.Calls != len(knowledge.RemountPlaybookCommands()) {
		t.Fatalf("land calls=%d out=%s", ex.Calls, out)
	}
	b, errRead := os.ReadFile(auditPath)
	if errRead != nil {
		t.Fatal(errRead)
	}
	if !strings.Contains(string(b), `"apply"`) {
		t.Fatalf("apply must be audited: %s", b)
	}
}

func TestConsult_TTY_YesFlagSkipsSecondConfirmStillRequiresY(t *testing.T) {
	ex := &apply.CountingExecutor{}
	env := consultPlaybookEnv(t, []string{"--yes", "consult", "ZFS", "root", "is", "read-only", "after", "boot"}, "\n", fakeHost{usr: true, varp: true}, true, ex, nil)
	code, out, err := runConsult(t, env)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if strings.Contains(out, "Apply for real?") {
		t.Fatalf("empty first answer must not land: %s", out)
	}
	if ex.Calls != 0 {
		t.Fatalf("--yes without first y mutated: %d", ex.Calls)
	}

	ex = &apply.CountingExecutor{}
	env = consultPlaybookEnv(t, []string{"--yes", "consult", "ZFS", "root", "is", "read-only", "after", "boot"}, "y\n", fakeHost{usr: true, varp: true}, true, ex, nil)
	code, out, err = runConsult(t, env)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if strings.Contains(out, "Apply for real?") {
		t.Fatalf("--yes should skip second confirm: %s", out)
	}
	if ex.Calls != len(knowledge.RemountPlaybookCommands()) {
		t.Fatalf("consult --yes + y should land playbook: %d %s", ex.Calls, out)
	}
}

func TestConsult_TTY_DryRunFlagDoesNotLand(t *testing.T) {
	ex := &apply.CountingExecutor{}
	env := consultPlaybookEnv(t, []string{"--dry-run", "--yes", "consult", "ZFS", "root", "is", "read-only", "after", "boot"}, "y\ny\n", fakeHost{usr: true, varp: true}, true, ex, nil)
	code, out, err := runConsult(t, env)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if ex.Calls != 0 {
		t.Fatalf("--dry-run must win: %d %s", ex.Calls, out)
	}
	if !strings.Contains(out, `dry-run: zfs set readonly=off "$ROOTDS"`) {
		t.Fatal(out)
	}
}

func TestConsult_TTY_EditRoundTripThenDefaultN(t *testing.T) {
	ex := &apply.CountingExecutor{}
	editor := func(path string) error {
		return os.WriteFile(path, []byte("echo hawkeye-edit-ok\n"), 0o600)
	}
	env := consultEnv(t, []string{"consult", "hello"}, "e\n\n", fakeHost{usr: true, varp: true}, true, ex, editor)
	code, out, err := runConsult(t, env)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if !strings.Contains(out, "echo hawkeye-edit-ok") {
		t.Fatalf("want edited plan: %s", out)
	}
	if !strings.Contains(out, "dry-run: echo hawkeye-edit-ok") {
		t.Fatalf("want dry-run of edit: %s", out)
	}
	if ex.Calls != 0 {
		t.Fatalf("edit then default N mutated: %d", ex.Calls)
	}
	if !strings.Contains(out, "nothing applied") {
		t.Fatal(out)
	}
}

func TestConsult_TTY_EditThenLand(t *testing.T) {
	ex := &apply.CountingExecutor{}
	editor := func(path string) error {
		return os.WriteFile(path, []byte(`{"id":"edited","source":"operator","steps":[{"id":"1","action":"echo","argv":["echo","hawkeye-edit-ok"],"privileged":false}]}`), 0o600)
	}
	env := consultEnv(t, []string{"consult", "hello"}, "e\ny\n", fakeHost{usr: true, varp: true}, true, ex, editor)
	code, out, err := runConsult(t, env)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if ex.Calls != 1 {
		t.Fatalf("edit then y should land after dry-run: %d %s", ex.Calls, out)
	}
	if len(ex.Argv) != 1 || strings.Join(ex.Argv[0], " ") != "echo hawkeye-edit-ok" {
		t.Fatalf("argv %+v", ex.Argv)
	}
}

func TestConsult_TTY_EditAbortDoesNotApply(t *testing.T) {
	ex := &apply.CountingExecutor{}
	env := consultEnv(t, []string{"consult", "hello"}, "e\n", fakeHost{usr: true, varp: true}, true, ex, func(string) error {
		return os.ErrPermission
	})
	code, out, err := runConsult(t, env)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if !strings.Contains(out, "nothing applied") {
		t.Fatal(out)
	}
	if ex.Calls != 0 {
		t.Fatal(ex.Calls)
	}
}

func TestConsult_TTY_LandInvalidReprompt(t *testing.T) {
	ex := &apply.CountingExecutor{}
	env := consultEnv(t, []string{"consult", "hello"}, "y\nmaybe\n\n", fakeHost{usr: true, varp: true}, true, ex, nil)
	code, out, err := runConsult(t, env)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if strings.Count(out, "Apply for real? [y/N]") < 2 {
		t.Fatalf("want land re-prompt: %s", out)
	}
	if ex.Calls != 0 {
		t.Fatal(ex.Calls)
	}
}

func TestConsult_TTY_RedactsSecrets(t *testing.T) {
	ex := &apply.CountingExecutor{}
	env := consultEnv(t, []string{"consult", "password=fake-password-for-tests-only"}, "n\n", fakeHost{usr: true, varp: true}, true, ex, nil)
	code, out, err := runConsult(t, env)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if strings.Contains(out, "fake-password-for-tests-only") {
		t.Fatalf("secret in TTY output: %s", out)
	}
}

func TestRun_DetectsNonTTYStdin(t *testing.T) {
	var out, errb bytes.Buffer
	code := cli.Run([]string{"hawkeye", "consult", "zfs"}, bytes.NewReader(nil), &out, &errb)
	if code != 0 {
		t.Fatalf("%d %s", code, errb.String())
	}
	if strings.Contains(out.String(), "Apply these steps?") {
		t.Fatalf("Run with buffer stdin must not prompt: %s", out.String())
	}
}
