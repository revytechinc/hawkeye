// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package doctor_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/revytechinc/hawkeye/internal/config"
	"github.com/revytechinc/hawkeye/internal/doctor"
	"github.com/revytechinc/hawkeye/internal/headroom"
	"github.com/revytechinc/hawkeye/internal/probe"
)

func TestRun_UnhealthyWhenConfigBad(t *testing.T) {
	cfg := config.Default()
	cfg.LogLevel = "LOUD"
	r := doctor.Run(doctor.Deps{
		ConfigPath:  "/tmp/nope.json",
		Cfg:         cfg,
		Probe:       probe.Snapshot{Tier: 1},
		Headroom:    headroom.Snapshot{RAMFreeBytes: 1 << 30},
		KnowledgeOK: true,
	})
	if r.Healthy {
		t.Fatal("doctor must be unhealthy on bad config")
	}
}

func TestRun_UnhealthyWhenPidfileEmpty(t *testing.T) {
	r := doctor.Run(doctor.Deps{
		Cfg:         config.Default(),
		PidRunning:  true,
		PidContent:  "",
		PidOwnerOK:  true,
		KnowledgeOK: true,
		Headroom:    headroom.Snapshot{RAMFreeBytes: 1 << 30},
		Probe:       probe.Snapshot{Tier: 1},
	})
	if r.Healthy {
		t.Fatal("empty pidfile must be unhealthy")
	}
}

func TestRun_HealthyHappyPath(t *testing.T) {
	r := doctor.Run(doctor.Deps{
		Cfg:             config.Default(),
		PidRunning:      false,
		KnowledgeOK:     true,
		KnowledgeDetail: "open ro /boot/hawkeye/knowledge.sqlite",
		Headroom:        headroom.Snapshot{RAMFreeBytes: 1 << 30, DiskFreeBytes: 1 << 30},
		Probe:           probe.Snapshot{Tier: 1, UsrPresent: true, VarPresent: true},
		ConfigMode:      0o600,
	})
	if !r.Healthy {
		t.Fatalf("want healthy: %+v", r)
	}
	b, err := r.JSON()
	if err != nil || !json.Valid(b) {
		t.Fatalf("json: %v %s", err, b)
	}
	h := r.Human()
	if !strings.Contains(strings.ToLower(h), "healthy") {
		t.Fatalf("human: %q", h)
	}
}

func TestRun_ReportsAllResourcesEvenUnused(t *testing.T) {
	r := doctor.Run(doctor.Deps{
		Cfg:         config.Default(),
		KnowledgeOK: true,
		Headroom: headroom.Snapshot{
			RAMFreeBytes:  1 << 30,
			CPUPct:        3,
			DiskFreeBytes: 1 << 32,
			GPUPresent:    false,
		},
		Probe: probe.Snapshot{Tier: 1},
	})
	if r.Resources.RAMFreeBytes == 0 {
		t.Fatal("doctor must report RAM even if this job would not use GPU")
	}
	names := map[string]bool{}
	for _, c := range r.Checks {
		names[c.Name] = true
	}
	for _, want := range []string{"config", "permissions", "pidfile", "dependencies", "headroom"} {
		if !names[want] {
			t.Fatalf("missing check %s in %#v", want, names)
		}
	}
}
