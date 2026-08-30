// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package probe_test

import (
	"testing"

	"github.com/revytechinc/hawkeye/internal/probe"
)

func TestLive_DoesNotPanic(t *testing.T) {
	h := probe.Live()
	s := probe.Probe(h)
	_ = s.FirstSkill()
	_ = h.GPUPresent()
	_ = h.NetworkCarrier()
	_ = h.PathExists("/")
	_ = h.PathWritable(t.TempDir())
	_, _ = h.SysctlInt("kern.securelevel")
	_ = h.MountReadOnly("/")
}
