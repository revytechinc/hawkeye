// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package probe

import (
	"path/filepath"
	"strings"
)

func inspectRC(r *Report, src Sources) {
	if src.ReadFile == nil && !src.Live && src.Root == "" {
		return
	}
	raw, err := src.read("/etc/rc.conf")
	if err != nil {
		return
	}
	text := string(raw)
	if rcConfSyntaxBroken(text) {
		r.add("rc", "/etc/rc.conf has unmatched quotes; fix syntax before service starts")
	}
	for _, name := range rcEnabled(text) {
		script, ok := findRCScript(src, name)
		if !ok {
			r.add("rc", name+"_enable=YES but the rc.d script is missing; install the package or disable it")
			continue
		}
		body, err := src.read(script)
		if err != nil {
			continue
		}
		cmd := rcCommand(string(body))
		if cmd == "" || cmd == "daemon" {
			continue
		}
		if rcBinaryMissing(src, cmd) {
			r.add("rc", name+"_enable=YES but "+cmd+" is missing; restore the binary or disable "+name)
		}
	}
}

func rcConfSyntaxBroken(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		if unmatchedQuotes(trim) {
			return true
		}
	}
	return false
}

func unmatchedQuotes(s string) bool {
	dq, sq := 0, 0
	esc := false
	for _, r := range s {
		if esc {
			esc = false
			continue
		}
		if r == '\\' {
			esc = true
			continue
		}
		switch r {
		case '"':
			dq++
		case '\'':
			sq++
		}
	}
	return dq%2 == 1 || sq%2 == 1
}

func rcEnabled(text string) []string {
	var out []string
	seen := map[string]bool{}
	for _, line := range strings.Split(text, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		key, val, ok := splitAssign(trim)
		if !ok || !strings.HasSuffix(key, "_enable") {
			continue
		}
		name := strings.TrimSuffix(key, "_enable")
		if name == "" || name == "rc" {
			continue
		}
		if !rcYes(val) {
			continue
		}
		if seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

func splitAssign(line string) (key, val string, ok bool) {
	i := strings.IndexByte(line, '=')
	if i <= 0 {
		return "", "", false
	}
	key = strings.TrimSpace(line[:i])
	val = strings.TrimSpace(line[i+1:])
	val = strings.Trim(val, `"'`)
	return key, val, true
}

func rcYes(v string) bool {
	switch strings.ToUpper(strings.TrimSpace(v)) {
	case "YES", "TRUE", "ON", "1":
		return true
	}
	return false
}

func findRCScript(src Sources, name string) (string, bool) {
	for _, dir := range []string{"/etc/rc.d", "/usr/local/etc/rc.d"} {
		p := filepath.Join(dir, name)
		if src.exists(p) {
			return p, true
		}
	}
	return "", false
}

func rcCommand(body string) string {
	for _, line := range strings.Split(body, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		if strings.HasPrefix(trim, "command=") || strings.HasPrefix(trim, "procname=") {
			_, val, ok := splitAssign(trim)
			if ok {
				return val
			}
		}
	}
	return ""
}

func rcBinaryMissing(src Sources, cmd string) bool {
	if cmd == "" || !strings.Contains(cmd, "/") {
		return false
	}
	if src.exists(cmd) {
		return false
	}
	if src.LookPath != nil {
		if p := src.LookPath(filepath.Base(cmd)); p != "" {
			return false
		}
	}
	return true
}
