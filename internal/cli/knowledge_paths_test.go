// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package cli

import (
	"strings"
	"testing"

	"github.com/revytechinc/hawkeye/internal/config"
)

func TestKnowledgePaths_OverrideIsExclusive(t *testing.T) {
	env := Env{Getenv: func(k string) string {
		if k == "HAWKEYE_KNOWLEDGE_PATH" {
			return "/tmp/kit-fixture"
		}
		return ""
	}}
	got := knowledgePaths(env, config.Default())
	if len(got) != 1 || got[0] != "/tmp/kit-fixture" {
		t.Fatalf("HAWKEYE_KNOWLEDGE_PATH must isolate from live kit paths: %q", got)
	}
	for _, p := range got {
		if strings.Contains(p, "/usr/local/share/hawkeye") || strings.Contains(p, "/boot/hawkeye") {
			t.Fatalf("live kit path leaked: %q", got)
		}
	}
}

func TestKnowledgePaths_DefaultIncludesCompiled(t *testing.T) {
	env := Env{Getenv: func(string) string { return "" }}
	got := knowledgePaths(env, config.Default())
	found := false
	for _, p := range got {
		if p == "/usr/local/share/hawkeye" {
			found = true
		}
	}
	if !found {
		t.Fatalf("unset override must still search compiled paths: %q", got)
	}
}
