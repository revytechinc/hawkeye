// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package probe

import (
	"strings"
)

func inspectNet(r *Report, h Host, src Sources) {
	carrier := false
	if src.Ifaces != nil {
		if ifaces, err := src.Ifaces(); err == nil {
			carrier = CarrierUp(ifaces)
		}
	} else if h != nil {
		carrier = h.NetworkCarrier()
	}
	if !carrier {
		r.add("net", "no network carrier on any non-loopback interface; check the cable and ifconfig")
	}

	routes := ""
	if src.Routes != nil {
		if b, err := src.Routes(); err == nil {
			routes = b
		}
	} else if src.Live {
		if b, err := liveRoutes(); err == nil {
			routes = b
		}
	}
	hasDefault := false
	if src.Routes != nil || src.Live {
		hasDefault = hasDefaultRoute(routes)
		if !hasDefault {
			r.add("net", "no default route; add a gateway before off-link traffic will work")
		}
	}

	netExpected := carrier || hasDefault
	if !netExpected {
		return
	}
	if src.ReadFile == nil && !src.Live && src.Root == "" {
		return
	}
	raw, err := src.read("/etc/resolv.conf")
	if err != nil {
		r.add("net", "/etc/resolv.conf is missing; names will not resolve")
		return
	}
	if !resolvHasNameserver(string(raw)) {
		r.add("net", "/etc/resolv.conf has no nameserver; DNS will fail while the net is up")
	}
}

func hasDefaultRoute(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "Destination") {
			continue
		}
		low := strings.ToLower(trim)
		if strings.HasPrefix(trim, "default") || strings.HasPrefix(trim, "0.0.0.0") || strings.HasPrefix(trim, "::/0") {
			return true
		}
		if strings.Contains(low, "route to: default") || strings.Contains(low, "destination: default") {
			return true
		}
		fields := strings.Fields(trim)
		if len(fields) > 0 && (fields[0] == "default" || fields[0] == "0.0.0.0/0") {
			return true
		}
	}
	return false
}

func resolvHasNameserver(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		if strings.HasPrefix(trim, "nameserver") {
			f := strings.Fields(trim)
			if len(f) >= 2 {
				return true
			}
		}
	}
	return false
}
