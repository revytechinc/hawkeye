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

	"github.com/revytechinc/hawkeye/internal/cli"
	"github.com/revytechinc/hawkeye/internal/config"
	"github.com/revytechinc/hawkeye/internal/knowledge"
	"github.com/revytechinc/hawkeye/internal/probe"
)

type fakeHost struct {
	ro      bool
	usr     bool
	varp    bool
	rescue  bool
	carrier bool
	gpu     bool
	sec     int
	secOK   bool
}

func (f fakeHost) SysctlInt(string) (int, bool) { return f.sec, f.secOK }
func (f fakeHost) PathExists(path string) bool {
	switch path {
	case "/usr":
		return f.usr
	case "/var":
		return f.varp
	case "/rescue":
		return f.rescue
	default:
		return true
	}
}
func (f fakeHost) PathWritable(path string) bool {
	if path == "/" {
		return !f.ro
	}
	return !f.ro
}
func (f fakeHost) MountReadOnly(path string) bool { return path == "/" && f.ro }
func (f fakeHost) NetworkCarrier() bool           { return f.carrier }
func (f fakeHost) GPUPresent() bool               { return f.gpu }

func run(t *testing.T, args []string, stdin string, host probe.Host, env map[string]string) (int, string, string) {
	t.Helper()
	in := bytes.NewBufferString(stdin)
	var out, err bytes.Buffer
	code := cli.RunEnv(cli.Env{
		Args:   append([]string{"hawkeye"}, args...),
		Stdin:  in,
		Stdout: &out,
		Stderr: &err,
		Getenv: func(k string) string { return env[k] },
		Host:   host,
	})
	return code, out.String(), err.String()
}

func TestInitAndCheckConfig(t *testing.T) {
	dir := t.TempDir()
	code, out, err := run(t, []string{"init", dir}, "", fakeHost{usr: true, varp: true}, nil)
	if code != 0 {
		t.Fatalf("init %d %s %s", code, out, err)
	}
	p := filepath.Join(dir, "config.json")
	if _, e := os.Stat(p); e != nil {
		t.Fatal(e)
	}
	if err := config.CheckFile(p); err != nil {
		t.Fatal(err)
	}
	code, out, err = run(t, []string{"--config", p, "--check-config"}, "", fakeHost{}, nil)
	if code != 0 {
		t.Fatalf("check-config %d %s %s", code, out, err)
	}
}

func TestCheckConfig_MissingUsesDefaults(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	code, out, err := run(t, []string{"--config", p, "--check-config"}, "", fakeHost{usr: true, varp: true}, nil)
	if code != 0 {
		t.Fatalf("missing config must be valid defaults: %d %s %s", code, out, err)
	}
	if strings.Contains(err, "no such file") {
		t.Fatalf("must not require a live config.json: %s", err)
	}
	if !strings.Contains(strings.ToLower(out), "ok") {
		t.Fatalf("want configuration ok: %s", out)
	}
}

