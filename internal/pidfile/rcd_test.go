// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package pidfile_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func rcdScript(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "rc.d", "hawkeye"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestRcd_DaemonPidfileLeftReadable0644(t *testing.T) {
	src := rcdScript(t)
	if !strings.Contains(src, `command="/usr/sbin/daemon"`) {
		t.Fatal("rc.d must use daemon(8)")
	}
	if !strings.Contains(src, "-p ${pidfile}") {
		t.Fatal("rc.d must keep daemon -p so status matches the child")
	}
	if !strings.Contains(src, "hawkeye_pidfile_operator_readable") {
		t.Fatal("rc.d must have hawkeye_pidfile_operator_readable so daemon -p cannot leave 0600")
	}
	pre := extractFunc(src, "hawkeye_prestart")
	if !strings.Contains(pre, "hawkeye_pidfile_operator_readable") {
		t.Fatal("prestart must seed 0644 before daemon -p pidfile_open(0600)")
	}
	post := extractFunc(src, "hawkeye_poststart")
	if !strings.Contains(post, "hawkeye_pidfile_operator_readable") {
		t.Fatal("poststart must chmod 0644 after daemon -p")
	}
	helper := extractFunc(src, "hawkeye_pidfile_operator_readable")
	if !strings.Contains(helper, "chmod 0644") {
		t.Fatal("helper must chmod 0644; PID is not a secret")
	}
	if !strings.Contains(helper, "umask 022") && !strings.Contains(helper, ": >") && !strings.Contains(helper, "install -m 0644") {
		t.Fatal("helper must seed the pidfile before daemon creates 0600")
	}
}

func TestRcdHelper_FixesDaemon0600(t *testing.T) {
	helper := extractFunc(rcdScript(t), "hawkeye_pidfile_operator_readable")
	if helper == "" {
		t.Fatal("missing hawkeye_pidfile_operator_readable")
	}
	dir := t.TempDir()
	pid := filepath.Join(dir, "hawkeye.pid")
	if err := os.WriteFile(pid, []byte("4242\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runRcdHelper(t, helper, pid)
	assertMode(t, pid, 0o644)
}

func TestRcdHelper_SeedsMissingPidfile0644(t *testing.T) {
	helper := extractFunc(rcdScript(t), "hawkeye_pidfile_operator_readable")
	if helper == "" {
		t.Fatal("missing hawkeye_pidfile_operator_readable")
	}
	pid := filepath.Join(t.TempDir(), "hawkeye.pid")
	runRcdHelper(t, helper, pid)
	assertMode(t, pid, 0o644)
}

func extractFunc(src, name string) string {
	needle := name + "()"
	i := strings.Index(src, needle)
	if i < 0 {
		return ""
	}
	rest := src[i:]
	open := strings.Index(rest, "{")
	if open < 0 {
		return ""
	}
	depth := 0
	for j := open; j < len(rest); j++ {
		switch rest[j] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return rest[:j+1]
			}
		}
	}
	return ""
}

func runRcdHelper(t *testing.T, helper, pid string) {
	t.Helper()
	script := "warn() { :; }\n" + helper + "\nhawkeye_pidfile_operator_readable\n"
	cmd := exec.Command("sh", "-c", script)
	cmd.Env = append(os.Environ(), "pidfile="+pid)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("helper: %v\n%s", err, out)
	}
}

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fi.Mode().Perm(); got != want {
		t.Fatalf("%s mode %04o, want %04o", path, got, want)
	}
}
