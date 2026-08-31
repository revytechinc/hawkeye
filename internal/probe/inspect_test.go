// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package probe_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/revytechinc/hawkeye/internal/probe"
)

func TestInspect_RORootMissingUsrNoCarrier(t *testing.T) {
	h := fakeHost{
		exists: map[string]bool{"/usr": false, "/var": false, "/rescue": true},
		ro:     map[string]bool{"/": true},
	}
	rep := probe.Inspect(h, probe.Sources{})
	text := rep.Human()
	if text == "" {
		t.Fatal("RO root / missing /usr / no carrier must not be silent")
	}
	if !strings.Contains(strings.ToLower(text), "read-only") && !strings.Contains(text, "RO") {
		t.Fatalf("want root read-only:\n%s", text)
	}
	if !strings.Contains(text, "/usr") {
		t.Fatalf("want missing /usr:\n%s", text)
	}
	if !strings.Contains(strings.ToLower(text), "carrier") {
		t.Fatalf("want no carrier:\n%s", text)
	}
	if strings.Contains(text, "pidfile") || strings.Contains(text, `"root_readonly"`) {
		t.Fatalf("first-look is not doctor/JSON keys:\n%s", text)
	}
	assertOneFindingPerLine(t, text)
}

func TestInspect_FakeFstabVsMounts(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"etc/fstab": `# Device Mountpoint FStype Options Dump Pass#
/dev/gpt/root / ufs rw 1 1
/dev/gpt/var /var ufs rw 2 2
/dev/gpt/usr /usr ufs rw,noauto 2 2
/dev/gpt/mystery /mnt mysteryfs rw 9 9
fdesc /dev/fd fdescfs rw 0 0
`,
		"etc/rc.conf":     "hostname=\"panicbox\"\n",
		"etc/resolv.conf": "# empty on purpose\n",
	})
	src := probe.Sources{
		Root:     root,
		ReadFile: os.ReadFile,
		Stat:     os.Stat,
		MountTable: func() (string, error) {
			return `/dev/gpt/root / ufs ro 0 0
fdesc /dev/fd fdescfs rw 0 0
`, nil
		},
	}
	h := fakeHost{
		exists:  map[string]bool{"/usr": false, "/var": false, "/rescue": true},
		ro:      map[string]bool{"/": true},
		carrier: false,
	}
	rep := probe.Inspect(h, src)
	text := rep.Human()
	if !strings.Contains(text, "/var") || !strings.Contains(strings.ToLower(text), "not mounted") {
		t.Fatalf("want fstab /var not mounted:\n%s", text)
	}
	if !strings.Contains(text, "noauto") {
		t.Fatalf("want noauto required-at-boot /usr:\n%s", text)
	}
	if !strings.Contains(strings.ToLower(text), "mysteryfs") || !strings.Contains(strings.ToLower(text), "unknown") {
		t.Fatalf("want unknown vfs:\n%s", text)
	}
	if !strings.Contains(strings.ToLower(text), "dump") && !strings.Contains(strings.ToLower(text), "pass") {
		t.Fatalf("want dump/pass nonsense:\n%s", text)
	}
	if !strings.Contains(strings.ToLower(text), "read-only") {
		t.Fatalf("want RO root with type:\n%s", text)
	}
	if !strings.Contains(strings.ToLower(text), "ufs") {
		t.Fatalf("want filesystem type ufs:\n%s", text)
	}
	assertActionableHuman(t, text)
}

func TestInspect_MissingFstab(t *testing.T) {
	root := t.TempDir()
	src := probe.Sources{Root: root, ReadFile: os.ReadFile, Stat: os.Stat}
	rep := probe.Inspect(fakeHost{exists: map[string]bool{"/usr": true}}, src)
	if !strings.Contains(rep.Human(), "/etc/fstab") {
		t.Fatalf("want missing fstab:\n%s", rep.Human())
	}
}

