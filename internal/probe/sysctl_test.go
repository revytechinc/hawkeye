// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package probe_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/revytechinc/hawkeye/internal/probe"
)

func TestParseSysctlN(t *testing.T) {
	v, ok := probe.ParseSysctlN("0\n")
	if !ok || v != 0 {
		t.Fatalf("0: %d %v", v, ok)
	}
	v, ok = probe.ParseSysctlN("  -1 \n")
	if !ok || v != -1 {
		t.Fatalf("-1: %d %v", v, ok)
	}
	v, ok = probe.ParseSysctlN("2")
	if !ok || v != 2 {
		t.Fatalf("2: %d %v", v, ok)
	}
	if _, ok := probe.ParseSysctlN(""); ok {
		t.Fatal("empty")
	}
	if _, ok := probe.ParseSysctlN("kern.securelevel: 0"); ok {
		t.Fatal("verbose sysctl without -n is not -n output")
	}
	if _, ok := probe.ParseSysctlN("not-an-int"); ok {
		t.Fatal("garbage")
	}
}

func TestSignedSysctl32_SecurelevelMinusOne(t *testing.T) {
	if got := probe.SignedSysctl32(4294967295); got != -1 {
		t.Fatalf("uint32 wrap of -1: got %d", got)
	}
	if got := probe.SignedSysctl32(0); got != 0 {
		t.Fatalf("0: %d", got)
	}
	if got := probe.SignedSysctl32(1); got != 1 {
		t.Fatalf("1: %d", got)
	}
	if got := probe.SignedSysctl32(2); got != 2 {
		t.Fatalf("2: %d", got)
	}
}

func TestSysctl8Int_UsesInjectedRunner(t *testing.T) {
	var got []string
	v, ok := probe.Sysctl8Int("kern.securelevel", func(argv []string) (string, error) {
		got = append([]string(nil), argv...)
		return "1\n", nil
	})
	if !ok || v != 1 {
		t.Fatalf("got %d %v", v, ok)
	}
	if len(got) < 3 || got[1] != "-n" || got[2] != "kern.securelevel" {
		t.Fatalf("sysctl(8) argv: %v", got)
	}
	if !strings.Contains(got[0], "sysctl") {
		t.Fatalf("must exec sysctl(8): %v", got)
	}
}

func TestSysctl8Int_RejectsUnsafeMIB(t *testing.T) {
	called := false
	_, ok := probe.Sysctl8Int("kern.securelevel;reboot", func([]string) (string, error) {
		called = true
		return "0", nil
	})
	if ok || called {
		t.Fatal("unsafe MIB must not exec")
	}
}

func TestSysctl8Int_RunnerError(t *testing.T) {
	_, ok := probe.Sysctl8Int("kern.securelevel", func([]string) (string, error) {
		return "", errors.New("sysctl: unknown oid")
	})
	if ok {
		t.Fatal("error is unknown")
	}
}

func TestDefaultHost_Sysctl8OverlayWhenUnixMissing(t *testing.T) {
	h := probe.DefaultHost{
		Sysctl: nil,
		Sysctl8: func(name string) (string, error) {
			if name != "kern.securelevel" {
				t.Fatalf("name %q", name)
			}
			return "0\n", nil
		},
	}
	v, ok := h.SysctlInt("kern.securelevel")
	if !ok || v != 0 {
		t.Fatalf("host overlay sysctl(8): %d %v", v, ok)
	}
}
