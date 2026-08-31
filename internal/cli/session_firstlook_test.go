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
	"github.com/revytechinc/hawkeye/internal/probe"
)

func firstLookBlock(out string) string {
	rest := strings.TrimPrefix(out, "hawkeye\n")
	if i := strings.Index(rest, "> "); i >= 0 {
		return rest[:i]
	}
	return rest
}

func panicSources(t *testing.T) probe.Sources {
	t.Helper()
	root := t.TempDir()
	mustWrite := func(rel, body string) {
		t.Helper()
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("etc/fstab", `/dev/gpt/root / ufs rw 1 1
/dev/gpt/var /var ufs rw 2 2
`)
	mustWrite("etc/rc.conf", "sshd_enable=\"YES\"\n")
	mustWrite("etc/resolv.conf", "")
	return probe.Sources{
		Root:     root,
		ReadFile: os.ReadFile,
		Stat:     os.Stat,
		MountTable: func() (string, error) {
			return "/dev/gpt/root / ufs ro 0 0\n", nil
		},
		Disk: func(path string) (probe.DiskUse, bool) {
			if path == "/" {
				return probe.DiskUse{TotalBytes: 100, FreeBytes: 0, TotalInodes: 10, FreeInodes: 1}, true
			}
			return probe.DiskUse{}, false
		},
		Ifaces: func() ([]probe.IfaceStatus, error) {
			return []probe.IfaceStatus{{Name: "em0", Up: true, CarrierKnown: true, Carrier: false}}, nil
		},
		Routes: func() (string, error) { return "127.0.0.1          link#2             UH        lo0\n", nil },
	}
}

func TestSession_TTY_FirstLookBeforePrompt(t *testing.T) {
	ex := &apply.CountingExecutor{}
	env := sessionEnv(t, nil, "quit\n", fakeHost{ro: true, rescue: true}, true, ex, nil)
	env.Sources = panicSources(t)
	code, out, err := runConsult(t, env)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	head := firstLookBlock(out)
	if strings.TrimSpace(head) == "" {
		t.Fatalf("TTY session must print host first-look before >:\n%s", out)
	}
	if !strings.Contains(strings.ToLower(head), "read-only") {
		t.Fatalf("want RO root before prompt:\n%s", out)
	}
	if !strings.Contains(head, "/var") || !strings.Contains(strings.ToLower(head), "not mounted") {
		t.Fatalf("want fstab vs mounts before prompt:\n%s", out)
	}
	if !strings.Contains(head, "sshd") {
		t.Fatalf("want missing rc enable script before prompt:\n%s", out)
	}
	if !strings.Contains(strings.ToLower(head), "full") {
		t.Fatalf("want full disk before prompt:\n%s", out)
	}
	if !strings.Contains(strings.ToLower(head), "carrier") {
		t.Fatalf("want no carrier before prompt:\n%s", out)
	}
	if !hasSessionPrompt(out) {
		t.Fatalf("want > after first-look:\n%s", out)
	}
	if strings.Index(out, "read-only") > strings.Index(out, "> ") && strings.Index(out, "> ") >= 0 {
		// first-look must precede the first prompt
		if !strings.Contains(head, "read-only") {
			t.Fatalf("first-look after prompt:\n%s", out)
		}
	}
	if strings.Contains(head, "pidfile") || strings.Contains(head, "hawkeye doctor") {
		t.Fatalf("first-look must not be hawkeye doctor:\n%s", out)
	}
	if strings.Contains(head, `"findings"`) || strings.Contains(head, `"area"`) {
		t.Fatalf("first-look must be human text, not JSON:\n%s", out)
	}
	assertNoMachineChrome(t, out)
}

func TestSession_TTY_HostOnlyFirstLookWhenSourcesEmpty(t *testing.T) {
	ex := &apply.CountingExecutor{}
	env := sessionEnv(t, nil, "quit\n", fakeHost{ro: true, rescue: true}, true, ex, nil)
	code, out, err := runConsult(t, env)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	head := firstLookBlock(out)
	if !strings.Contains(strings.ToLower(head), "read-only") {
		t.Fatalf("RO fake host must first-look the root before >:\n%s", out)
	}
	if strings.Contains(head, "pidfile") {
		t.Fatalf("must not mention hawkeye pidfile:\n%s", out)
	}
}

func TestSession_JSON_EmptyIsInspectNotREPL(t *testing.T) {
	ex := &apply.CountingExecutor{}
	env := sessionEnv(t, []string{"--json"}, "", fakeHost{ro: true, rescue: true}, true, ex, nil)
	env.Sources = panicSources(t)
	code, out, err := runConsult(t, env)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if hasSessionPrompt(out) {
		t.Fatalf("--json must not enter REPL:\n%s", out)
	}
	if !json.Valid([]byte(strings.TrimSpace(out))) {
		t.Fatalf("--json empty session must dump inspect JSON:\n%s", out)
	}
	var rep probe.Report
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &rep); err != nil {
		t.Fatalf("inspect JSON: %v\n%s", err, out)
	}
	if len(rep.Findings) == 0 {
		t.Fatalf("inspect JSON needs findings:\n%s", out)
	}
	if ex.Calls != 0 {
		t.Fatal(ex.Calls)
	}
}

