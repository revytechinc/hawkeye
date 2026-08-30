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
