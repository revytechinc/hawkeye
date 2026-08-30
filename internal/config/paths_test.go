// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/revytechinc/hawkeye/internal/config"
)

func TestUserDir_XDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/xdg/config")
	if got := config.UserDir(); got != "/xdg/config/hawkeye" {
		t.Fatal(got)
	}
}

func TestResolvePath_ExplicitWins(t *testing.T) {
	if got := config.ResolvePath("/tmp/x.json"); got != "/tmp/x.json" {
		t.Fatal(got)
	}
}

func TestLoad_MissingReturnsDefault(t *testing.T) {
	c, err := config.Load(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatal(err)
	}
	if c.LogLevel != "INFO" {
		t.Fatal(c.LogLevel)
	}
}

func TestSystemDir(t *testing.T) {
	if config.SystemDir() != "/usr/local/etc/cloudbsd/hawkeye" {
		t.Fatal(config.SystemDir())
	}
	_ = config.ExamplePath()
	_ = os.ErrNotExist
}