func TestInspect_MissingEnableScriptAndBinary(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"etc/fstab": "/dev/gpt/root / ufs rw 1 1\n",
		"etc/rc.conf": `sshd_enable="YES"
ntpd_enable="YES"
# commented_enable="YES"
hostname="box"
`,
		"etc/rc.d/ntpd":   "#!/bin/sh\n# PROVIDE: ntpd\ncommand=\"/usr/sbin/ntpd\"\n",
		"etc/resolv.conf": "nameserver 9.9.9.9\n",
	})
	src := probe.Sources{
		Root:     root,
		ReadFile: os.ReadFile,
		Stat:     os.Stat,
		MountTable: func() (string, error) {
			return "/dev/gpt/root / ufs rw 0 0\n", nil
		},
	}
	h := fakeHost{
		exists:  map[string]bool{"/usr": true, "/var": true},
		carrier: true,
	}
	text := probe.Inspect(h, src).Human()
	if !strings.Contains(text, "sshd") || !strings.Contains(strings.ToLower(text), "missing") {
		t.Fatalf("want missing sshd rc.d script:\n%s", text)
	}
	if !strings.Contains(text, "ntpd") || !strings.Contains(text, "/usr/sbin/ntpd") {
		t.Fatalf("want missing ntpd binary:\n%s", text)
	}
	if strings.Contains(text, "commented") {
		t.Fatalf("commented enable must be silent:\n%s", text)
	}
	assertActionableHuman(t, text)
}

func TestInspect_FullDiskAndInodes(t *testing.T) {
	src := probe.Sources{
		Disk: func(path string) (probe.DiskUse, bool) {
			switch path {
			case "/":
				return probe.DiskUse{TotalBytes: 100, FreeBytes: 0, TotalInodes: 50, FreeInodes: 0}, true
			case "/var":
				return probe.DiskUse{TotalBytes: 1000, FreeBytes: 10, TotalInodes: 100, FreeInodes: 20}, true
			default:
				return probe.DiskUse{}, false
			}
		},
		MountTable: func() (string, error) {
			return `/dev/gpt/root / ufs rw 0 0
/dev/gpt/var /var ufs rw 0 0
`, nil
		},
	}
	h := fakeHost{exists: map[string]bool{"/usr": true, "/var": true}, carrier: true}
	text := probe.Inspect(h, src).Human()
	if !strings.Contains(text, "/") || !strings.Contains(strings.ToLower(text), "full") {
		t.Fatalf("want / full:\n%s", text)
	}
	if !strings.Contains(strings.ToLower(text), "inode") {
		t.Fatalf("want inode exhaustion:\n%s", text)
	}
	if !strings.Contains(text, "/var") {
		t.Fatalf("want /var nearly full:\n%s", text)
	}
	assertActionableHuman(t, text)
}

func TestInspect_ZFSDegradedReadonlyBootfs(t *testing.T) {
	src := probe.Sources{
		ZpoolList: func() (string, error) {
			return "zroot\tDEGRADED\n", nil
		},
		ZpoolStatus: func() (string, error) {
			return `  pool: zroot
 state: DEGRADED
status: One or more devices is unavailable.
config:
	NAME        STATE
	zroot       DEGRADED
	  ada0p4    FAULTED
`, nil
		},
		ZpoolGet: func() (string, error) {
			return "zroot\treadonly\ton\t-\nzroot\tbootfs\t-\t-\n", nil
		},
	}
	h := fakeHost{exists: map[string]bool{"/usr": true}, carrier: true}
	text := probe.Inspect(h, src).Human()
	if !strings.Contains(text, "zroot") || !strings.Contains(text, "DEGRADED") {
		t.Fatalf("want degraded pool:\n%s", text)
	}
	if !strings.Contains(text, "FAULTED") {
		t.Fatalf("want faulted vdev:\n%s", text)
	}
	if !strings.Contains(strings.ToLower(text), "readonly") {
		t.Fatalf("want readonly pool:\n%s", text)
	}
	if !strings.Contains(strings.ToLower(text), "bootfs") {
		t.Fatalf("want missing bootfs:\n%s", text)
	}
	assertActionableHuman(t, text)
}