func TestCheckConfig_SampleOnlyDir(t *testing.T) {
	dir := t.TempDir()
	b, initErr := config.InitJSON()
	if initErr != nil {
		t.Fatal(initErr)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json.sample"), b, 0o644); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "config.json")
	code, out, stderr := run(t, []string{"--config", p, "--check-config"}, "", fakeHost{usr: true, varp: true}, nil)
	if code != 0 {
		t.Fatalf("sample-only dir: %d %s %s", code, out, stderr)
	}
	if strings.Contains(stderr, "no such file") {
		t.Fatalf("must not require copying the sample: %s", stderr)
	}
	if !strings.Contains(strings.ToLower(out), "ok") {
		t.Fatalf("want configuration ok: %s", out)
	}
}

func TestCheckConfig_AgreesWithDoctorOnMissing(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	ccode, cout, cerr := run(t, []string{"--config", p, "--check-config"}, "", fakeHost{usr: true, varp: true}, nil)
	if ccode != 0 {
		t.Fatalf("check-config: %d %s %s", ccode, cout, cerr)
	}
	_, dout, derr := run(t, []string{"--config", p, "--json", "doctor"}, "", fakeHost{usr: true, varp: true}, nil)
	_ = derr
	var rep struct {
		Checks []struct {
			Name string `json:"name"`
			OK   bool   `json:"ok"`
		} `json:"checks"`
	}
	if err := json.Unmarshal([]byte(dout), &rep); err != nil {
		t.Fatalf("doctor json: %v %s", err, dout)
	}
	found := false
	for _, c := range rep.Checks {
		if c.Name != "config" {
			continue
		}
		found = true
		if !c.OK {
			t.Fatal("doctor config check must be ok when the file is missing")
		}
	}
	if !found {
		t.Fatalf("doctor missing config check: %s", dout)
	}
}

func TestCheckConfig_FailsOnGarbage(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.json")
	if err := os.WriteFile(p, []byte("nope"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, _, err := run(t, []string{"--config", p, "--check-config"}, "", fakeHost{}, nil)
	if code == 0 {
		t.Fatal("expected failure")
	}
	if err == "" {
		t.Fatal("expected stderr")
	}
}

func TestPlan_RORootUnlockRW(t *testing.T) {
	code, out, err := run(t, []string{"plan", "pkg install foo"}, "", fakeHost{ro: true, rescue: true}, nil)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if !strings.Contains(out, "unlock-rw") {
		t.Fatal(out)
	}
}

func TestApply_DefaultDryRun(t *testing.T) {
	plan := `{"id":"p","source":"operator","steps":[{"id":"1","action":"echo","argv":["echo","hi"],"privileged":false}]}`
	code, out, err := run(t, []string{"apply"}, plan, fakeHost{usr: true, varp: true}, nil)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if !strings.Contains(out, `"dry_run": true`) && !strings.Contains(out, `"dry_run":true`) {
		t.Fatal(out)
	}
}

func TestApply_YesRunsEcho(t *testing.T) {
	plan := `{"id":"p","source":"operator","steps":[{"id":"1","action":"echo","argv":["echo","hawkeye-apply-yes"],"privileged":false}]}`
	dir := t.TempDir()
	audit := filepath.Join(dir, "audit.log")
	cfg, _ := config.InitJSON()
	var c config.Config
	_ = json.Unmarshal(cfg, &c)
	c.AuditLog = audit
	b, _ := json.Marshal(c)
	cp := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cp, b, 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, err := run(t, []string{"--config", cp, "apply", "--yes"}, plan, fakeHost{usr: true, varp: true}, map[string]string{"HAWKEYE_CONFIG": cp})
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if strings.Contains(out, `"applied": false`) {
		t.Fatal(out)
	}
}

func TestConsult_WithKnowledge(t *testing.T) {
	dir := t.TempDir()
	if err := knowledge.CreateTestDB(filepath.Join(dir, "knowledge.sqlite")); err != nil {
		t.Fatal(err)
	}
	code, out, err := run(t, []string{"consult", "zfs", "readonly"}, "", fakeHost{ro: true, rescue: true}, map[string]string{"HAWKEYE_KNOWLEDGE_PATH": dir})
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if !strings.Contains(out, "unlock-rw") {
		t.Fatal(out)
	}
}

func TestDoctor_JSONExit(t *testing.T) {
	dir := t.TempDir()
	if err := knowledge.CreateTestDB(filepath.Join(dir, "knowledge.sqlite")); err != nil {
		t.Fatal(err)
	}
	b, _ := config.InitJSON()
	cp := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cp, b, 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, err := run(t, []string{"--config", cp, "--json", "doctor"}, "", fakeHost{usr: true, varp: true, carrier: false}, map[string]string{"HAWKEYE_KNOWLEDGE_PATH": dir, "HAWKEYE_CONFIG": cp})
	_ = err
	if !strings.Contains(out, "healthy") {
		t.Fatalf("code=%d out=%s err=%s", code, out, err)
	}
}

func TestMCP_StdioInitialize(t *testing.T) {
	in := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}` + "\n"
	code, out, err := run(t, []string{"mcp", "--stdio"}, in, fakeHost{usr: true, varp: true}, nil)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if !strings.Contains(out, "hawkeye") {
		t.Fatal(out)
	}
}

func TestUpdate_RefusesRO(t *testing.T) {
	code, _, err := run(t, []string{"update", "--src", "x", "--dest", "y"}, "", fakeHost{ro: true}, nil)
	if code == 0 {
		t.Fatal("expected fail")
	}
	if err == "" {
		t.Fatal("stderr")
	}
}

func TestHelpAndUnknown(t *testing.T) {
	code, out, _ := run(t, []string{"--help"}, "", fakeHost{}, nil)
	if code != 0 || !strings.Contains(out, "consult") {
		t.Fatal(out)
	}
	code, _, err := run(t, []string{"nope"}, "", fakeHost{}, nil)
	if code == 0 || err == "" {
		t.Fatal("unknown")
	}
}

func TestVersion(t *testing.T) {
	code, out, _ := run(t, []string{"--version"}, "", fakeHost{}, nil)
	if code != 0 || !strings.Contains(out, "0.1.0") {
		t.Fatal(out)
	}
}
