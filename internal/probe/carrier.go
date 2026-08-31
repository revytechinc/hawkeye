// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package probe

import (
	"net"
	"strings"
	"unicode"
)

// ifmedia(4) status bits from FreeBSD net/if_media.h.
const (
	IFM_AVALID = 0x00000001
	IFM_ACTIVE = 0x00000002
)

// IfaceStatus is one network interface after getifaddrs, SIOCGIFMEDIA,
// or ifconfig(8) text. Tests inject this instead of Linux /sys.
type IfaceStatus struct {
	Name         string
	Loopback     bool
	Up           bool
	Running      bool
	CarrierKnown bool
	Carrier      bool
}

// CarrierUp is true when a non-loopback interface has link.
// SIOCGIFMEDIA / ifconfig "status:" wins over IFF_RUNNING when known.
func CarrierUp(ifaces []IfaceStatus) bool {
	for _, iface := range ifaces {
		if iface.Loopback || IsLoopbackName(iface.Name) {
			continue
		}
		if iface.CarrierKnown {
			if iface.Carrier {
				return true
			}
			continue
		}
		if iface.Up && iface.Running {
			return true
		}
	}
	return false
}

// IsLoopbackName matches lo(4) names (lo, lo0, lo1).
func IsLoopbackName(name string) bool {
	if name == "lo" {
		return true
	}
	if !strings.HasPrefix(name, "lo") {
		return false
	}
	rest := name[2:]
	if rest == "" {
		return true
	}
	for _, r := range rest {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// IfmediaActive decodes ifm_status from SIOCGIFMEDIA.
func IfmediaActive(status int32) (active bool, valid bool) {
	if status&IFM_AVALID == 0 {
		return false, false
	}
	return status&IFM_ACTIVE != 0, true
}

// ParseIfconfig reads ifconfig -a text (FreeBSD). Not Linux sysfs.
func ParseIfconfig(text string) []IfaceStatus {
	var out []IfaceStatus
	var cur *IfaceStatus
	flush := func() {
		if cur == nil {
			return
		}
		out = append(out, *cur)
		cur = nil
	}
	for _, line := range strings.Split(text, "\n") {
		if name, ok := ifconfigHeader(line); ok {
			flush()
			st := IfaceStatus{Name: name, Loopback: IsLoopbackName(name)}
			up, running, loopback := ifconfigFlags(line)
			st.Up = up
			st.Running = running
			st.Loopback = st.Loopback || loopback
			cur = &st
			continue
		}
		if cur == nil {
			continue
		}
		trim := strings.TrimSpace(line)
		if !strings.HasPrefix(trim, "status:") {
			continue
		}
		status := strings.TrimSpace(strings.TrimPrefix(trim, "status:"))
		cur.CarrierKnown = true
		cur.Carrier = status == "active"
	}
	flush()
	return out
}

func ifconfigHeader(line string) (string, bool) {
	if line == "" || line[0] == '\t' || line[0] == ' ' {
		return "", false
	}
	colon := strings.IndexByte(line, ':')
	if colon <= 0 {
		return "", false
	}
	name := line[:colon]
	if !isIfaceName(name) {
		return "", false
	}
	return name, true
}

func isIfaceName(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			continue
		}
		if r == '.' || r == '_' {
			if i == 0 {
				return false
			}
			continue
		}
		return false
	}
	return true
}

func ifconfigFlags(line string) (up, running, loopback bool) {
	i := strings.IndexByte(line, '<')
	j := strings.IndexByte(line, '>')
	if i < 0 || j <= i {
		return
	}
	for _, f := range strings.Split(line[i+1:j], ",") {
		switch f {
		case "UP":
			up = true
		case "RUNNING":
			running = true
		case "LOOPBACK":
			loopback = true
		}
	}
	return
}

func statusesFromGoIfaces(ifaces []net.Interface) []IfaceStatus {
	out := make([]IfaceStatus, 0, len(ifaces))
	for _, iface := range ifaces {
		out = append(out, IfaceStatus{
			Name:     iface.Name,
			Loopback: iface.Flags&net.FlagLoopback != 0 || IsLoopbackName(iface.Name),
			Up:       iface.Flags&net.FlagUp != 0,
			Running:  iface.Flags&net.FlagRunning != 0,
		})
	}
	return out
}
