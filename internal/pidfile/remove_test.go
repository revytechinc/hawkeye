// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package pidfile_test

import (
	"path/filepath"
	"testing"

	"github.com/revytechinc/hawkeye/internal/pidfile"
)

func TestRemove_MissingOK(t *testing.T) {
	if err := pidfile.Remove(filepath.Join(t.TempDir(), "no.pid")); err != nil {
		t.Fatal(err)
	}
}
