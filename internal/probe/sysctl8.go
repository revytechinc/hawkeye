// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package probe

import (
	"os/exec"
	"strconv"
	"strings"
)

// Sysctl8Run runs `sysctl -n <name>` (sysctl(8)). Tests replace it.
var Sysctl8Run = defaultSysctl8Run

// DefaultSysctl8Run is the production runner (reset tests with this).
var DefaultSysctl8Run = defaultSysctl8Run

func defaultSysctl8Run(name string) (string, error) {
	bin := "/sbin/sysctl"
	if _, err := exec.LookPath(bin); err != nil {
		bin = "sysctl"
	}
	out, err := exec.Command(bin, "-n", name).CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// ParseSysctl8Output parses `sysctl -n` stdout into an int.
func ParseSysctl8Output(out string) (int, bool) {
	s := strings.TrimSpace(out)
	if s == "" {
		return 0, false
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return v, true
}

// Sysctl8Int reads an integer OID via sysctl(8) (host overlay).
func Sysctl8Int(name string) (int, bool) {
	run := Sysctl8Run
	if run == nil {
		run = defaultSysctl8Run
	}
	out, err := run(name)
	if err != nil {
		return 0, false
	}
	return ParseSysctl8Output(out)
}
