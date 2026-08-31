// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package pidfile_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/revytechinc/hawkeye/internal/pidfile"
)

func TestWriteReadRemove(t *testing.T) {
	p := filepath.Join(t.TempDir(), "hawkeye.pid")
	if err := pidfile.Write(p, 4242); err != nil {
		t.Fatal(err)
	}
	n, err := pidfile.Read(p)
	if err != nil || n != 4242 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	if err := pidfile.Remove(p); err != nil {
		t.Fatal(err)
	}
}

func TestWrite_Mode0644(t *testing.T) {
	p := filepath.Join(t.TempDir(), "hawkeye.pid")
	if err := pidfile.Write(p, 4242); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if got := fi.Mode().Perm(); got != 0o644 {
		t.Fatalf("Write mode %04o, want 0644 so operator doctor can read", got)
	}
}

func TestOperatorReadable(t *testing.T) {
	if !pidfile.OperatorReadable(0o644) {
		t.Fatal("0644 must be operator-readable")
	}
	if pidfile.OperatorReadable(0o600) {
		t.Fatal("0600 is daemon(8) default; operator doctor cannot read it")
	}
	if pidfile.OperatorReadable(0o640) {
		t.Fatal("0640 is not world-readable; unprivileged doctor fails")
	}
}

func TestWrite_RejectsNonPositive(t *testing.T) {
	p := filepath.Join(t.TempDir(), "hawkeye.pid")
	if err := pidfile.Write(p, 0); err == nil {
		t.Fatal("expected error")
	}
	if err := pidfile.Write(p, -1); err == nil {
		t.Fatal("expected error")
	}
}

func TestRead_EmptyAndNegative(t *testing.T) {
	p := filepath.Join(t.TempDir(), "hawkeye.pid")
	if err := os.WriteFile(p, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := pidfile.Read(p); err == nil {
		t.Fatal("empty pidfile")
	}
	if err := os.WriteFile(p, []byte("-3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := pidfile.Read(p); err == nil {
		t.Fatal("negative pidfile")
	}
}
