// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/revytechinc/hawkeye/internal/config"
)

func TestShippedExampleJSON(t *testing.T) {
	p := filepath.Join("..", "..", "configs", "config.example.json")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	c, err := config.Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	if c.Listen.MCPHTTP != "127.0.0.1:8741" {
		t.Fatal(c.Listen.MCPHTTP)
	}
}
