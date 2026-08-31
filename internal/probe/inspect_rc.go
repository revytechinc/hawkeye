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
		cmd := rcCommand(string(body), name)
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

func rcCommand(body, scriptName string) string {
	for _, line := range strings.Split(body, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		if strings.HasPrefix(trim, "command=") || strings.HasPrefix(trim, "procname=") {
			_, val, ok := splitAssign(trim)
			if ok {
				return expandRCSubrName(val, scriptName)
			}
		}
	}
	return ""
}

// expandRCSubrName applies the same ${name}/$name substitution rc.subr
// uses before exec: name is the rc.d script basename.
func expandRCSubrName(cmd, name string) string {
	if name == "" || cmd == "" {
		return cmd
	}
	cmd = strings.ReplaceAll(cmd, "${name}", name)
	return expandDollarName(cmd, name)
}

func expandDollarName(cmd, name string) string {
	const tok = "$name"
	var b strings.Builder
	for i := 0; i < len(cmd); {
		if strings.HasPrefix(cmd[i:], tok) {
			end := i + len(tok)
			if end == len(cmd) || !isRCIdent(cmd[end]) {
				b.WriteString(name)
				i = end
				continue
			}
		}
		b.WriteByte(cmd[i])
		i++
	}
	return b.String()
}

func isRCIdent(c byte) bool {
	return c == '_' ||
		(c >= 'A' && c <= 'Z') ||
		(c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9')
}

func rcBinaryMissing(src Sources, cmd string) bool {
	if cmd == "" || !strings.Contains(cmd, "/") {
		return false
	}
	if strings.Contains(cmd, "$") {
		// Other rc.subr vars are unresolved; diagnose-only — do not guess.
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
