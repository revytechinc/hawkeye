// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package doctor_test

import (
	"encoding/json"
	"os"
	"path/filepath"
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

func TestRun_MissingOptionalGGUFIsNoteNotFail(t *testing.T) {
	cfg := config.Default()
	cfg.LLM.Local.ModelPath = ""
	r := doctor.Run(doctor.Deps{
		Cfg:         cfg,
		KnowledgeOK: true,
		Headroom:    headroom.Snapshot{RAMFreeBytes: 1 << 30},
		Probe:       probe.Snapshot{Tier: 1},
	})
	if !r.Healthy {
		t.Fatalf("missing optional GGUF must not fail doctor: %+v", r)
	}
	found := false
	for _, c := range r.Checks {
		if c.Name != "local_llm" {
			continue
		}
		found = true
		if !c.OK {
			t.Fatalf("local_llm must stay ok: %+v", c)
		}
		if !strings.Contains(strings.ToLower(c.Detail), "optional") {
			t.Fatalf("want optional-GGUF note: %q", c.Detail)
		}
	}
	if !found {
		t.Fatal("doctor must note optional local GGUF")
	}
}

func TestRun_SecurelevelKnownIsNoteNotFail(t *testing.T) {
	r := doctor.Run(doctor.Deps{
		Cfg:         config.Default(),
		KnowledgeOK: true,
		Headroom:    headroom.Snapshot{RAMFreeBytes: 1 << 30},
		Probe:       probe.Snapshot{Tier: 1, Securelevel: 1, SecurelevelOK: true},
	})
	if !r.Healthy {
		t.Fatalf("known securelevel must not fail doctor: %+v", r)
	}
	found := false
	for _, c := range r.Checks {
		if c.Name != "securelevel" {
			continue
		}
		found = true
		if !c.OK || !strings.Contains(c.Detail, "kern.securelevel=1") {
			t.Fatalf("%+v", c)
		}
	}
	if !found {
		t.Fatal("doctor must report sysctl(8) securelevel")
	}
	if !strings.Contains(r.Human(), "securelevel") {
		t.Fatalf("human missing securelevel:\n%s", r.Human())
	}
}

func TestRun_SecurelevelUnknownIsNoteNotFail(t *testing.T) {
	r := doctor.Run(doctor.Deps{
		Cfg:         config.Default(),
		KnowledgeOK: true,
		Headroom:    headroom.Snapshot{RAMFreeBytes: 1 << 30},
		Probe:       probe.Snapshot{Tier: 2},
	})
	if !r.Healthy {
		t.Fatalf("unknown securelevel must not fail doctor: %+v", r)
	}
	found := false
	for _, c := range r.Checks {
		if c.Name != "securelevel" {
			continue
		}
		found = true
		if !c.OK || !strings.Contains(c.Detail, "unknown") {
			t.Fatalf("%+v", c)
		}
	}
	if !found {
		t.Fatal("missing securelevel check")
	}
}

func TestRun_PresentOptionalGGUFIsStillHealthy(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "tiny.gguf")
	if err := os.WriteFile(p, []byte("not-a-real-gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.LLM.Local.ModelPath = p
	r := doctor.Run(doctor.Deps{
		Cfg:         cfg,
		KnowledgeOK: true,
		Headroom:    headroom.Snapshot{RAMFreeBytes: 1 << 30},
		Probe:       probe.Snapshot{Tier: 1},
	})
	if !r.Healthy {
		t.Fatalf("present optional GGUF must stay healthy: %+v", r)
	}
	for _, c := range r.Checks {
		if c.Name == "local_llm" && !strings.Contains(c.Detail, "present") {
			t.Fatalf("want present: %q", c.Detail)
		}
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
	for _, want := range []string{"config", "permissions", "pidfile", "dependencies", "headroom", "local_llm", "securelevel"} {
		if !names[want] {
			t.Fatalf("missing check %s in %#v", want, names)
		}
	}
}

func TestRun_UnhealthyWhenPidfileMode0600(t *testing.T) {
	r := doctor.Run(doctor.Deps{
		Cfg:         config.Default(),
		PidRunning:  true,
		PidContent:  "4242\n",
		PidOwnerOK:  true,
		PidMode:     0o600,
		KnowledgeOK: true,
		Headroom:    headroom.Snapshot{RAMFreeBytes: 1 << 30},
		Probe:       probe.Snapshot{Tier: 1},
	})
	if r.Healthy {
		t.Fatal("0600 pidfile must be unhealthy even when root can read it")
	}
	for _, c := range r.Checks {
		if c.Name != "pidfile" {
			continue
		}
		if strings.Contains(c.Detail, "empty") {
			t.Fatalf("0600 must not be reported as empty: %q", c.Detail)
		}
		if !strings.Contains(c.Detail, "0600") && !strings.Contains(c.Detail, "world-readable") {
			t.Fatalf("want world-readable/0600: %q", c.Detail)
		}
		return
	}
	t.Fatal("missing pidfile check")
}

func TestRun_HealthyWhenPidfileReadable0644(t *testing.T) {
	r := doctor.Run(doctor.Deps{
		Cfg:         config.Default(),
		PidRunning:  true,
		PidContent:  "4242\n",
		PidOwnerOK:  true,
		PidMode:     0o644,
		KnowledgeOK: true,
		Headroom:    headroom.Snapshot{RAMFreeBytes: 1 << 30},
		Probe:       probe.Snapshot{Tier: 1},
	})
	if !r.Healthy {
		t.Fatalf("0644 pidfile must be healthy for operator doctor: %+v", r)
	}
}

func TestRun_UnreadablePidfileNotEmpty(t *testing.T) {
	r := doctor.Run(doctor.Deps{
		Cfg:         config.Default(),
		PidRunning:  true,
		PidReadErr:  "pidfile is unreadable: permission denied",
		PidOwnerOK:  true,
		KnowledgeOK: true,
		Headroom:    headroom.Snapshot{RAMFreeBytes: 1 << 30},
		Probe:       probe.Snapshot{Tier: 1},
	})
	if r.Healthy {
		t.Fatal("unreadable pidfile must be unhealthy")
	}
	for _, c := range r.Checks {
		if c.Name != "pidfile" {
			continue
		}
		if strings.Contains(c.Detail, "empty") {
			t.Fatalf("unreadable must not be reported as empty: %q", c.Detail)
		}
		if !strings.Contains(c.Detail, "unreadable") {
			t.Fatalf("want unreadable: %q", c.Detail)
		}
		return
	}
	t.Fatal("missing pidfile check")
}
