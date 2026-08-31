// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package apply

import "testing"

func TestSysExecutor_stdinClosedWrite(t *testing.T) {
	s := &SysExecutor{}
	t.Cleanup(func() { _ = s.Close() })
	s.mu.Lock()
	if err := s.ensureLocked(); err != nil {
		s.mu.Unlock()
		t.Fatal(err)
	}
	_ = s.stdin.Close()
	s.mu.Unlock()
	if _, _, err := s.Run([]string{"echo x && true"}); err == nil {
		t.Fatal("write to closed stdin must fail")
	}
}

func TestErrSince(t *testing.T) {
	s := &SysExecutor{}
	if s.errSince(0) != "" {
		t.Fatal("empty")
	}
	s.errBuf.WriteString("note")
	if s.errSince(0) != "note" {
		t.Fatal(s.errSince(0))
	}
	if s.errSince(4) != "" {
		t.Fatal("past end")
	}
}
