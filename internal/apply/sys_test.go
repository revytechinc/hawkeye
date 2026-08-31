// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package apply_test

import (
	"strings"
	"testing"

	"github.com/revytechinc/hawkeye/internal/apply"
)

func TestSysExecutor_Echo(t *testing.T) {
	out, errOut, err := apply.SysExecutor{}.Run([]string{"echo", "hawkeye-skeleton"})
	if err != nil {
		t.Fatal(err)
	}
	if errOut != "" {
		t.Fatal(errOut)
	}
	if out == "" {
		t.Fatal("empty stdout")
	}
}

func TestSysExecutor_Empty(t *testing.T) {
	if _, _, err := (apply.SysExecutor{}).Run(nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestSysExecutor_StoredShellLine(t *testing.T) {
	out, errOut, err := apply.SysExecutor{}.Run([]string{"echo hawkeye-shell && echo ok"})
	if err != nil {
		t.Fatal(err, errOut)
	}
	if !strings.Contains(out, "hawkeye-shell") {
		t.Fatalf("stored playbook line must run via sh -c: %q %q", out, errOut)
	}
}
