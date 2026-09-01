// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package doctor

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/revytechinc/hawkeye/internal/config"
	"github.com/revytechinc/hawkeye/internal/headroom"
	"github.com/revytechinc/hawkeye/internal/pidfile"
	"github.com/revytechinc/hawkeye/internal/probe"
)

type Check struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

type Report struct {
	Healthy   bool              `json:"healthy"`
	Checks    []Check           `json:"checks"`
	Resources headroom.Snapshot `json:"resources"`
	Tier      int               `json:"tier"`
}

type Deps struct {
	ConfigPath      string
	Cfg             config.Config
	Probe           probe.Snapshot
	Headroom        headroom.Snapshot
	PidRunning      bool
	PidContent      string
	PidReadErr      string
	PidOwnerOK      bool
	PidMode         int
	ConfigMode      int
	KnowledgeOK     bool
	KnowledgeDetail string
}

func Run(d Deps) Report {
	r := Report{
		Resources: d.Headroom,
		Tier:      d.Probe.Tier,
	}

	cfgOK := true
	cfgDetail := "configuration is valid"
	if err := config.Validate(d.Cfg); err != nil {
		cfgOK = false
		cfgDetail = err.Error()
	}
	r.Checks = append(r.Checks, Check{Name: "config", OK: cfgOK, Detail: cfgDetail})

	permOK := true
	permDetail := "config permissions acceptable"
	if d.ConfigMode != 0 {
		if d.ConfigMode&0o002 != 0 {
			permOK = false
			permDetail = fmt.Sprintf("config is world-writable (mode %04o)", d.ConfigMode)
		} else {
			permDetail = fmt.Sprintf("config mode %04o", d.ConfigMode)
		}
	}
	r.Checks = append(r.Checks, Check{Name: "permissions", OK: permOK, Detail: permDetail})

	pidOK := true
	pidDetail := "service is not running; pidfile not required"
	if d.PidReadErr != "" {
		pidOK = false
		pidDetail = d.PidReadErr
	} else if d.PidMode != 0 && !pidfile.OperatorReadable(os.FileMode(d.PidMode)) {
		pidOK = false
		pidDetail = fmt.Sprintf("pidfile is not world-readable (mode %04o); operator doctor cannot read it", d.PidMode)
	} else if d.PidRunning {
		pidDetail = "pidfile present"
		s := strings.TrimSpace(d.PidContent)
		if s == "" {
			pidOK = false
			pidDetail = "pidfile is empty"
		} else {
			n, err := strconv.Atoi(s)
			if err != nil || n < 0 {
				pidOK = false
				pidDetail = "pidfile must be a non-negative integer"
			} else if n == 0 {
				pidOK = false
				pidDetail = "pidfile must not be zero"
			} else if !d.PidOwnerOK {
				pidOK = false
				pidDetail = "pidfile owner is not the service user"
			} else {
				pidDetail = fmt.Sprintf("pid %d", n)
			}
		}
	}
	r.Checks = append(r.Checks, Check{Name: "pidfile", OK: pidOK, Detail: pidDetail})

	depOK := d.KnowledgeOK
	depDetail := d.KnowledgeDetail
	if depDetail == "" {
		if depOK {
			depDetail = "knowledge store reachable"
		} else {
			depDetail = "knowledge store missing"
		}
	}
	r.Checks = append(r.Checks, Check{Name: "dependencies", OK: depOK, Detail: depDetail})

	headOK := true
	headDetail := "resource snapshot recorded"
	job := headroom.Job{NeedRAM: true}
	if err := headroom.Allow(job, d.Headroom, d.Cfg.Resources.RAMMinFreeBytes, d.Cfg.Resources.CPUMaxPct, nil, nil); err != nil {
		headOK = false
		headDetail = err.Error()
	} else if !d.Headroom.GPUPresent {
		headDetail = "gpu absent (ok; not required for this operation)"
	}
	r.Checks = append(r.Checks, Check{Name: "headroom", OK: headOK, Detail: headDetail})

	modelDetail := "optional local GGUF missing (consult skips quietly)"
	if p := strings.TrimSpace(d.Cfg.LLM.Local.ModelPath); p != "" {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			modelDetail = "optional local GGUF present"
		}
	}
	r.Checks = append(r.Checks, Check{Name: "local_llm", OK: true, Detail: modelDetail})

	slOK := true
	slDetail := "kern.securelevel unknown (sysctl(8) not available)"
	if d.Probe.SecurelevelOK {
		slDetail = fmt.Sprintf("kern.securelevel=%d (sysctl(8))", d.Probe.Securelevel)
	}
	r.Checks = append(r.Checks, Check{Name: "securelevel", OK: slOK, Detail: slDetail})

	r.Healthy = true
	for _, c := range r.Checks {
		if !c.OK {
			r.Healthy = false
			break
		}
	}
	return r
}

func (r Report) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

func (r Report) Human() string {
	var b strings.Builder
	if r.Healthy {
		b.WriteString("hawkeye doctor: healthy\n")
	} else {
		b.WriteString("hawkeye doctor: UNHEALTHY\n")
	}
	fmt.Fprintf(&b, "tier: %d\n", r.Tier)
	for _, c := range r.Checks {
		mark := "ok"
		if !c.OK {
			mark = "FAIL"
		}
		fmt.Fprintf(&b, "  %-14s %s  %s\n", c.Name, mark, c.Detail)
	}
	fmt.Fprintf(&b, "resources: ram_free=%d disk_free=%d gpu=%v\n", r.Resources.RAMFreeBytes, r.Resources.DiskFreeBytes, r.Resources.GPUPresent)
	return b.String()
}
