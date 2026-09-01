// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package probe_test

import (
	"errors"
	"strconv"
	"testing"

	"github.com/revytechinc/hawkeye/internal/probe"
)

func TestParseSysctl8Output(t *testing.T) {
	v, ok := probe.ParseSysctl8Output("1\n")
	if !ok || v != 1 {
		t.Fatalf("got %d %v", v, ok)
	}
	v, ok = probe.ParseSysctl8Output("  -1  \n")
	if !ok || v != -1 {
		t.Fatalf("got %d %v", v, ok)
	}
	if _, ok := probe.ParseSysctl8Output(""); ok {
		t.Fatal("empty")
	}
	if _, ok := probe.ParseSysctl8Output("not-a-number"); ok {
		t.Fatal("junk")
	}
}

func TestSysctl8Int_UsesSysctlBinary(t *testing.T) {
	var gotName string
	probe.Sysctl8Run = func(name string) (string, error) {
		gotName = name
		return "2\n", nil
	}
	t.Cleanup(func() { probe.Sysctl8Run = probe.DefaultSysctl8Run })

	v, ok := probe.Sysctl8Int("kern.securelevel")
	if !ok || v != 2 {
		t.Fatalf("got %d %v", v, ok)
	}
	if gotName != "kern.securelevel" {
		t.Fatalf("name %q", gotName)
	}
}

func TestSysctl8Int_ExecError(t *testing.T) {
	probe.Sysctl8Run = func(string) (string, error) {
		return "", errors.New("sysctl: unknown oid")
	}
	t.Cleanup(func() { probe.Sysctl8Run = probe.DefaultSysctl8Run })
	if _, ok := probe.Sysctl8Int("kern.securelevel"); ok {
		t.Fatal("expected miss")
	}
}

func TestLive_SecurelevelViaSysctl8Overlay(t *testing.T) {
	probe.Sysctl8Run = func(name string) (string, error) {
		if name != "kern.securelevel" {
			return "", errors.New("unexpected " + name)
		}
		return strconv.Itoa(1), nil
	}
	t.Cleanup(func() { probe.Sysctl8Run = probe.DefaultSysctl8Run })

	h := probe.Live()
	// Force the Sysctl hook empty so Live falls through to the host overlay.
	h.Sysctl = nil
	v, ok := h.SysctlInt("kern.securelevel")
	if !ok || v != 1 {
		t.Fatalf("host overlay securelevel: got %d ok=%v", v, ok)
	}
}
