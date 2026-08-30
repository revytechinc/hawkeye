// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package mcp_test

import (
	"testing"

	"github.com/revytechinc/hawkeye/internal/mcp"
)

func TestHandle_UnknownMethod(t *testing.T) {
	s := mcp.New(mcp.Handlers{})
	resp := s.Handle(mcp.Request{JSONRPC: "2.0", ID: 9, Method: "nope"})
	if resp.Error == nil {
		t.Fatal("expected error")
	}
}

func TestToolsUseApplyGate(t *testing.T) {
	if !mcp.ToolsUseApplyGate() {
		t.Fatal("gate")
	}
	if mcp.NormalizeBind(" 127.0.0.1:8741 ") != "127.0.0.1:8741" {
		t.Fatal("trim")
	}
}
