// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package consult_test

import (
	"testing"

	"github.com/revytechinc/hawkeye/internal/consult"
	"github.com/revytechinc/hawkeye/internal/probe"
)

func TestResultJSON(t *testing.T) {
	r, err := consult.Run("hello", probe.Snapshot{Tier: 2}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := r.JSON()
	if err != nil || len(b) == 0 {
		t.Fatal(err)
	}
}
