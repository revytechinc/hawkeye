// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/revytechinc/hawkeye/internal/config"
)

func TestDefault_IsLoopbackAndJSON(t *testing.T) {
	c := config.Default()
	if !strings.HasPrefix(c.Listen.MCPHTTP, "127.0.0.1:") && !strings.HasPrefix(c.Listen.MCPHTTP, "[::1]:") {
		t.Fatalf("mcp listen must default to loopback, got %q", c.Listen.MCPHTTP)
	}
	if c.LLM.Local.RequireGPU {
		t.Fatal("missing GPU must not be required")
	}
	if c.LLM.Local.Backend != "llama.cpp" {
		t.Fatalf("backend %q", c.LLM.Local.Backend)
	}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(b) {
		t.Fatal("default config is not RFC 8259 JSON")
	}
}

func TestCheckFile_ValidSample(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	b, err := config.InitJSON()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, b, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := config.CheckFile(p); err != nil {
		t.Fatal(err)
	}
}

func TestCheckFile_MissingUsesDefaults(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatal("expected missing live config")
	}
	if err := config.CheckFile(p); err != nil {
		t.Fatalf("missing config must use compiled defaults (same as doctor): %v", err)
	}
}

func TestCheckFile_SampleOnlyDirUsesDefaults(t *testing.T) {
	dir := t.TempDir()
	b, err := config.InitJSON()
	if err != nil {
		t.Fatal(err)
	}
	sample := filepath.Join(dir, "config.json.sample")
	if err := os.WriteFile(sample, b, 0o644); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(string(b)), "sk-") || strings.Contains(string(b), "BEGIN ") {
		t.Fatal("sample fixture must not contain secrets")
	}
	live := filepath.Join(dir, "config.json")
	if _, err := os.Stat(live); !os.IsNotExist(err) {
		t.Fatal("live config.json must be absent in a sample-only dir")
	}
	if err := config.CheckFile(live); err != nil {
		t.Fatalf("sample-only dir must use defaults; operators must not be required to copy the sample: %v", err)
	}
}

func TestCheckFile_RejectsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(p, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := config.CheckFile(p); err == nil {
		t.Fatal("expected invalid JSON to fail --check-config")
	}
}

func TestCheckFile_RejectsComments(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "c.json")
	if err := os.WriteFile(p, []byte("{\n  // nope\n  \"log_level\": \"INFO\"\n}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := config.CheckFile(p); err == nil {
		t.Fatal("JSONC must be rejected")
	}
}

func TestValidate_RejectsPublicBind(t *testing.T) {
	c := config.Default()
	c.Listen.MCPHTTP = "0.0.0.0:8741"
	if err := config.Validate(c); err == nil {
		t.Fatal("public bind must be rejected by default validator")
	}
}

func TestValidate_RejectsUnknownLogLevel(t *testing.T) {
	c := config.Default()
	c.LogLevel = "LOUD"
	if err := config.Validate(c); err == nil {
		t.Fatal("expected log_level error")
	}
}

func TestParse_SecretsStayInEnvNameOnly(t *testing.T) {
	c := config.Default()
	if c.LLM.Remote.APIKeyEnv == "" {
		t.Fatal("api_key_env must name an environment variable")
	}
	b, _ := json.Marshal(c)
	if strings.Contains(strings.ToLower(string(b)), "sk-") {
		t.Fatal("config JSON must not contain secret material")
	}
}
