// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package probe_test

import (
	"testing"

	"github.com/revytechinc/hawkeye/internal/probe"
)

type fakeHost struct {
	sysctl   map[string]int
	exists   map[string]bool
	writable map[string]bool
	ro       map[string]bool
	carrier  bool
	gpu      bool
}

func (f fakeHost) SysctlInt(name string) (int, bool) {
	v, ok := f.sysctl[name]
	return v, ok
}
func (f fakeHost) PathExists(path string) bool    { return f.exists[path] }
func (f fakeHost) PathWritable(path string) bool  { return f.writable[path] }
func (f fakeHost) MountReadOnly(path string) bool { return f.ro[path] }
func (f fakeHost) NetworkCarrier() bool           { return f.carrier }
func (f fakeHost) GPUPresent() bool               { return f.gpu }

func TestProbe_RescueROIsTier0AndUnlockRW(t *testing.T) {
	h := fakeHost{
		sysctl:   map[string]int{"kern.securelevel": 1},
		exists:   map[string]bool{"/rescue": true, "/usr": false, "/var": false},
		writable: map[string]bool{"/": false},
		ro:       map[string]bool{"/": true},
		carrier:  false,
		gpu:      false,
	}
	s := probe.Probe(h)
	if !s.RootRO {
		t.Fatalf("expected root RO: %+v", s)
	}
	if s.Tier != 0 {
		t.Fatalf("tier = %d, want 0", s.Tier)
	}
	if s.FirstSkill() != "unlock-rw" {
		t.Fatalf("first skill = %q, want unlock-rw (not pkg)", s.FirstSkill())
	}
	if s.NetworkUp || s.UsrPresent || s.VarPresent {
		t.Fatalf("tier-0 snapshot still sees usr/var/net: %+v", s)
	}
}

func TestProbe_WritableIslandedIsTier1(t *testing.T) {
	h := fakeHost{
		sysctl:   map[string]int{"kern.securelevel": 0},
		exists:   map[string]bool{"/usr": true, "/var": true, "/rescue": true},
		writable: map[string]bool{"/": true, "/usr": true, "/var": true},
		ro:       map[string]bool{"/": false},
		carrier:  false,
		gpu:      false,
	}
	s := probe.Probe(h)
	if s.RootRO {
		t.Fatal("root should be writable")
	}
	if s.Tier != 1 {
		t.Fatalf("tier = %d, want 1", s.Tier)
	}
	if s.FirstSkill() == "unlock-rw" {
		t.Fatal("writable root should not demand unlock-rw")
	}
}

func TestProbe_NetworkUpIsTier2(t *testing.T) {
	h := fakeHost{
		exists:   map[string]bool{"/usr": true, "/var": true},
		writable: map[string]bool{"/": true, "/usr": true, "/var": true},
		carrier:  true,
		gpu:      true,
	}
	s := probe.Probe(h)
	if s.Tier != 2 {
		t.Fatalf("tier = %d, want 2", s.Tier)
	}
	if !s.GPUPresent || !s.NetworkUp {
		t.Fatalf("%+v", s)
	}
}

func TestProbe_ZFSReadOnlyCountsAsRootRO(t *testing.T) {
	h := fakeHost{
		exists:   map[string]bool{"/usr": true, "/var": true},
		writable: map[string]bool{"/": false},
		ro:       map[string]bool{"/": true},
	}
	s := probe.Probe(h)
	if !s.RootRO || s.Tier != 0 {
		t.Fatalf("%+v", s)
	}
}

func TestProbe_RWMountNotRootROWhenAccessFails(t *testing.T) {
	h := fakeHost{
		exists:   map[string]bool{"/usr": true, "/var": true, "/rescue": true},
		writable: map[string]bool{"/": false},
		ro:       map[string]bool{"/": false},
		carrier:  true,
		gpu:      false,
	}
	s := probe.Probe(h)
	if s.RootRO {
		t.Fatalf("rw mount must not be RootRO just because Access(/) failed: %+v", s)
	}
	if s.FirstSkill() == "unlock-rw" {
		t.Fatal("first skill must not be unlock-rw on a writable mount")
	}
	if s.Tier == 0 {
		t.Fatalf("rw root with /usr /var must not be forced to tier 0: %+v", s)
	}
}
