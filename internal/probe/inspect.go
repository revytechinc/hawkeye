// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package probe

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Finding is one host first-look line. Human() prints Text only — no JSON keys.
type Finding struct {
	Area string `json:"area"`
	Text string `json:"text"`
}

// Report is diagnose-only host first-look. Not hawkeye doctor (pidfile/config).
type Report struct {
	Findings []Finding `json:"findings"`
}

func (r Report) Human() string {
	var b strings.Builder
	for _, f := range r.Findings {
		if t := strings.TrimSpace(f.Text); t != "" {
			b.WriteString(t)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (r Report) JSON() ([]byte, error) {
	if r.Findings == nil {
		r.Findings = []Finding{}
	}
	return json.MarshalIndent(r, "", "  ")
}

func (r *Report) add(area, text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	r.Findings = append(r.Findings, Finding{Area: area, Text: text})
}

// DiskUse is one mounted filesystem's capacity. Tests inject FAKE values.
type DiskUse struct {
	TotalBytes  int64
	FreeBytes   int64
	TotalInodes int64
	FreeInodes  int64
}

// Sources is the host view first-look reads. Tests inject FAKE trees and
// command output. Zero Sources is Host-only (root RO, /usr, carrier) and
// never opens the live machine's /etc.
type Sources struct {
	// Live permits reading the real host. Production Run() sets this.
	Live bool

	Root     string
	ReadFile func(string) ([]byte, error)
	Stat     func(string) (os.FileInfo, error)

	MountTable  func() (string, error)
	Ifaces      func() ([]IfaceStatus, error)
	ZpoolList   func() (string, error)
	ZpoolStatus func() (string, error)
	ZpoolGet    func() (string, error)
	Routes      func() (string, error)
	GeliStatus  func() (string, error)
	Disk        func(path string) (DiskUse, bool)
	LookPath    func(name string) string
}

func (s Sources) path(p string) string {
	if s.Root == "" {
		return p
	}
	return filepath.Join(s.Root, strings.TrimPrefix(p, "/"))
}

func (s Sources) read(p string) ([]byte, error) {
	if s.ReadFile == nil {
		if !s.Live {
			return nil, os.ErrNotExist
		}
		return os.ReadFile(s.path(p))
	}
	return s.ReadFile(s.path(p))
}

func (s Sources) exists(p string) bool {
	if s.Stat != nil {
		_, err := s.Stat(s.path(p))
		return err == nil
	}
	if s.ReadFile != nil {
		_, err := s.ReadFile(s.path(p))
		return err == nil
	}
	if !s.Live {
		return false
	}
	_, err := os.Stat(s.path(p))
	return err == nil
}

func (s Sources) mounts() string {
	if s.MountTable != nil {
		if b, err := s.MountTable(); err == nil {
			return b
		}
	}
	if !s.Live {
		return ""
	}
	b, err := liveMountTable()
	if err != nil {
		return ""
	}
	return b
}

// Inspect is diagnose-only. It never mutates. Silence is OK when a subsystem is fine.
func Inspect(h Host, src Sources) Report {
	var r Report
	mounts := src.mounts()
	inspectRoot(&r, h, src, mounts)
	inspectFstab(&r, src, mounts)
	inspectZFS(&r, src)
	inspectRC(&r, src)
	inspectNet(&r, h, src)
	inspectDisk(&r, src, mounts)
	inspectGeli(&r, src)
	return r
}

func inspectRoot(r *Report, h Host, src Sources, mounts string) {
	ro := false
	if h != nil {
		ro = h.MountReadOnly("/")
	}
	fstype := mountFSType(mounts, "/")
	if !ro {
		if roFound, found := MountPointReadOnly(mounts, "/"); found {
			ro = roFound
			if fstype == "" {
				fstype = mountFSType(mounts, "/")
			}
		}
	}
	usr := false
	if h != nil {
		usr = h.PathExists("/usr")
	}
	if !usr && src.exists("/usr") {
		usr = true
	}
	if ro {
		switch fstype {
		case "zfs":
			r.add("root", "root is read-only (zfs); unlock-rw (zfs set readonly=off) before pkg")
		case "ufs":
			r.add("root", "root is read-only (ufs); remount rw before writes")
		case "":
			r.add("root", "root is read-only; unlock-rw before pkg")
		default:
			r.add("root", "root is read-only ("+fstype+"); remount rw before writes")
		}
	}
	if !usr {
		r.add("root", "/usr is missing; mount it or you are in rescue/single-user")
	}
}

func mountFSType(table, mountpoint string) string {
	for _, e := range parseMountLines(table) {
		if e.file == mountpoint {
			return e.vfs
		}
	}
	return ""
}

func mountedSet(table string) map[string]mountEnt {
	out := map[string]mountEnt{}
	for _, e := range parseMountLines(table) {
		out[e.file] = e
	}
	return out
}

type mountEnt struct {
	spec, file, vfs, opts string
}

func parseMountLines(table string) []mountEnt {
	var ents []mountEnt
	for _, line := range strings.Split(table, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Fields(line)
		if len(f) < 3 {
			continue
		}
		e := mountEnt{spec: f[0], file: f[1], vfs: f[2]}
		if len(f) >= 4 {
			e.opts = f[3]
		}
		ents = append(ents, e)
	}
	return ents
}
