// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package probe

import "strings"

func inspectZFS(r *Report, src Sources) {
	list := ""
	if src.ZpoolList != nil {
		b, err := src.ZpoolList()
		if err != nil {
			return
		}
		list = b
	} else if src.Live {
		b, err := liveCmd("zpool", "list", "-H", "-o", "name,health")
		if err != nil {
			return
		}
		list = b
	} else {
		return
	}
	if strings.TrimSpace(list) == "" {
		return
	}

	status := ""
	if src.ZpoolStatus != nil {
		if b, err := src.ZpoolStatus(); err == nil {
			status = b
		}
	} else if src.Live {
		if b, err := liveCmd("zpool", "status"); err == nil {
			status = b
		}
	}
	props := ""
	if src.ZpoolGet != nil {
		if b, err := src.ZpoolGet(); err == nil {
			props = b
		}
	} else if src.Live {
		if b, err := liveCmd("zpool", "get", "-H", "readonly,bootfs"); err == nil {
			props = b
		}
	}

	for _, line := range strings.Split(list, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "NAME") {
			continue
		}
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		name, health := f[0], strings.ToUpper(f[len(f)-1])
		switch health {
		case "DEGRADED", "FAULTED", "UNAVAIL", "SUSPENDED":
			r.add("zfs", "zpool "+name+" is "+health+"; check zpool status and replace the bad vdev")
		}
	}

	for _, bad := range zpoolBadVdevs(status) {
		r.add("zfs", "zpool vdev "+bad.name+" is "+bad.state+"; take it offline or replace it")
	}

	ro, bootfs := parseZpoolProps(props)
	for pool, on := range ro {
		if on {
			r.add("zfs", "zpool "+pool+" is readonly; zfs set readonly=off only after you intend to write")
		}
	}
	// bootfs: report only when a pool looks like the system pool and bootfs is unset
	for pool, fs := range bootfs {
		if fs == "" || fs == "-" {
			if looksLikeRootPool(pool, status) {
				r.add("zfs", "zpool "+pool+" has no bootfs; set bootfs before the next reboot")
			}
		}
	}
}

type vdevState struct {
	name, state string
}

func zpoolBadVdevs(status string) []vdevState {
	var out []vdevState
	inConfig := false
	for _, line := range strings.Split(status, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "config:") {
			inConfig = true
			continue
		}
		if inConfig && (trim == "" || strings.HasPrefix(trim, "errors:")) {
			if strings.HasPrefix(trim, "errors:") {
				inConfig = false
			}
			continue
		}
		if !inConfig {
			continue
		}
		f := strings.Fields(trim)
		if len(f) < 2 {
			continue
		}
		name, st := f[0], strings.ToUpper(f[1])
		if name == "NAME" || name == "mirror" || strings.HasPrefix(name, "mirror-") || strings.HasPrefix(name, "raidz") {
			continue
		}
		switch st {
		case "FAULTED", "UNAVAIL", "OFFLINE", "REMOVED", "DEGRADED":
			// pool-level DEGRADED is already reported from list
			if name != "" && !strings.EqualFold(st, "ONLINE") {
				if looksLikePoolHeader(name, status) && st == "DEGRADED" {
					continue
				}
				out = append(out, vdevState{name: name, state: st})
			}
		}
	}
	return out
}

func looksLikePoolHeader(name, status string) bool {
	return strings.Contains(status, "pool: "+name)
}

func looksLikeRootPool(name, status string) bool {
	if name == "zroot" || name == "root" || strings.Contains(name, "root") {
		return true
	}
	return strings.Contains(status, "pool: "+name)
}

func parseZpoolProps(text string) (readonly map[string]bool, bootfs map[string]string) {
	readonly = map[string]bool{}
	bootfs = map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		f := strings.Fields(line)
		if len(f) < 3 {
			continue
		}
		pool, prop, val := f[0], f[1], f[2]
		switch prop {
		case "readonly":
			readonly[pool] = strings.EqualFold(val, "on") || val == "yes"
		case "bootfs":
			bootfs[pool] = val
		}
	}
	return readonly, bootfs
}
