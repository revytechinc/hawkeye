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
	for _, want := range []string{"install-rescue", "DESTDIR", "/rescue", "/boot/hawkeye"} {
		if !strings.Contains(mk, want) {
			t.Fatalf("Makefile must stage rescue layout (%s missing)", want)
		}
	}
	port, err := os.ReadFile(filepath.Join(root, "ports", "sysutils", "hawkeye", "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	pm := string(port)
	for _, want := range []string{"RESCUE", "/rescue/hawkeye", "/boot/hawkeye"} {
		if !strings.Contains(pm, want) {
			t.Fatalf("port Makefile must offer RESCUE layout (%s missing)", want)
		}
	}
	if strings.Contains(pm, "hawkeye-rescue") {
		t.Fatal("do not invent a hawkeye-rescue package name")
	}
}
