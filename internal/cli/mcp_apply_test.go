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
	"github.com/revytechinc/hawkeye/internal/config"
)

const mcpExecToken = "hawkeye-mcp-real-exec"

func mcpStdioApply(t *testing.T, yes bool, privileged bool, extraEnv map[string]string) (int, string, string) {
	t.Helper()
	plan := apply.Plan{
		ID:     "mcp-apply",
		Source: "operator",
		Steps: []apply.Step{{
			ID:         "1",
			Action:     "echo",
			Argv:       []string{"echo", mcpExecToken},
			Privileged: privileged,
		}},
	}
	args, err := json.Marshal(map[string]any{
		"name": "apply",
		"arguments": map[string]any{
			"yes":  yes,
			"plan": plan,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	req, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params":  json.RawMessage(args),
	})
	if err != nil {
		t.Fatal(err)
	}
	return run(t, []string{"mcp", "--stdio"}, string(req)+"\n", fakeHost{usr: true, varp: true}, extraEnv)
}

func decodeMCPApplyResult(t *testing.T, out string) apply.Result {
	t.Helper()
	var resp struct {
		Error  any            `json:"error"`
		Result map[string]any `json:"result"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &resp); err != nil {
		t.Fatalf("mcp json: %v %s", err, out)
	}
	if resp.Error != nil {
		t.Fatalf("mcp error: %v %s", resp.Error, out)
	}
	raw, err := json.Marshal(resp.Result)
	if err != nil {
		t.Fatal(err)
	}
	var res apply.Result
	if err := json.Unmarshal(raw, &res); err != nil {
		t.Fatalf("result: %v %s", err, raw)
	}
	return res
}

func TestMCP_UnprivilegedYesUsesRealExecutor(t *testing.T) {
	dir := t.TempDir()
	auditPath := filepath.Join(dir, "audit.log")
	cfgPath := writeAuditCfg(t, dir, auditPath)
	code, out, errb := mcpStdioApply(t, true, false, map[string]string{"HAWKEYE_CONFIG": cfgPath})
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, errb)
	}
	res := decodeMCPApplyResult(t, out)
	if res.DryRun || !res.Applied {
		t.Fatalf("unprivileged + yes must apply: %+v %s", res, out)
	}
	joined := out
	for _, s := range res.Steps {
		joined += s.Output
	}
	if !strings.Contains(joined, mcpExecToken) {
		t.Fatalf("Apply claimed success without a real executor (CountingExecutor returns ok, no process): %s", out)
	}
	if strings.TrimSpace(stepOutputs(res)) == "ok" && !strings.Contains(stepOutputs(res), mcpExecToken) {
		t.Fatal("CountingExecutor lie")
	}
	b, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"apply"`) {
		t.Fatalf("MCP apply must use the CLI auditor: %s", b)
	}
}

func TestMCP_UnprivilegedDefaultIsDryRun(t *testing.T) {
	code, out, errb := mcpStdioApply(t, false, false, nil)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, errb)
	}
	res := decodeMCPApplyResult(t, out)
	if !res.DryRun || res.Applied {
		t.Fatalf("default MCP apply must stay dry-run: %+v", res)
	}
	if strings.Contains(stepOutputs(res), mcpExecToken) && !strings.Contains(stepOutputs(res), "dry-run") {
		t.Fatalf("default must not exec: %s", out)
	}
}

func TestMCP_PrivilegedYesStaysDryRun(t *testing.T) {
	ex := &apply.CountingExecutor{}
	dir := t.TempDir()
	auditPath := filepath.Join(dir, "audit.log")
	cfgPath := writeAuditCfg(t, dir, auditPath)
	var out, errb strings.Builder
	plan := apply.Plan{
		ID:     "mcp-priv",
		Source: "knowledge",
		Steps: []apply.Step{{
			ID:         "1",
			Action:     "echo",
			Argv:       []string{"echo", mcpExecToken},
			Privileged: true,
		}},
	}
	args, _ := json.Marshal(map[string]any{
		"name":      "apply",
		"arguments": map[string]any{"yes": true, "plan": plan},
	})
	req, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": json.RawMessage(args),
	})
	code := cli.RunEnv(cli.Env{
		Args:   []string{"hawkeye", "--config", cfgPath, "mcp", "--stdio"},
		Stdin:  strings.NewReader(string(req) + "\n"),
		Stdout: &out,
		Stderr: &errb,
		Getenv: func(k string) string {
			if k == "HAWKEYE_CONFIG" {
				return cfgPath
			}
			return ""
		},
		Host: fakeHost{usr: true, varp: true},
		Exec: ex,
	})
	if code != 0 {
		t.Fatalf("%d %s %s", code, out.String(), errb.String())
	}
	res := decodeMCPApplyResult(t, out.String())
	if !res.DryRun || res.Applied {
		t.Fatalf("privileged MCP apply stays dry-run unless the operator CLI --yes path: %+v", res)
	}
	if ex.Calls != 0 {
		t.Fatalf("privileged MCP must not exec: %d", ex.Calls)
	}
}

func stepOutputs(res apply.Result) string {
	var b strings.Builder
	for _, s := range res.Steps {
		b.WriteString(s.Output)
	}
	return b.String()
}

func writeAuditCfg(t *testing.T, dir, auditPath string) string {
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
