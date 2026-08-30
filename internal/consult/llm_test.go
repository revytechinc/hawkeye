// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package consult_test

import (
	"testing"

	"github.com/revytechinc/hawkeye/internal/consult"
	"github.com/revytechinc/hawkeye/internal/llm"
	"github.com/revytechinc/hawkeye/internal/probe"
)

func TestRun_Tier1LLMSkipped(t *testing.T) {
	r, err := consult.Run("hello", probe.Snapshot{Tier: 1, RootRO: false}, nil, llm.None{})
	if err != nil {
		t.Fatal(err)
	}
	if r.LLM != nil {
		t.Fatal("none completer should skip")
	}
}
