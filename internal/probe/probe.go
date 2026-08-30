// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package probe

type Host interface {
	SysctlInt(name string) (int, bool)
	PathExists(path string) bool
	PathWritable(path string) bool
	MountReadOnly(path string) bool
	NetworkCarrier() bool
	GPUPresent() bool
}

type Snapshot struct {
	Securelevel   int  `json:"securelevel"`
	SecurelevelOK bool `json:"securelevel_known"`
	RootRO        bool `json:"root_readonly"`
	UsrPresent    bool `json:"usr_present"`
	VarPresent    bool `json:"var_present"`
	NetworkUp     bool `json:"network_up"`
	GPUPresent    bool `json:"gpu_present"`
	RescuePresent bool `json:"rescue_present"`
	ZFSReadOnly   bool `json:"zfs_readonly"`
	Tier          int  `json:"tier"`
}

// Probe classifies the host into Hawkeye tiers:
// 0 rescue (RO root, maybe no /usr /var, no net, no GPU),
// 1 root writable and islanded,
// 2 net up (remote LLM / GPU optional).
func Probe(h Host) Snapshot {
	s := Snapshot{}
	if v, ok := h.SysctlInt("kern.securelevel"); ok {
		s.Securelevel = v
		s.SecurelevelOK = true
	}
	s.UsrPresent = h.PathExists("/usr")
	s.VarPresent = h.PathExists("/var")
	s.RescuePresent = h.PathExists("/rescue")
	s.NetworkUp = h.NetworkCarrier()
	s.GPUPresent = h.GPUPresent()
	s.ZFSReadOnly = h.MountReadOnly("/")
	s.RootRO = s.ZFSReadOnly || h.MountReadOnly("/") || !h.PathWritable("/")
	switch {
	case s.RootRO || !s.UsrPresent || !s.VarPresent:
		s.Tier = 0
	case !s.NetworkUp:
		s.Tier = 1
	default:
		s.Tier = 2
	}
	return s
}

// FirstSkill is unlock-rw when the root is read-only. Never "pkg" on a RO root.
func (s Snapshot) FirstSkill() string {
	if s.RootRO {
		return "unlock-rw"
	}
	return ""
}