func TestInspect_NoDefaultRouteEmptyResolvWhenNetExpected(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"etc/resolv.conf": "# no nameserver\n",
	})
	src := probe.Sources{
		Root:     root,
		ReadFile: os.ReadFile,
		Stat:     os.Stat,
		Ifaces: func() ([]probe.IfaceStatus, error) {
			return []probe.IfaceStatus{{Name: "em0", Up: true, Running: true, CarrierKnown: true, Carrier: true}}, nil
		},
		Routes: func() (string, error) {
			return `Destination        Gateway            Flags     Netif
127.0.0.1          link#2             UH        lo0
192.168.1.0/24     link#1             U         em0
`, nil
		},
	}
	h := fakeHost{exists: map[string]bool{"/usr": true}, carrier: true}
	text := probe.Inspect(h, src).Human()
	if !strings.Contains(strings.ToLower(text), "default route") {
		t.Fatalf("want no default route:\n%s", text)
	}
	if !strings.Contains(strings.ToLower(text), "resolv") {
		t.Fatalf("want empty resolv.conf:\n%s", text)
	}
	assertActionableHuman(t, text)
}

func TestInspect_GeliKeystatusUnavailable(t *testing.T) {
	src := probe.Sources{
		GeliStatus: func() (string, error) {
			return `Name          Status    Components
ada0p4.eli    UNAVAIL   ada0p4
`, nil
		},
	}
	h := fakeHost{exists: map[string]bool{"/usr": true}, carrier: true}
	text := probe.Inspect(h, src).Human()
	if !strings.Contains(strings.ToLower(text), "geli") || !strings.Contains(strings.ToLower(text), "unavailable") {
		t.Fatalf("want geli keystatus unavailable:\n%s", text)
	}
	assertActionableHuman(t, text)
}

func TestInspect_RCExpandsNameLikeRCSubr(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"etc/fstab":       "/dev/gpt/root / ufs rw 1 1\n",
		"etc/rc.conf":     "sshd_enable=\"YES\"\nntpd_enable=\"YES\"\n",
		"etc/rc.d/sshd":   "#!/bin/sh\n# PROVIDE: sshd\ncommand=\"/usr/sbin/${name}\"\n",
		"etc/rc.d/ntpd":   "#!/bin/sh\ncommand=/usr/sbin/$name\n",
		"usr/sbin/sshd":   "",
		"usr/sbin/ntpd":   "",
		"etc/resolv.conf": "nameserver 1.1.1.1\n",
	})
	src := probe.Sources{
		Root:     root,
		ReadFile: os.ReadFile,
		Stat:     os.Stat,
		MountTable: func() (string, error) {
			return "/dev/gpt/root / ufs rw 0 0\n", nil
		},
	}
	h := fakeHost{exists: map[string]bool{"/usr": true, "/var": true}, carrier: true}
	text := probe.Inspect(h, src).Human()
	if strings.Contains(text, "${name}") || strings.Contains(text, "/usr/sbin/$name") {
		t.Fatalf("rc.subr expands name; must not report the unexpanded path:\n%s", text)
	}
	if strings.Contains(strings.ToLower(text), "missing") && (strings.Contains(text, "sshd") || strings.Contains(text, "ntpd")) {
		t.Fatalf("stock rc.d ${name}/$name binaries exist; inspect must be silent:\n%s", text)
	}
}

