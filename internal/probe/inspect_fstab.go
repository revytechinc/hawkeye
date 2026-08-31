// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package probe

import (
	"os"
	"strconv"
	"strings"
)

var knownVFS = map[string]bool{
	"ufs": true, "zfs": true, "nfs": true, "nfs4": true, "nullfs": true,
	"unionfs": true, "tmpfs": true, "procfs": true, "fdescfs": true,
	"linprocfs": true, "linsysfs": true, "devfs": true, "cd9660": true,
	"msdosfs": true, "ntfs": true, "ext2fs": true, "udf": true,
	"smbfs": true, "fuse": true, "fusefs": true, "autofs": true,
	"swap": true, "mfs": true, "nwfs": true, "coda": true,
}

var skipFSCK = map[string]bool{
	"procfs": true, "fdescfs": true, "devfs": true, "tmpfs": true,
	"nullfs": true, "linprocfs": true, "linsysfs": true, "swap": true,
	"autofs": true, "fuse": true, "fusefs": true,
}

var requiredAtBoot = map[string]bool{
	"/": true, "/usr": true, "/var": true,
}

type fstabEnt struct {
	spec, file, vfs, opts string
	dump, pass            string
	dumpN, passN          int
	hasDump, hasPass      bool
}

func inspectFstab(r *Report, src Sources, mounts string) {
	if src.ReadFile == nil && !src.Live && src.Root == "" {
		return
	}
	raw, err := src.read("/etc/fstab")
	if err != nil {
		if os.IsNotExist(err) {
			r.add("fstab", "/etc/fstab is missing; mounts may not come back after reboot")
		}
		return
	}
	ents := parseFstab(string(raw))
	mounted := mountedSet(mounts)
	for _, e := range ents {
		opts := splitOpts(e.opts)
		noauto := opts["noauto"]
		if requiredAtBoot[e.file] && noauto {
			r.add("fstab", "fstab "+e.file+" is noauto; that mount is required at boot")
		}
		if e.vfs != "" && !knownVFS[e.vfs] {
			r.add("fstab", "fstab "+e.file+" uses unknown vfs "+e.vfs+"; check the type or load the module")
		}
		if !skipFSCK[e.vfs] {
			if e.hasDump && (e.dumpN < 0 || e.dumpN > 1) {
				r.add("fstab", "fstab "+e.file+" dump="+e.dump+" is nonsense; use 0 or 1")
			}
			if e.hasPass && (e.passN < 0 || e.passN > 2) {
				r.add("fstab", "fstab "+e.file+" pass="+e.pass+" is nonsense; use 0, 1, or 2")
			}
			if e.file == "/" && e.vfs == "ufs" && e.hasPass && e.passN == 0 {
				r.add("fstab", "fstab / has pass=0; fsck will skip UFS root")
			}
		}
		if e.vfs == "swap" {
			continue
		}
		if noauto && !requiredAtBoot[e.file] {
			continue
		}
		if _, ok := mounted[e.file]; !ok {
			r.add("fstab", "fstab lists "+e.spec+" on "+e.file+" but it is not mounted")
		}
	}
}

func parseFstab(text string) []fstabEnt {
	var out []fstabEnt
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Fields(line)
		if len(f) < 3 {
			continue
		}
		e := fstabEnt{spec: f[0], file: f[1], vfs: f[2]}
		if len(f) >= 4 {
			e.opts = f[3]
		}
		if len(f) >= 5 {
			e.dump = f[4]
			if n, err := strconv.Atoi(f[4]); err == nil {
				e.dumpN, e.hasDump = n, true
			} else {
				e.hasDump = true
				e.dumpN = 99
			}
		}
		if len(f) >= 6 {
			e.pass = f[5]
			if n, err := strconv.Atoi(f[5]); err == nil {
				e.passN, e.hasPass = n, true
			} else {
				e.hasPass = true
				e.passN = 99
			}
		}
		out = append(out, e)
	}
	return out
}

func splitOpts(opts string) map[string]bool {
	m := map[string]bool{}
	for _, o := range strings.Split(opts, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			m[o] = true
		}
	}
	return m
}
