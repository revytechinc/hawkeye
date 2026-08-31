// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package cli_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/revytechinc/hawkeye/internal/apply"
	"github.com/revytechinc/hawkeye/internal/cli"
	"github.com/revytechinc/hawkeye/internal/config"
)

// Compact plan JSON with a password: kv value. Redacting the raw document
// eats from password: through the rest of the line (JSON punctuation is \S).
const fakePasswordPlan = `{"id":"p","source":"operator","summary":"rotate password: fake-password-for-tests-only","steps":[{"id":"1","action":"echo","argv":["echo","password:fake-password-for-tests-only"],"privileged":false}]}`

func TestApply_PasswordKVStillUnmarshals(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "plan.json")
	if err := os.WriteFile(p, []byte(fakePasswordPlan), 0o600); err != nil {
		t.Fatal(err)
	}
	cfgPath, _ := writePanicCfg(t, dir)
	code, out, errb := run(t, []string{"--config", cfgPath, "apply", "--dry-run", p}, "", fakeHost{usr: true, varp: true}, map[string]string{"HAWKEYE_CONFIG": cfgPath})
	if code != 0 {
		t.Fatalf("file apply must parse then redact by field, not redact raw JSON: %d %s %s", code, out, errb)
	}
	if strings.Contains(out, "fake-password-for-tests-only") || strings.Contains(errb, "fake-password-for-tests-only") {
		t.Fatalf("secret leaked: %s %s", out, errb)
	}
	if !strings.Contains(out, `"dry_run": true`) && !strings.Contains(out, `"dry_run":true`) {
		t.Fatalf("want dry-run result: %s", out)
	}
	var res apply.Result
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("result JSON: %v %s", err, out)
	}
	if res.Applied {
		t.Fatal("dry-run must not apply")
	}
}

func TestApply_StepFailureExitsNonZero(t *testing.T) {
	dir := t.TempDir()
	cfgPath, _ := writePanicCfg(t, dir)
	plan := `{"id":"p","source":"operator","steps":[{"id":"1","action":"false","argv":["false"],"privileged":false}]}`
	code, out, errb := run(t, []string{"--config", cfgPath, "apply", "--yes"}, plan, fakeHost{usr: true, varp: true}, map[string]string{"HAWKEYE_CONFIG": cfgPath})
	if code == 0 {
		t.Fatalf("failed land must exit non-zero: %s %s", out, errb)
	}
	if strings.Contains(out, `"applied": true`) || strings.Contains(out, `"applied":true`) {
		t.Fatalf("TTY/JSON must not claim applied: %s", out)
	}
	var res apply.Result
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("still emit result JSON: %v %s", err, out)
	}
	if res.Applied {
		t.Fatalf("%+v", res)
	}
}

func TestApply_ROMissingVarStillLands(t *testing.T) {
	ro := t.TempDir()
	if err := os.Chmod(ro, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(ro, 0o755) })
	auditLog := filepath.Join(ro, "var", "log", "hawkeye", "audit.log")

	dir := t.TempDir()
	cfgPath := writeAuditPathCfg(t, dir, auditLog)
	plan := `{"id":"p","source":"operator","steps":[{"id":"1","action":"echo","argv":["echo","hawkeye-ro-audit-land"],"privileged":false}]}`
	code, out, errb := run(t, []string{"--config", cfgPath, "apply", "--yes"}, plan, fakeHost{ro: true, rescue: true}, map[string]string{"HAWKEYE_CONFIG": cfgPath})
	if code != 0 {
		t.Fatalf("ModeApply on RO missing /var must still land: %d %s %s", code, out, errb)
	}
	if !strings.Contains(out, "hawkeye-ro-audit-land") {
		t.Fatalf("must exec: %s", out)
	}
	if strings.Contains(out, `"applied": false`) {
		t.Fatalf("successful land: %s", out)
	}
	if !strings.Contains(errb, "audit") {
		t.Fatalf("want stderr note that audit degraded: %s", errb)
	}
}

func writePanicCfg(t *testing.T, dir string) (cfgPath, auditPath string) {
	t.Helper()
	return writeAuditPathCfg(t, dir, filepath.Join(dir, "audit.log")), filepath.Join(dir, "audit.log")
}

func writeAuditPathCfg(t *testing.T, dir, auditPath string) string {
	t.Helper()
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
	cfgPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cfgPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return cfgPath
}

func TestConsult_TTY_FailedLandDoesNotClaimApplied(t *testing.T) {
	kd := playbookDir(t)
	cfgPath, _ := writePanicCfg(t, t.TempDir())
	var out, errb bytes.Buffer
	code := cli.RunEnv(cli.Env{
		Args:   []string{"hawkeye", "--config", cfgPath, "consult", "ZFS", "root", "is", "read-only", "after", "boot"},
		Stdin:  strings.NewReader("y\ny\n"),
		Stdout: &out,
		Stderr: &errb,
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
		Exec: failExecCLI{},
	})
	if code == 0 {
		t.Fatalf("failed TTY land must be non-zero: %s %s", out.String(), errb.String())
	}
	got := out.String()
	if strings.Contains(got, "applied") && !strings.Contains(got, "nothing applied") {
		t.Fatalf("TTY must not claim applied: %s", got)
	}
}

type failExecCLI struct{}

func (failExecCLI) Run([]string) (string, string, error) {
	return "", "boom", errors.New("boom")
}