func TestInspect_RCMissingBinaryUsesExpandedName(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"etc/fstab":       "/dev/gpt/root / ufs rw 1 1\n",
		"etc/rc.conf":     "sshd_enable=\"YES\"\n",
		"etc/rc.d/sshd":   "#!/bin/sh\ncommand=\"/usr/sbin/${name}\"\n",
		"etc/resolv.conf": "nameserver 1.1.1.1\n",
	})
	src := probe.Sources{
		Root:     root,
		ReadFile: os.ReadFile,
		Stat:     os.Stat,
		MountTable: func() (string, error) {
			return "/dev/gpt/root / ufs rw 0 0\n", nil
		},
	}
	h := fakeHost{exists: map[string]bool{"/usr": true}, carrier: true}
	text := probe.Inspect(h, src).Human()
	if strings.Contains(text, "${name}") {
		t.Fatalf("missing-binary text must use the expanded path:\n%s", text)
	}
	if !strings.Contains(text, "/usr/sbin/sshd") || !strings.Contains(strings.ToLower(text), "missing") {
		t.Fatalf("want expanded /usr/sbin/sshd missing:\n%s", text)
	}
	assertActionableHuman(t, text)
}

func TestInspect_RCDoesNotExpandNamePrefix(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"etc/fstab":       "/dev/gpt/root / ufs rw 1 1\n",
		"etc/rc.conf":     "sshd_enable=\"YES\"\n",
		"etc/rc.d/sshd":   "#!/bin/sh\ncommand=\"/opt/$namespace/bin/sshd\"\n",
		"etc/resolv.conf": "nameserver 1.1.1.1\n",
	})
	src := probe.Sources{
		Root:     root,
		ReadFile: os.ReadFile,
		Stat:     os.Stat,
		MountTable: func() (string, error) {
			return "/dev/gpt/root / ufs rw 0 0\n", nil
		},
	}
	h := fakeHost{exists: map[string]bool{"/usr": true}, carrier: true}
	text := probe.Inspect(h, src).Human()
	if strings.Contains(text, "sshdspace") || strings.Contains(text, "/opt/sshd") {
		t.Fatalf("$namespace must not be treated as $name:\n%s", text)
	}
	if strings.Contains(strings.ToLower(text), "missing") {
		t.Fatalf("unexpanded rc vars must not false-positive a missing binary:\n%s", text)
	}
}

func TestInspect_HealthyIsSilent(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"etc/fstab":       "/dev/gpt/root / ufs rw 1 1\n/dev/gpt/var /var ufs rw 0 2\n",
		"etc/rc.conf":     "sshd_enable=\"YES\"\nhostname=\"ok\"\n",
		"etc/rc.d/sshd":   "#!/bin/sh\ncommand=\"/usr/sbin/${name}\"\n",
		"usr/sbin/sshd":   "",
		"etc/resolv.conf": "nameserver 1.1.1.1\n",
	})
	src := probe.Sources{
		Root:     root,
		ReadFile: os.ReadFile,
		Stat:     os.Stat,
		MountTable: func() (string, error) {
			return "/dev/gpt/root / ufs rw 0 0\n/dev/gpt/var /var ufs rw 0 0\n", nil
		},
		Ifaces: func() ([]probe.IfaceStatus, error) {
			return []probe.IfaceStatus{{Name: "em0", Up: true, Running: true, CarrierKnown: true, Carrier: true}}, nil
		},
		Routes: func() (string, error) {
			return "default            192.168.1.1        UGS        em0\n", nil
		},
		ZpoolList: func() (string, error) {
			return "zroot\tONLINE\n", nil
		},
		ZpoolStatus: func() (string, error) {
			return "  pool: zroot\n state: ONLINE\n", nil
		},
		ZpoolGet: func() (string, error) {
			return "zroot\treadonly\toff\t-\nzroot\tbootfs\tzroot/ROOT/default\t-\n", nil
		},
		Disk: func(string) (probe.DiskUse, bool) {
			return probe.DiskUse{TotalBytes: 1000, FreeBytes: 400, TotalInodes: 100, FreeInodes: 50}, true
		},
		GeliStatus: func() (string, error) { return "", os.ErrNotExist },
	}
	h := fakeHost{
		exists:  map[string]bool{"/usr": true, "/var": true},
		carrier: true,
	}
	rep := probe.Inspect(h, src)
	if text := strings.TrimSpace(rep.Human()); text != "" {
		t.Fatalf("healthy host must be silent:\n%s", text)
	}
	if len(rep.Findings) != 0 {
		t.Fatalf("findings=%v", rep.Findings)
	}
}

