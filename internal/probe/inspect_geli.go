// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package probe

import "strings"

func inspectGeli(r *Report, src Sources) {
	text := ""
	if src.GeliStatus != nil {
		b, err := src.GeliStatus()
		if err != nil {
			return
		}
		text = b
	} else if src.Live {
		b, err := liveCmd("geli", "status")
		if err != nil {
			return
		}
		text = b
	} else {
		return
	}
	if strings.TrimSpace(text) == "" {
		return
	}
	for _, name := range geliUnavailable(text) {
		r.add("geli", "geli "+name+": keystatus unavailable; attach the provider before mounting")
	}
}

func geliUnavailable(text string) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "Name") {
			continue
		}
		low := strings.ToLower(trim)
		if !strings.Contains(low, "unavail") && !strings.Contains(low, "unavailable") {
			continue
		}
		f := strings.Fields(trim)
		if len(f) == 0 {
			continue
		}
		name := strings.TrimSuffix(f[0], ":")
		out = append(out, name)
	}
	return out
}
