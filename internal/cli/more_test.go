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
)

func TestApplyEmptyAndBadJSON(t *testing.T) {
	code, _, err := run(t, []string{"apply"}, "   ", fakeHost{}, nil)
	if code == 0 || err == "" {
		t.Fatal("empty")
	}
	code, _, err = run(t, []string{"apply"}, "{", fakeHost{}, nil)
	if code == 0 {
		t.Fatal("bad json")
	}
}

func TestApplyFromFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "plan.json")
	if err := os.WriteFile(p, []byte(`{"id":"p","source":"operator","steps":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, err := run(t, []string{"apply", p}, "", fakeHost{usr: true, varp: true}, nil)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if !strings.Contains(out, "dry_run") {
		t.Fatal(out)
	}
}

func TestInitStdoutAndVersionCmd(t *testing.T) {
	code, out, err := run(t, []string{"init", "-"}, "", fakeHost{}, nil)
	if code != 0 {
		t.Fatalf("%s %s", out, err)
	}
	if !strings.Contains(out, "log_level") {
		t.Fatal(out)
	}
	code, out, _ = run(t, []string{"version"}, "", fakeHost{}, nil)
	if code != 0 || !strings.Contains(out, "0.1.0") {
		t.Fatal(out)
	}
}

func TestMCPHTTPRequiresToken(t *testing.T) {
	dir := t.TempDir()
	b, _ := config.InitJSON()
	cp := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cp, b, 0o600); err != nil {
		t.Fatal(err)
	}
	code, _, err := run(t, []string{"--config", cp, "mcp", "--http"}, "", fakeHost{usr: true, varp: true}, nil)
	if code == 0 {
		t.Fatal("expected token required")
	}
	if !strings.Contains(strings.ToLower(err), "token") {
		t.Fatal(err)
	}
	if strings.Contains(err, "test-mcp-token-fixture-not-production") {
		t.Fatal("stderr leaked fixture token")
	}
}

func TestUpdateCopies(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.sqlite")
	dst := filepath.Join(dir, "out", "knowledge.sqlite")
	if err := os.WriteFile(src, []byte("abc"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, err := run(t, []string{"update", "--src", src, "--dest", dst}, "", fakeHost{usr: true, varp: true}, nil)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
}

func TestConsultStdinAndPlanStdin(t *testing.T) {
	code, out, err := run(t, []string{"consult"}, "zfs status", fakeHost{ro: true, rescue: true}, nil)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	code, out, err = run(t, []string{"plan"}, "hello", fakeHost{usr: true, varp: true}, nil)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
}

func TestDoctorHuman(t *testing.T) {
	code, out, _ := run(t, []string{"doctor"}, "", fakeHost{usr: true, varp: true}, nil)
	if !strings.Contains(strings.ToLower(out), "doctor") {
		t.Fatalf("code=%d out=%s", code, out)
	}
}

func TestHelpNoArgs(t *testing.T) {
	code, out, _ := run(t, []string{}, "", fakeHost{}, nil)
	if code != 0 || !strings.Contains(out, "Usage") {
		t.Fatal(out)
	}
}

func TestRunWrapper(t *testing.T) {
	var out, errb bytes.Buffer
	code := cli.Run([]string{"hawkeye", "--version"}, bytes.NewReader(nil), &out, &errb)
	if code != 0 {
		t.Fatal(code, errb.String())
	}
}

func TestDoctorUnreadablePidfile(t *testing.T) {
	dir := t.TempDir()
	pidp := filepath.Join(dir, "hawkeye.pid")
	if err := os.WriteFile(pidp, []byte("12345\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(pidp, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(pidp, 0o600) })
	cfgp := filepath.Join(dir, "config.json")
	b, err := config.InitJSON()
	if err != nil {
		t.Fatal(err)
	}
	// Point pidfile at the unreadable file.
	c, err := config.Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	c.PidFile = pidp
	raw, err := jsonIndent(c)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgp, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, _ := run(t, []string{"--config", cfgp, "doctor", "--json"}, "", fakeHost{usr: true, varp: true}, nil)
	if code == 0 {
		t.Fatal("expected unhealthy")
	}
	if strings.Contains(out, "pidfile is empty") {
		t.Fatalf("unreadable reported as empty: %s", out)
	}
	if !strings.Contains(out, "unreadable") {
		t.Fatalf("want unreadable: %s", out)
	}
}

func jsonIndent(c config.Config) ([]byte, error) {
	return json.MarshalIndent(c, "", "  ")
}

