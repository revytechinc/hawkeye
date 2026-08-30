// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

//go:build !freebsd

package headroom

func liveRAM() (int64, int64) { return 0, 0 }
