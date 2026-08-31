// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package cli

import "testing"

func TestIsKnownCommand(t *testing.T) {
	for _, c := range []string{"consult", "plan", "apply", "doctor", "mcp", "update", "init", "version", "help"} {
		if !isKnownCommand(c) {
			t.Fatalf("%q must stay a command", c)
		}
	}
	for _, q := range []string{"ZFS", "zfs", "nope", "quit", ""} {
		if isKnownCommand(q) {
			t.Fatalf("%q must be a session query, not a command", q)
		}
	}
}

func TestParse_UnknownWordsAreQuery(t *testing.T) {
	fs := parse([]string{"ZFS", "root", "is", "read-only"})
	if fs.cmd != "" {
		t.Fatalf("cmd=%q want empty (session query)", fs.cmd)
	}
	if len(fs.rest) != 4 || fs.rest[0] != "ZFS" {
		t.Fatalf("rest=%v", fs.rest)
	}
	fs = parse([]string{"consult", "zfs"})
	if fs.cmd != "consult" || len(fs.rest) != 1 || fs.rest[0] != "zfs" {
		t.Fatalf("consult: cmd=%q rest=%v", fs.cmd, fs.rest)
	}
}

func TestSessionMeta(t *testing.T) {
	if !isSessionQuit("quit") || !isSessionQuit("EXIT") || !isSessionQuit(" q ") {
		t.Fatal("quit words")
	}
	if isSessionQuit("quitter") || isSessionQuit("zfs") {
		t.Fatal("not quit")
	}
	if !isSessionHelp("help") || !isSessionHelp("?") {
		t.Fatal("help")
	}
	if isSessionHelp("helpful") {
		t.Fatal("not help")
	}
}
