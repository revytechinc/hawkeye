// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdate_NoSourceSkipsHealthy(t *testing.T) {
	code, out, err := run(t, []string{"update"}, "", fakeHost{usr: true, varp: true}, nil)
	if code != 0 {
		t.Fatalf("unset source must be a healthy skip for rc start: %d %s %s", code, out, err)
	}
	if strings.Contains(err, "source and destination are required") {
		t.Fatalf("rc start must not log missing src/dest: %s", err)
	}
	if strings.Contains(out, "/usr/local/share/hawkeye") {
		t.Fatalf("skip must not print a dest as if written: %s", out)
	}
}

func TestUpdate_SourceFromEnv(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.sqlite")
	dst := filepath.Join(dir, "dest", "knowledge.sqlite")
	if err := os.WriteFile(src, []byte("kit"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, err := run(t, []string{"update", "--dest", dst}, "", fakeHost{usr: true, varp: true}, map[string]string{
		"HAWKEYE_UPDATE_SOURCE": src,
	})
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	b, readErr := os.ReadFile(dst)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(b) != "kit" {
		t.Fatalf("got %q", b)
	}
}
