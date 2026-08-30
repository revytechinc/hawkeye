// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package version_test

import (
	"testing"

	"github.com/revytechinc/hawkeye/internal/version"
)

func TestConstants(t *testing.T) {
	if version.Number == "" || version.Product != "hawkeye" {
		t.Fatal(version.Number, version.Product, version.FullName)
	}
}
