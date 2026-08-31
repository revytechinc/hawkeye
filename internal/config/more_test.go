// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/revytechinc/hawkeye/internal/config"
)

func TestValidate_MoreCases(t *testing.T) {
	c := config.Default()
	c.Listen.MCPHTTP = ""
	if err := config.Validate(c); err == nil {
		t.Fatal("empty listen")
	}
	c = config.Default()
	c.Listen.MCPHTTP = "not-a-port"
	if err := config.Validate(c); err == nil {
		t.Fatal("bad hostport")
	}
	c = config.Default()
	c.PidFile = ""
	if err := config.Validate(c); err == nil {
		t.Fatal("empty pidfile")
	}
	c = config.Default()
	c.LLM.Local.RequireGPU = true
	c.LLM.Local.PreferGPU = false
	if err := config.Validate(c); err == nil {
		t.Fatal("require without prefer")
	}
	c = config.Default()
	c.Listen.MCPHTTP = "localhost:9"
	if err := config.Validate(c); err != nil {
		t.Fatal(err)
	}
}

func TestLoad_GoodFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "c.json")
	b, err := config.InitJSON()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, b, 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if c.LogLevel != "INFO" {
		t.Fatal(c.LogLevel)
	}
}

func TestResolvePath_UserConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	p := filepath.Join(dir, "hawkeye", "config.json")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	b, _ := config.InitJSON()
	if err := os.WriteFile(p, b, 0o600); err != nil {
		t.Fatal(err)
	}
	got := config.ResolvePath("")
	if got != p {
		t.Fatalf("got %s want %s", got, p)
	}
}

func TestValidate_MCPTokenEnv(t *testing.T) {
	c := config.Default()
	if c.Listen.MCPTokenEnv != "HAWKEYE_MCP_TOKEN" {
		t.Fatal(c.Listen.MCPTokenEnv)
	}
	c.Listen.MCPTokenEnv = ""
	if err := config.Validate(c); err == nil {
		t.Fatal("empty token env")
	}
	c = config.Default()
	c.Listen.MCPTokenEnv = "sk-this-looks-like-a-secret"
	if err := config.Validate(c); err == nil {
		t.Fatal("secret-shaped env name")
	}
	c = config.Default()
	c.Listen.MCPTokenEnv = "HAWKEYE_MCP_TOKEN"
	if err := config.Validate(c); err != nil {
		t.Fatal(err)
	}
}

func TestApplyEnv_LLMAndUpdate(t *testing.T) {
	c := config.Default()
	if c.LLM.Local.Bin != "" {
		t.Fatal("bin must default empty (configured or PATH llama-cli)")
	}
	if c.Update.Dest != "/usr/local/share/hawkeye/knowledge.sqlite" {
		t.Fatalf("update dest default = %q", c.Update.Dest)
	}
	if c.Update.Source != "" {
		t.Fatal("update source must default empty so rc start skips")
	}
	got := config.ApplyEnv(c, func(k string) string {
		switch k {
		case "HAWKEYE_LLM_MODEL":
			return "/models/fake.gguf"
		case "HAWKEYE_LLM_BIN":
			return "/usr/local/bin/llama-cli"
		case "HAWKEYE_UPDATE_SOURCE":
			return "/var/cache/hawkeye-data/knowledge.sqlite"
		default:
			return ""
		}
	})
	if got.LLM.Local.ModelPath != "/models/fake.gguf" {
		t.Fatal(got.LLM.Local.ModelPath)
	}
	if got.LLM.Local.Bin != "/usr/local/bin/llama-cli" {
		t.Fatal(got.LLM.Local.Bin)
	}
	if got.Update.Source != "/var/cache/hawkeye-data/knowledge.sqlite" {
		t.Fatal(got.Update.Source)
	}
	legacy := config.ApplyEnv(config.Default(), func(k string) string {
		if k == "HAWKEYE_DATA_ARTIFACT" {
			return "/legacy/knowledge.sqlite"
		}
		return ""
	})
	if legacy.Update.Source != "/legacy/knowledge.sqlite" {
		t.Fatalf("legacy artifact env: %q", legacy.Update.Source)
	}
}
