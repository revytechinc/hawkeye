// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package headroom

import (
	"fmt"
	"strings"
)

type Snapshot struct {
	RAMFreeBytes     int64   `json:"ram_free_bytes"`
	RAMTotalBytes    int64   `json:"ram_total_bytes"`
	CPUPct           float64 `json:"cpu_pct"`
	DiskFreeBytes    int64   `json:"disk_free_bytes"`
	GPUPresent       bool    `json:"gpu_present"`
	GPUVRAMFreeBytes *int64  `json:"gpu_vram_free_bytes"`
}

type Job struct {
	NeedRAM  bool
	NeedCPU  bool
	NeedDisk bool
	NeedGPU  bool
}

func Allow(job Job, snap Snapshot, ramMin *int64, cpuMax *float64, diskMin *int64, vramMin *int64) error {
	var errs []string
	if job.NeedRAM && ramMin != nil && snap.RAMFreeBytes < *ramMin {
		errs = append(errs, fmt.Sprintf("ram free %d < min %d", snap.RAMFreeBytes, *ramMin))
	}
	if job.NeedCPU && cpuMax != nil && snap.CPUPct > *cpuMax {
		errs = append(errs, fmt.Sprintf("cpu %.1f%% > max %.1f%%", snap.CPUPct, *cpuMax))
	}
	if job.NeedDisk && diskMin != nil && snap.DiskFreeBytes < *diskMin {
		errs = append(errs, fmt.Sprintf("disk free %d < min %d", snap.DiskFreeBytes, *diskMin))
	}
	if job.NeedGPU {
		if !snap.GPUPresent {
			errs = append(errs, "gpu missing")
		} else if vramMin != nil {
			free := int64(0)
			if snap.GPUVRAMFreeBytes != nil {
				free = *snap.GPUVRAMFreeBytes
			}
			if free < *vramMin {
				errs = append(errs, fmt.Sprintf("gpu vram free %d < min %d", free, *vramMin))
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("insufficient headroom: %s", strings.Join(errs, ", "))
	}
	return nil
}
