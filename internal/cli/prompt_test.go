// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/revytechinc/hawkeye/internal/apply"
	"github.com/revytechinc/hawkeye/internal/config"
)

func TestParseApplyChoice(t *testing.T) {
	cases := []struct {
		in   string
		want applyChoice
	}{
		{"y", choiceYes},
		{"YES", choiceYes},
		{" yes ", choiceYes},
		{"e", choiceEdit},
		{"EDIT", choiceEdit},
		{"n", choiceNo},
		{"no", choiceNo},
		{"", choiceNo},
		{"  ", choiceNo},
		{"maybe", choiceInvalid},
		{"x", choiceInvalid},
	}
	for _, tc := range cases {
		if got := parseApplyChoice(tc.in); got != tc.want {
			t.Fatalf("%q: got %v want %v", tc.in, got, tc.want)
		}
	}
}

func TestReaderIsTTY(t *testing.T) {
	if readerIsTTY(bytes.NewBufferString("x")) {
		t.Fatal("buffer is not a TTY")
	}
	f, err := os.CreateTemp(t.TempDir(), "regular")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if readerIsTTY(f) {
		t.Fatal("regular file is not a TTY")
	}
}

func TestEditorArgv(t *testing.T) {
	got := editorArgv(func(string) string { return "" }, "p.json")
	if got[0] != "vi" || got[len(got)-1] != "p.json" {
		t.Fatalf("default vi: %v", got)
	}
	got = editorArgv(func(k string) string {
		if k == "VISUAL" {
			return "emacsclient -c"
		}
		return "nano"
	}, "p.json")
	want := []string{"emacsclient", "-c", "p.json"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("VISUAL: got %v want %v", got, want)
	}
	got = editorArgv(func(k string) string {
		if k == "EDITOR" {
			return "nano"
		}
		return ""
	}, "p.json")
	if got[0] != "nano" {
		t.Fatalf("EDITOR: %v", got)
	}
}

func TestParseEditedPlan_JSONAndCommands(t *testing.T) {
	orig := apply.Plan{
		ID:     "consult-plan",
		Source: "knowledge",
		Steps: []apply.Step{{
			ID: "1", Action: "unlock-rw", Argv: []string{"zfs", "set", "readonly=off", "old"}, Privileged: true,
		}},
	}
	p, ok := parseEditedPlan([]byte(`{"id":"edited","source":"operator","steps":[{"id":"1","action":"echo","argv":["echo","ok"],"privileged":false}]}`), orig)
	if !ok || p.ID != "edited" || len(p.Steps) != 1 || p.Steps[0].Argv[1] != "ok" {
		t.Fatalf("json: %+v ok=%v", p, ok)
	}
	p, ok = parseEditedPlan([]byte("echo hawkeye-edit-ok\n"), orig)
	if !ok || len(p.Steps) != 1 || strings.Join(p.Steps[0].Argv, " ") != "echo hawkeye-edit-ok" {
		t.Fatalf("commands: %+v ok=%v", p, ok)
	}
	if !p.Privileged() {
		t.Fatal("command-list edit must keep privileged from the recommended plan")
	}
	if _, ok = parseEditedPlan([]byte("   \n"), orig); ok {
		t.Fatal("empty must abort")
	}
	if _, ok = parseEditedPlan([]byte("{"), orig); ok {
		t.Fatal("bad json must abort")
	}
	if _, ok = parseEditedPlan([]byte(`{"id":"x","steps":[]}`), orig); ok {
		t.Fatal("empty steps must abort")
	}
}

func TestRedactPlan_KeepsJSONValid(t *testing.T) {
	p := apply.Plan{
		ID: "p", Source: "operator",
		Summary: "password=fake-password-for-tests-only",
		Steps:   []apply.Step{{ID: "1", Action: "echo", Argv: []string{"echo", "x"}}},
	}
	got := redactPlan(p)
	if strings.Contains(got.Summary, "fake-password-for-tests-only") {
		t.Fatal(got.Summary)
	}
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var round apply.Plan
	if err := json.Unmarshal(b, &round); err != nil {
		t.Fatal(err)
	}
}

