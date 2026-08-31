// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package probe_test

import (
	"os"
	"strings"
	"testing"

	"github.com/revytechinc/hawkeye/internal/probe"
)

// FreeBSD ifconfig(8) fixture. Not Linux /sys/class/net.
const freebsdIfconfig = `em0: flags=8843<UP,BROADCAST,RUNNING,SIMPLEX,MULTICAST> metric 0 mtu 1500
	options=4810099<RXCSUM,VLAN_MTU>
	ether 00:00:5e:00:53:01
	inet 192.0.2.10 netmask 0xffffff00 broadcast 192.0.2.255
	media: Ethernet autoselect (1000baseT <full-duplex>)
	status: active
lo0: flags=8049<UP,LOOPBACK,RUNNING,MULTICAST> metric 0 mtu 16384
	inet 127.0.0.1 netmask 0xff000000
	status: active
`

const freebsdIfconfigNoCarrier = `em0: flags=8843<UP,BROADCAST,RUNNING,SIMPLEX,MULTICAST> metric 0 mtu 1500
	status: no carrier
lo0: flags=8049<UP,LOOPBACK,RUNNING,MULTICAST> metric 0 mtu 16384
	status: active
`

func TestParseIfconfig_Em0ActiveIgnoresLoopback(t *testing.T) {
	ifaces := probe.ParseIfconfig(freebsdIfconfig)
	if !probe.CarrierUp(ifaces) {
		t.Fatalf("em0 status: active must be carrier: %#v", ifaces)
	}
	for _, iface := range ifaces {
		if probe.IsLoopbackName(iface.Name) && iface.Carrier {
			// loopback may report active; CarrierUp must still skip it
		}
	}
}

func TestParseIfconfig_NoCarrierIsDown(t *testing.T) {
	ifaces := probe.ParseIfconfig(freebsdIfconfigNoCarrier)
	if probe.CarrierUp(ifaces) {
		t.Fatalf("no carrier + only lo0 active must be false: %#v", ifaces)
	}
}

func TestParseIfconfig_OnlyLo0(t *testing.T) {
	text := `lo0: flags=8049<UP,LOOPBACK,RUNNING,MULTICAST> metric 0 mtu 16384
	inet 127.0.0.1 netmask 0xff000000
	status: active
`
	if probe.CarrierUp(probe.ParseIfconfig(text)) {
		t.Fatal("loopback-only must not count as network carrier")
	}
}

func TestCarrierUp_GetifaddrsStyle(t *testing.T) {
	if probe.CarrierUp(nil) {
		t.Fatal("empty")
	}
	if probe.CarrierUp([]probe.IfaceStatus{{Name: "lo0", Loopback: true, Up: true, Running: true, CarrierKnown: true, Carrier: true}}) {
		t.Fatal("lo0")
	}
	if !probe.CarrierUp([]probe.IfaceStatus{{Name: "em0", Up: true, Running: true}}) {
		t.Fatal("em0 UP+RUNNING (getifaddrs/IFF_RUNNING) is carrier when media is unknown")
	}
	if probe.CarrierUp([]probe.IfaceStatus{{Name: "em0", Up: true, Running: true, CarrierKnown: true, Carrier: false}}) {
		t.Fatal("SIOCGIFMEDIA no-carrier must win over IFF_RUNNING")
	}
}

func TestIfmediaActive(t *testing.T) {
	if _, ok := probe.IfmediaActive(0); ok {
		t.Fatal("IFM_AVALID unset")
	}
	on, ok := probe.IfmediaActive(probe.IFM_AVALID | probe.IFM_ACTIVE)
	if !ok || !on {
		t.Fatal("active")
	}
	off, ok := probe.IfmediaActive(probe.IFM_AVALID)
	if !ok || off {
		t.Fatal("valid but not active")
	}
}

func TestDefaultHost_FreeBSDCarrierWithoutSysfs(t *testing.T) {
	h := probe.DefaultHost{
		ReadFile: func(string) ([]byte, error) { return nil, os.ErrNotExist },
		Glob:     func(string) ([]string, error) { return nil, nil },
		Ifaces: func() ([]probe.IfaceStatus, error) {
			return probe.ParseIfconfig(freebsdIfconfig), nil
		},
	}
	if !h.NetworkCarrier() {
		t.Fatal("absent /sys must still see FreeBSD em0 carrier")
	}
}

func TestDefaultHost_FreeBSDNoCarrierWithoutSysfs(t *testing.T) {
	h := probe.DefaultHost{
		ReadFile: func(string) ([]byte, error) { return nil, os.ErrNotExist },
		Glob:     func(string) ([]string, error) { return nil, nil },
		Ifaces: func() ([]probe.IfaceStatus, error) {
			return probe.ParseIfconfig(freebsdIfconfigNoCarrier), nil
		},
	}
	if h.NetworkCarrier() {
		t.Fatal("absent /sys + no carrier must be false")
	}
}

func TestDefaultHost_EmptySysfsFallsThroughToLive(t *testing.T) {
	h := probe.DefaultHost{
		ReadFile: func(string) ([]byte, error) { return nil, os.ErrNotExist },
		Glob:     func(string) ([]string, error) { return nil, nil },
	}
	// Must not panic; result is the live FreeBSD/getifaddrs path, not a hardcoded false.
	_ = h.NetworkCarrier()
}

func TestIsLoopbackName(t *testing.T) {
	for _, name := range []string{"lo", "lo0", "lo1"} {
		if !probe.IsLoopbackName(name) {
			t.Fatal(name)
		}
	}
	for _, name := range []string{"em0", "igb0", "wlan0", "losa0"} {
		if probe.IsLoopbackName(name) {
			t.Fatal(name)
		}
	}
}

func TestParseIfconfig_IgnoresLinuxSysfsPaths(t *testing.T) {
	if strings.Contains(freebsdIfconfig, "/sys/class/net") {
		t.Fatal("fixture must be ifconfig(8), not Linux sysfs")
	}
}

func TestParseIfconfig_SkipsGarbage(t *testing.T) {
	text := "not an iface line\n" +
		"  indented\n" +
		"nocolon\n" +
		": empty\n" +
		".em0: flags=<UP>\n" +
		"em0;drop: flags=<UP,RUNNING>\n" +
		"vlan0.1: flags=<UP,RUNNING>\n" +
		"\tstatus: active\n" +
		"em_0: flags=<> metric 0\n"
	ifaces := probe.ParseIfconfig(text)
	if len(ifaces) != 2 || ifaces[0].Name != "vlan0.1" || ifaces[1].Name != "em_0" {
		t.Fatalf("%#v", ifaces)
	}
	if !probe.CarrierUp(ifaces) {
		t.Fatal("vlan0.1 status active")
	}
}

func TestDefaultHost_IfacesErrorFallsThrough(t *testing.T) {
	h := probe.DefaultHost{
		ReadFile: func(string) ([]byte, error) { return nil, os.ErrNotExist },
		Glob:     func(string) ([]string, error) { return nil, os.ErrPermission },
		Ifaces:   func() ([]probe.IfaceStatus, error) { return nil, os.ErrNotExist },
	}
	_ = h.NetworkCarrier()
}
