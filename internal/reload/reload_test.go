// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package reload_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/revytechinc/hawkeye/internal/config"
	"github.com/revytechinc/hawkeye/internal/reload"
)

func TestReloadFile_KeepsOldOnBadConfig(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.json")
	bad := filepath.Join(dir, "bad.json")
	b, err := config.InitJSON()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(good, b, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bad, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(good)
	if err != nil {
		t.Fatal(err)
	}
	h := reload.New(cfg)
	if err := h.ReloadFile(bad); err == nil {
		t.Fatal("expected bad config to fail")
	}
	if h.Get().LogLevel != cfg.LogLevel {
		t.Fatal("bad reload mutated live config")
	}
	if err := h.ReloadFile(good); err != nil {
		t.Fatal(err)
	}
}
