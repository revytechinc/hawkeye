// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package llm_test

import (
	"context"
	"errors"
	"testing"

	"github.com/revytechinc/hawkeye/internal/llm"
)

func TestNone(t *testing.T) {
	_, err := llm.None{}.Complete(context.Background(), llm.Request{})
	if !errors.Is(err, llm.ErrNoModel) {
		t.Fatal(err)
	}
}