func TestParseEditedPlan_Redacts(t *testing.T) {
	orig := apply.Plan{ID: "p", Steps: []apply.Step{{ID: "1", Argv: []string{"true"}}}}
	raw := []byte(`{"id":"p","source":"operator","summary":"password=fake-password-for-tests-only","steps":[{"id":"1","action":"echo","argv":["echo","x"],"privileged":false}]}`)
	p, ok := parseEditedPlan(raw, orig)
	if !ok {
		t.Fatal("ok")
	}
	if strings.Contains(p.Summary, "fake-password-for-tests-only") {
		t.Fatalf("summary leaked: %q", p.Summary)
	}
}

func TestEditPlanFileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	orig := apply.Plan{
		ID: "consult-plan", Source: "knowledge",
		Steps: []apply.Step{{ID: "1", Action: "echo", Argv: []string{"echo", "old"}}},
	}
	editor := func(path string) error {
		return os.WriteFile(path, []byte("echo hawkeye-edit-ok\n"), 0o600)
	}
	p, ok := editPlan(Env{Editor: editor, Getenv: func(string) string { return dir }}, orig)
	if !ok {
		t.Fatal("edit")
	}
	if strings.Join(p.Steps[0].Argv, " ") != "echo hawkeye-edit-ok" {
		t.Fatalf("%+v", p)
	}
}

func TestEditPlan_AbortEmptyAndError(t *testing.T) {
	orig := apply.Plan{ID: "p", Steps: []apply.Step{{ID: "1", Argv: []string{"true"}}}}
	if _, ok := editPlan(Env{Editor: func(string) error { return os.ErrPermission }}, orig); ok {
		t.Fatal("editor error must abort")
	}
	if _, ok := editPlan(Env{Editor: func(path string) error {
		return os.WriteFile(path, []byte("  \n"), 0o600)
	}}, orig); ok {
		t.Fatal("empty file must abort")
	}
}

func TestApplyAuditor_MkdirFail(t *testing.T) {
	notdir := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(notdir, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := applyAuditor(config.Config{AuditLog: filepath.Join(notdir, "audit.log")}, apply.ModeApply)
	if err == nil {
		t.Fatal("expected mkdir fail on apply")
	}
	a, err := applyAuditor(config.Config{AuditLog: filepath.Join(notdir, "audit.log")}, apply.ModeDryRun)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := a.(apply.NopAuditor); !ok {
		t.Fatalf("%T", a)
	}
	a, err = applyAuditor(config.Config{}, apply.ModeApply)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := a.(apply.NopAuditor); !ok {
		t.Fatalf("%T", a)
	}
}

func TestDefaultEdit_VISUAL(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "ed")
	body := "#!/bin/sh\nprintf '%s\n' '{\"id\":\"from-visual\",\"steps\":[{\"id\":\"1\",\"action\":\"echo\",\"argv\":[\"echo\",\"from-visual\"],\"privileged\":false}]}' > \"$1\"\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	p, ok := editPlan(Env{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Getenv: func(k string) string {
			switch k {
			case "VISUAL":
				return script
			case "TMPDIR":
				return dir
			default:
				return ""
			}
		},
	}, apply.Plan{ID: "old", Steps: []apply.Step{{ID: "1", Argv: []string{"false"}}}})
	if !ok || p.ID != "from-visual" {
		t.Fatalf("%+v ok=%v", p, ok)
	}
}

func TestEditPlan_BadTMPDIR(t *testing.T) {
	_, ok := editPlan(Env{Getenv: func(k string) string {
		if k == "TMPDIR" {
			return filepath.Join(t.TempDir(), "missing-dir")
		}
		return ""
	}}, apply.Plan{ID: "p", Steps: []apply.Step{{ID: "1", Argv: []string{"true"}}}})
	if ok {
		t.Fatal("CreateTemp in missing dir must abort")
	}
}