func TestInspect_JSONIsStructuredNotHumanKeys(t *testing.T) {
	h := fakeHost{ro: map[string]bool{"/": true}, exists: map[string]bool{"/usr": false}}
	rep := probe.Inspect(h, probe.Sources{})
	b, err := rep.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(b) {
		t.Fatal(string(b))
	}
	var got probe.Report
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Findings) == 0 {
		t.Fatal("JSON must carry findings")
	}
	if strings.Contains(rep.Human(), `"area"`) || strings.Contains(rep.Human(), `"text"`) {
		t.Fatalf("human must not print JSON keys:\n%s", rep.Human())
	}
}

func TestInspect_UFSRootPassZeroAndOtherFS(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"etc/fstab": "/dev/gpt/root / ufs rw 1 0\n/dev/gpt/swap none swap sw 0 0\n",
	})
	src := probe.Sources{
		Root:     root,
		ReadFile: os.ReadFile,
		Stat:     os.Stat,
		MountTable: func() (string, error) {
			return "/dev/gpt/root / ufs rw 0 0\nnone /tmp tmpfs rw 0 0\n", nil
		},
	}
	h := fakeHost{exists: map[string]bool{"/usr": true}, carrier: true}
	text := probe.Inspect(h, src).Human()
	if !strings.Contains(text, "pass=0") {
		t.Fatalf("want UFS root pass=0:\n%s", text)
	}
	if strings.Contains(strings.ToLower(text), "swap") && strings.Contains(text, "not mounted") {
		t.Fatalf("swap must not be a missing mount:\n%s", text)
	}

	src.MountTable = func() (string, error) { return "/dev/md0 / nfs ro 0 0\n", nil }
	h = fakeHost{ro: map[string]bool{"/": true}, exists: map[string]bool{"/usr": true}, carrier: true}
	text = probe.Inspect(h, src).Human()
	if !strings.Contains(text, "nfs") {
		t.Fatalf("want other fstype on RO root:\n%s", text)
	}
}

func TestInspect_RcConfSyntax(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"etc/rc.conf": "sshd_enable=\"YES\nhostname=broken\n",
	})
	src := probe.Sources{Root: root, ReadFile: os.ReadFile, Stat: os.Stat}
	text := probe.Inspect(fakeHost{exists: map[string]bool{"/usr": true}, carrier: true}, src).Human()
	if !strings.Contains(strings.ToLower(text), "rc.conf") {
		t.Fatalf("want rc.conf syntax:\n%s", text)
	}
}

func TestLiveSources_DoesNotPanic(t *testing.T) {
	src := probe.LiveSources()
	rep := probe.Inspect(probe.Live(), src)
	_ = rep.Human()
	_, _ = rep.JSON()
}

func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, body := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func assertOneFindingPerLine(t *testing.T, text string) {
	t.Helper()
	for _, line := range strings.Split(strings.TrimSpace(text), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "{") {
			t.Fatalf("JSON line in human output: %s", line)
		}
	}
}

func assertActionableHuman(t *testing.T, text string) {
	t.Helper()
	if strings.TrimSpace(text) == "" {
		t.Fatal("expected findings")
	}
	if strings.Contains(text, "pidfile") || strings.Contains(text, "hawkeye doctor") {
		t.Fatalf("first-look must not be doctor:\n%s", text)
	}
	if strings.Contains(text, `"area"`) || strings.Contains(text, `"findings"`) {
		t.Fatalf("human must not print JSON keys:\n%s", text)
	}
	assertOneFindingPerLine(t, text)
}
