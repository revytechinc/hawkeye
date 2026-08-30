// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package update_test

import (
	"testing"

	"github.com/revytechinc/hawkeye/internal/probe"
	"github.com/revytechinc/hawkeye/internal/update"
)

func TestRun_MissingArgs(t *testing.T) {
	if _, err := update.Run("", "d", probe.Snapshot{}); err == nil {
		t.Fatal("src")
	}
	if _, err := update.Run("s", "", probe.Snapshot{}); err == nil {
		t.Fatal("dest")
	}
	if _, err := update.Run("/no/such/src", "/tmp/x", probe.Snapshot{}); err == nil {
		t.Fatal("open")
	}
}
