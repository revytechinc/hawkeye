// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

//go:build freebsd

package probe

import "net"

func liveIfaceStatuses() ([]IfaceStatus, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	st := statusesFromGoIfaces(ifaces)
	refineIfmedia(st)
	return st, nil
}