func TestParseEditedPlan_CommentsOnly(t *testing.T) {
	orig := apply.Plan{ID: "p", Steps: []apply.Step{{ID: "1", Argv: []string{"true"}}}}
	if _, ok := parseEditedPlan([]byte("# just a comment\n\n"), orig); ok {
		t.Fatal("comments only must abort")
	}
}

func TestPrintApply_AuditorFail(t *testing.T) {
	notdir := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(notdir, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	var out, errb strings.Builder
	code := printApply(Env{Stdout: &out, Stderr: &errb, Exec: &apply.CountingExecutor{}}, config.Config{AuditLog: filepath.Join(notdir, "audit.log")}, apply.Plan{ID: "p"}, apply.ModeApply)
	if code == 0 {
		t.Fatal("expected audit mkdir failure")
	}
	if !strings.Contains(errb.String(), "audit log") {
		t.Fatal(errb.String())
	}
}

func TestPrintApply_StepError(t *testing.T) {
	var out, errb strings.Builder
	code := printApply(Env{
		Stdout: &out,
		Stderr: &errb,
		Exec:   failExec{},
	}, config.Config{}, apply.Plan{ID: "p", Steps: []apply.Step{{ID: "1", Argv: []string{"nope"}}}}, apply.ModeApply)
	if code != 0 {
		t.Fatalf("step error is recorded, not fatal: %d %s", code, errb.String())
	}
	if !strings.Contains(out.String(), "boom") {
		t.Fatal(out.String())
	}
}

type failExec struct{}

func (failExec) Run([]string) (string, string, error) {
	return "", "boom", errors.New("boom")
}

func TestEditPlan_UsesTempDir(t *testing.T) {
	dir := t.TempDir()
	var saw string
	orig := apply.Plan{ID: "p", Steps: []apply.Step{{ID: "1", Argv: []string{"true"}}}}
	_, _ = editPlan(Env{
		Getenv: func(k string) string {
			if k == "TMPDIR" {
				return dir
			}
			return ""
		},
		Editor: func(path string) error {
			saw = path
			return os.WriteFile(path, []byte(`{"id":"p","steps":[{"id":"1","argv":["true"]}]}`), 0o600)
		},
	}, orig)
	if saw == "" || filepath.Dir(saw) != dir {
		t.Fatalf("temp path %q dir %q", saw, dir)
	}
}

func TestMCPApply_UnprivilegedYesRunsProcess(t *testing.T) {
	got, err := mcpApply(Env{}, config.Config{}, apply.Plan{
		ID:     "p",
		Source: "operator",
		Steps:  []apply.Step{{ID: "1", Action: "echo", Argv: []string{"echo", "hawkeye-mcp-real-exec"}}},
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	res, ok := got.(apply.Result)
	if !ok {
		t.Fatalf("%T", got)
	}
	if res.DryRun || !res.Applied {
		t.Fatalf("%+v", res)
	}
	if len(res.Steps) != 1 || !strings.Contains(res.Steps[0].Output, "hawkeye-mcp-real-exec") {
		t.Fatalf("must run SysExecutor, not CountingExecutor: %+v", res)
	}
}

func TestMCPApply_PrivilegedYesDryRun(t *testing.T) {
	ex := &apply.CountingExecutor{}
	got, err := mcpApply(Env{Exec: ex}, config.Config{}, apply.Plan{
		ID:     "p",
		Source: "knowledge",
		Steps:  []apply.Step{{ID: "1", Privileged: true, Argv: []string{"echo", "hawkeye-mcp-real-exec"}}},
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	res := got.(apply.Result)
	if !res.DryRun || res.Applied {
		t.Fatalf("privileged MCP stays dry-run: %+v", res)
	}
	if ex.Calls != 0 {
		t.Fatal(ex.Calls)
	}
}

func TestMCPApply_NoYesIsDryRun(t *testing.T) {
	got, err := mcpApply(Env{}, config.Config{}, apply.Plan{
		Steps: []apply.Step{{ID: "1", Argv: []string{"echo", "hawkeye-mcp-real-exec"}}},
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	res := got.(apply.Result)
	if !res.DryRun || res.Applied {
		t.Fatalf("%+v", res)
	}
}
