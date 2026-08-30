// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package reload_test

import (
	"testing"

	"github.com/revytechinc/hawkeye/internal/reload"
)

func TestGet_NilHolder(t *testing.T) {
	var h *reload.Holder
	c := h.Get()
	if c.LogLevel == "" {
		t.Fatal("default")
	}
}
