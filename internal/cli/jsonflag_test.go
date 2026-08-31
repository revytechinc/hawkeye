// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package cli

import "testing"

func TestWantJSON(t *testing.T) {
	if wantJSON(flagset{}, nil) {
		t.Fatal("nil getenv must not force JSON")
	}
	if wantJSON(flagset{json: false}, func(string) string { return "" }) {
		t.Fatal("empty HAWKEYE_JSON must be human")
	}
	if !wantJSON(flagset{json: true}, nil) {
		t.Fatal("--json must force JSON")
	}
	if !wantJSON(flagset{}, func(string) string { return "true" }) {
		t.Fatal("HAWKEYE_JSON=true")
	}
	if !wantJSON(flagset{}, func(string) string { return "1" }) {
		t.Fatal("HAWKEYE_JSON=1")
	}
}
