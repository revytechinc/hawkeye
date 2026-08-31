// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package knowledge_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/revytechinc/hawkeye/internal/knowledge"
)

func TestRescueLayoutConstants(t *testing.T) {
	if knowledge.RescueDir != "/boot/hawkeye" {
		t.Fatal(knowledge.RescueDir)
	}
	if knowledge.SystemDir != "/usr/local/share/hawkeye" {
		t.Fatal(knowledge.SystemDir)
	}
	if knowledge.RescueBinary != "/rescue/hawkeye" {
		t.Fatal(knowledge.RescueBinary)
	}
	if knowledge.DBName != "knowledge.sqlite" {
		t.Fatal(knowledge.DBName)
	}
}

func TestMakefileInstallRescue(t *testing.T) {
	root := filepath.Join("..", "..")
	b, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	mk := string(b)
	for _, want := range []string{"install-rescue", "DESTDIR", "/rescue", "/boot/hawkeye", "! -L", "EROFS", "EACCES", "EPERM", "read-only"} {
		if !strings.Contains(mk, want) {
			t.Fatalf("Makefile must DESTDIR-stage, skip dangling /rescue, and skip RO /boot (%s missing)", want)
		}
	}
	if !strings.Contains(mk, "skip $(BOOT_HAWKEYE) (read-only)") {
		t.Fatal("RO /boot skip message must match skip /rescue style")
	}
	port, err := os.ReadFile(filepath.Join(root, "ports", "sysutils", "hawkeye", "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	pm := string(port)
	for _, want := range []string{"RESCUE", "/rescue/hawkeye", "/boot/hawkeye", "STAGEDIR"} {
		if !strings.Contains(pm, want) {
			t.Fatalf("port Makefile must offer DESTDIR-safe RESCUE layout (%s missing)", want)
		}
	}
	if strings.Contains(pm, "hawkeye-rescue") {
		t.Fatal("do not invent a hawkeye-rescue package name")
	}
	for _, line := range strings.Split(pm, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "#") {
			continue
		}
		if strings.Contains(trim, "RUN_DEPENDS") && strings.Contains(trim, "llama") {
			t.Fatalf("do not add a llama.cpp port as RUN_DEPENDS: %s", line)
		}
	}
	if strings.Contains(pm, "OPTIONS_DEFAULT") && strings.Contains(pm, "RESCUE") {
		if strings.Contains(pm, "OPTIONS_DEFAULT=\tRESCUE") || strings.Contains(pm, "OPTIONS_DEFAULT=	RESCUE") || strings.Contains(pm, "OPTIONS_DEFAULT= RESCUE") {
			t.Fatal("RESCUE must stay off by default (thin jail /rescue is a dangling symlink)")
		}
	}
}
