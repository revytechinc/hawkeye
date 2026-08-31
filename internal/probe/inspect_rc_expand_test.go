// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package probe

import "testing"

func TestExpandRCSubrName(t *testing.T) {
	cases := []struct {
		cmd, name, want string
	}{
		{"/usr/sbin/${name}", "sshd", "/usr/sbin/sshd"},
		{"/usr/sbin/$name", "sshd", "/usr/sbin/sshd"},
		{"/usr/sbin/$name-extra", "sshd", "/usr/sbin/sshd-extra"},
		{"/opt/$namespace/bin/sshd", "sshd", "/opt/$namespace/bin/sshd"},
		{"/usr/sbin/${name}", "", "/usr/sbin/${name}"},
		{"", "sshd", ""},
		{"/usr/sbin/sshd", "sshd", "/usr/sbin/sshd"},
		{"${name}$name", "sshd", "sshdsshd"},
	}
	for _, tc := range cases {
		got := expandRCSubrName(tc.cmd, tc.name)
		if got != tc.want {
			t.Fatalf("expandRCSubrName(%q, %q)=%q want %q", tc.cmd, tc.name, got, tc.want)
		}
	}
}

func TestRCCommandExpandsName(t *testing.T) {
	got := rcCommand("#!/bin/sh\ncommand=\"/usr/sbin/${name}\"\n", "sshd")
	if got != "/usr/sbin/sshd" {
		t.Fatalf("got %q", got)
	}
	got = rcCommand("# comment\nprocname=/usr/sbin/$name\n", "ntpd")
	if got != "/usr/sbin/ntpd" {
		t.Fatalf("procname got %q", got)
	}
	if rcCommand("# only comments\n", "sshd") != "" {
		t.Fatal("empty body")
	}
}

func TestRCBinaryMissingSkipsUnexpanded(t *testing.T) {
	if !rcBinaryMissing(Sources{}, "/usr/sbin/sshd") {
		t.Fatal("literal missing path must be missing when Sources is empty")
	}
	if rcBinaryMissing(Sources{}, "/opt/$prefix/sshd") {
		t.Fatal("unexpanded $ must not be treated as a missing binary")
	}
	if rcBinaryMissing(Sources{}, "sshd") {
		t.Fatal("PATH-relative names are not existence-checked")
	}
}

func TestRCBinaryMissingLookPath(t *testing.T) {
	src := Sources{LookPath: func(name string) string {
		if name == "sshd" {
			return "/rescue/sshd"
		}
		return ""
	}}
	if rcBinaryMissing(src, "/usr/sbin/sshd") {
		t.Fatal("LookPath hit must not be missing")
	}
	src.LookPath = func(string) string { return "" }
	if !rcBinaryMissing(src, "/usr/sbin/sshd") {
		t.Fatal("empty LookPath still missing")
	}
}
