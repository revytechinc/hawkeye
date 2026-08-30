// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package doctor_test

import (
	"strings"
	"testing"

	"github.com/revytechinc/hawkeye/internal/config"
	"github.com/revytechinc/hawkeye/internal/doctor"
	"github.com/revytechinc/hawkeye/internal/headroom"
	"github.com/revytechinc/hawkeye/internal/probe"
)

func TestRun_PidNegativeAndWorldWritable(t *testing.T) {
	r := doctor.Run(doctor.Deps{
		Cfg:         config.Default(),
		PidRunning:  true,
		PidContent:  "-1",
		PidOwnerOK:  true,
		ConfigMode:  0o666,
		KnowledgeOK: true,
		Headroom:    headroom.Snapshot{RAMFreeBytes: 1 << 30},
		Probe:       probe.Snapshot{Tier: 1},
	})
	if r.Healthy {
		t.Fatal("expected unhealthy")
	}
	h := r.Human()
	if !strings.Contains(h, "UNHEALTHY") {
		t.Fatal(h)
	}
}

func TestRun_PidZeroAndOwner(t *testing.T) {
	r := doctor.Run(doctor.Deps{
		Cfg:         config.Default(),
		PidRunning:  true,
		PidContent:  "0",
		PidOwnerOK:  false,
		KnowledgeOK: true,
		Headroom:    headroom.Snapshot{RAMFreeBytes: 1 << 30},
		Probe:       probe.Snapshot{},
	})
	if r.Healthy {
		t.Fatal("unhealthy")
	}
}

func TestRun_LowRAM(t *testing.T) {
	r := doctor.Run(doctor.Deps{
		Cfg:         config.Default(),
		KnowledgeOK: true,
		Headroom:    headroom.Snapshot{RAMFreeBytes: 1},
		Probe:       probe.Snapshot{Tier: 2},
	})
	if r.Healthy {
		t.Fatal("low ram should fail headroom")
	}
}

func TestRun_PidHappyRunning(t *testing.T) {
	r := doctor.Run(doctor.Deps{
		Cfg:         config.Default(),
		PidRunning:  true,
		PidContent:  "42\n",
		PidOwnerOK:  true,
		KnowledgeOK: true,
		ConfigMode:  0o600,
		Headroom:    headroom.Snapshot{RAMFreeBytes: 1 << 30, GPUPresent: false},
		Probe:       probe.Snapshot{Tier: 1},
	})
	if !r.Healthy {
		t.Fatalf("%+v", r)
	}
}