func TestSession_InspectCommand_HumanNotDoctor(t *testing.T) {
	ex := &apply.CountingExecutor{}
	env := sessionEnv(t, []string{"inspect"}, "", fakeHost{ro: true, rescue: true}, true, ex, nil)
	env.Sources = panicSources(t)
	code, out, err := runConsult(t, env)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if hasSessionPrompt(out) {
		t.Fatalf("inspect must not enter REPL:\n%s", out)
	}
	if !strings.Contains(strings.ToLower(out), "read-only") {
		t.Fatalf("inspect must first-look the host:\n%s", out)
	}
	if strings.Contains(out, "pidfile") || strings.Contains(strings.ToLower(out), "hawkeye doctor") {
		t.Fatalf("inspect is not doctor:\n%s", out)
	}
	if json.Valid([]byte(strings.TrimSpace(out))) && strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Fatalf("default inspect is human:\n%s", out)
	}
}

func TestSession_LandedApplyReinspects(t *testing.T) {
	ex := &apply.CountingExecutor{}
	editor := func(path string) error {
		return os.WriteFile(path, []byte("echo hawkeye-firstlook-reinspect\n"), 0o600)
	}
	env := sessionEnv(t, []string{"--yes"}, "hello\ne\ny\nquit\n", fakeHost{usr: true, varp: true}, true, ex, editor)
	env.Sources = panicSources(t)
	code, out, err := runConsult(t, env)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if ex.Calls == 0 {
		t.Fatal("expected landed apply")
	}
	if !strings.Contains(out, "applied") {
		t.Fatalf("want applied:\n%s", out)
	}
	// first-look at start plus compact re-inspect after land
	if strings.Count(strings.ToLower(out), "read-only") < 2 {
		t.Fatalf("landed apply must re-run first-look:\n%s", out)
	}
}

func TestInspect_JSONFlag(t *testing.T) {
	ex := &apply.CountingExecutor{}
	env := sessionEnv(t, []string{"--json", "inspect"}, "", fakeHost{ro: true, rescue: true}, true, ex, nil)
	env.Sources = panicSources(t)
	code, out, err := runConsult(t, env)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if hasSessionPrompt(out) {
		t.Fatalf("--json inspect must not enter REPL:\n%s", out)
	}
	var rep probe.Report
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &rep); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if len(rep.Findings) == 0 {
		t.Fatal(out)
	}
}

func TestDoctor_StillSelfHealthNotHostFirstLook(t *testing.T) {
	code, out, err := run(t, []string{"doctor"}, "", fakeHost{usr: true, varp: true}, nil)
	_ = code
	_ = err
	if !strings.Contains(strings.ToLower(out), "doctor") {
		t.Fatalf("doctor must stay service health:\n%s", out)
	}
	if strings.Contains(out, "fstab") && !strings.Contains(out, "pidfile") {
		t.Fatalf("doctor replaced by host inspect:\n%s", out)
	}
	if !strings.Contains(out, "pidfile") && !strings.Contains(out, "config") {
		t.Fatalf("doctor must still mention self-health:\n%s", out)
	}
	if hasSessionPrompt(out) {
		t.Fatalf("doctor must not enter REPL:\n%s", out)
	}
}
