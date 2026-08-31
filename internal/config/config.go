// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

type Listen struct {
	MCPHTTP     string `json:"mcp_http"`
	TLSCert     string `json:"tls_cert"`
	TLSKey      string `json:"tls_key"`
	MCPTokenEnv string `json:"mcp_token_env"`
}

type Knowledge struct {
	Paths []string `json:"paths"`
}

type LocalLLM struct {
	Backend    string `json:"backend"`
	Bin        string `json:"bin"`
	ModelPath  string `json:"model_path"`
	PreferGPU  bool   `json:"prefer_gpu"`
	RequireGPU bool   `json:"require_gpu"`
}

type RemoteLLM struct {
	Provider  string `json:"provider"`
	Endpoint  string `json:"endpoint"`
	APIKeyEnv string `json:"api_key_env"`
}

type LLM struct {
	Local  LocalLLM  `json:"local"`
	Remote RemoteLLM `json:"remote"`
}

type Resources struct {
	RAMMinFreeBytes     *int64   `json:"ram_min_free_bytes"`
	CPUMaxPct           *float64 `json:"cpu_max_pct"`
	DiskMinFreeBytes    *int64   `json:"disk_min_free_bytes"`
	GPUVRAMMinFreeBytes *int64   `json:"gpu_vram_min_free_bytes"`
}

type Update struct {
	Source string `json:"source"`
	Dest   string `json:"dest"`
}

type Config struct {
	LogLevel  string    `json:"log_level"`
	Listen    Listen    `json:"listen"`
	Knowledge Knowledge `json:"knowledge"`
	LLM       LLM       `json:"llm"`
	Update    Update    `json:"update"`
	Resources Resources `json:"resources"`
	PidFile   string    `json:"pidfile"`
	AuditLog  string    `json:"audit_log"`
}

func Default() Config {
	ram := int64(256 * 1024 * 1024)
	cpu := 90.0
	disk := int64(100 * 1024 * 1024)
	return Config{
		LogLevel: "INFO",
		Listen: Listen{
			MCPHTTP:     "127.0.0.1:8741",
			MCPTokenEnv: "HAWKEYE_MCP_TOKEN",
		},
		Knowledge: Knowledge{
			Paths: []string{"/boot/hawkeye", "/usr/local/share/hawkeye"},
		},
		LLM: LLM{
			Local: LocalLLM{
				Backend:    "llama.cpp",
				Bin:        "",
				PreferGPU:  true,
				RequireGPU: false,
			},
			Remote: RemoteLLM{
				APIKeyEnv: "HAWKEYE_LLM_API_KEY",
			},
		},
		Update: Update{
			Dest: "/usr/local/share/hawkeye/knowledge.sqlite",
		},
		Resources: Resources{
			RAMMinFreeBytes:  &ram,
			CPUMaxPct:        &cpu,
			DiskMinFreeBytes: &disk,
		},
		PidFile:  "/var/run/hawkeye.pid",
		AuditLog: "/var/log/hawkeye/audit.log",
	}
}

func Validate(c Config) error {
	switch strings.ToUpper(c.LogLevel) {
	case "DEBUG", "INFO", "WARN", "ERROR":
	default:
		return fmt.Errorf("log_level %q is invalid; use DEBUG, INFO, WARN, or ERROR", c.LogLevel)
	}
	if strings.TrimSpace(c.Listen.MCPHTTP) == "" {
		return fmt.Errorf("listen.mcp_http is required")
	}
	host, _, err := net.SplitHostPort(c.Listen.MCPHTTP)
	if err != nil {
		return fmt.Errorf("listen.mcp_http: %w", err)
	}
	if host != "localhost" {
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			return fmt.Errorf("listen.mcp_http must bind loopback (127.0.0.1 or ::1), got %q", c.Listen.MCPHTTP)
		}
	}
	envName := strings.TrimSpace(c.Listen.MCPTokenEnv)
	if envName == "" {
		return fmt.Errorf("listen.mcp_token_env is required (environment variable name, not the secret)")
	}
	if len(envName) > 64 {
		return fmt.Errorf("listen.mcp_token_env must be an environment variable name")
	}
	for i, r := range envName {
		ok := (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
		if i == 0 && !(r >= 'A' && r <= 'Z') {
			ok = false
		}
		if !ok {
			return fmt.Errorf("listen.mcp_token_env must be an environment variable name, not a secret")
		}
	}
	if strings.TrimSpace(c.PidFile) == "" {
		return fmt.Errorf("pidfile is required")
	}
	if c.LLM.Local.RequireGPU && !c.LLM.Local.PreferGPU {
		return fmt.Errorf("llm.local.require_gpu is true but prefer_gpu is false")
	}
	return nil
}

func Parse(b []byte) (Config, error) {
	if !json.Valid(b) {
		return Config{}, fmt.Errorf("not valid JSON (RFC 8259)")
	}
	c := Default()
	dec := json.NewDecoder(bytes.NewReader(b))
	if err := dec.Decode(&c); err != nil {
		return Config{}, err
	}
	if err := Validate(c); err != nil {
		return Config{}, err
	}
	return c, nil
}

func CheckFile(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Native install ships config.json.sample only. Missing live
			// config is valid: compiled defaults, same as doctor.Load.
			return Validate(Default())
		}
		return err
	}
	_, err = Parse(b)
	return err
}

func InitJSON() ([]byte, error) {
	b, err := json.MarshalIndent(Default(), "", "  ")
	if err != nil {
		return nil, err
	}
	b = append(b, '\n')
	return b, nil
}

var systemDir = "/usr/local/etc/cloudbsd/hawkeye"

func SystemDir() string { return systemDir }

func ExamplePath() string { return filepath.Join(SystemDir(), "config.json.sample") }

func UserDir() string {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "hawkeye")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("/tmp", "hawkeye")
	}
	return filepath.Join(home, ".config", "hawkeye")
}

func ResolvePath(explicit string) string {
	if explicit != "" {
		return explicit
	}
	candidates := []string{
		filepath.Join(UserDir(), "config.json"),
		filepath.Join(SystemDir(), "config.json"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return filepath.Join(SystemDir(), "config.json")
}

// ApplyEnv overlays operator environment. Model path and backend binary
// may live in JSON or env (HAWKEYE_LLM_MODEL, HAWKEYE_LLM_BIN). Tokens,
// if a remote key is ever set, stay in env only.
func ApplyEnv(c Config, getenv func(string) string) Config {
	if getenv == nil {
		return c
	}
	if v := strings.TrimSpace(getenv("HAWKEYE_LLM_MODEL")); v != "" {
		c.LLM.Local.ModelPath = v
	}
	if v := strings.TrimSpace(getenv("HAWKEYE_LLM_BIN")); v != "" {
		c.LLM.Local.Bin = v
	}
	if v := strings.TrimSpace(getenv("HAWKEYE_UPDATE_SOURCE")); v != "" {
		c.Update.Source = v
	} else if strings.TrimSpace(c.Update.Source) == "" {
		if v := strings.TrimSpace(getenv("HAWKEYE_DATA_ARTIFACT")); v != "" {
			c.Update.Source = v
		}
	}
	return c
}

func Load(path string) (Config, error) {
	if path == "" {
		return Default(), Validate(Default())
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return Config{}, err
	}
	return Parse(b)
}
