// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package probe

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"unicode"
)

// ParseSysctlN parses `sysctl -n` stdout (a single integer, optional newline).
func ParseSysctlN(out string) (int, bool) {
	s := strings.TrimSpace(out)
	if s == "" {
		return 0, false
	}
	for _, r := range s {
		if r != '-' && !unicode.IsDigit(r) {
			return 0, false
		}
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}

// SignedSysctl32 interprets a 32-bit sysctl as signed. kern.securelevel is
// -1 in the insecure default; unix.SysctlUint32 reports that as 4294967295.
func SignedSysctl32(u uint32) int {
	return int(int32(u))
}

func safeMIB(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '_' {
			continue
		}
		return false
	}
	return true
}

func sysctl8Bin() string {
	for _, p := range []string{"/sbin/sysctl", "/usr/sbin/sysctl", "/rescue/sysctl"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if p, err := exec.LookPath("sysctl"); err == nil {
		return p
	}
	return ""
}

func execSysctl8(argv []string) (string, error) {
	if len(argv) == 0 {
		return "", exec.ErrNotFound
	}
	out, err := exec.Command(argv[0], argv[1:]...).Output()
	return string(out), err
}

// Sysctl8Int runs sysctl(8) -n name via run (tests inject). Production uses
// /sbin/sysctl, then /usr/sbin/sysctl, then /rescue/sysctl.
func Sysctl8Int(name string, run func([]string) (string, error)) (int, bool) {
	if !safeMIB(name) || run == nil {
		return 0, false
	}
	bin := sysctl8Bin()
	if bin == "" {
		bin = "sysctl"
	}
	out, err := run([]string{bin, "-n", name})
	if err != nil {
		return 0, false
	}
	return ParseSysctlN(out)
}

func liveSysctl8Int(name string) (int, bool) {
	return Sysctl8Int(name, execSysctl8)
}
