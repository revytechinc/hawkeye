// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

//go:build !freebsd

package probe

import "net"

func liveSysctlInt(string) (int, bool) { return 0, false }

func liveMountTable() (string, error) { return "", nil }

func liveStatfsReadOnly(string) bool { return false }

func liveNetworkCarrier() bool {
	ifaces, err := net.Interfaces()
	if err != nil {
		return false
	}
	return CarrierUp(statusesFromGoIfaces(ifaces))
}
