// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package knowledge

import (
	"os"
	"path/filepath"
	"strings"
)

// LstatFunc is os.Lstat. Tests inject fixtures; production never remounts ZFS.
type LstatFunc func(string) (os.FileInfo, error)

func realDir(path string, lstat LstatFunc) bool {
	if lstat == nil {
		lstat = os.Lstat
	}
	fi, err := lstat(path)
	if err != nil || fi == nil {
		return false
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return false
	}
	return fi.IsDir()
}

// CanStageRescue is whether a hawkeye binary may be written at rescueDir.
// destDir is DESTDIR/STAGEDIR: when set, staging is always allowed (package
// build / chroot). A live path is allowed only when it is a real directory.
// A dangling bastille /rescue symlink is not a directory and is skipped.
func CanStageRescue(destDir, rescueDir string, lstat LstatFunc) bool {
	if strings.TrimSpace(destDir) != "" {
		return true
	}
	if strings.TrimSpace(rescueDir) == "" {
		return false
	}
	return realDir(rescueDir, lstat)
}

// CanStageBootKit is whether /boot/hawkeye (or destDir+that prefix) may be
// created. DESTDIR is always allowed. Live: /boot/hawkeye already a real
// directory, or its parent /boot is a real directory. Missing /boot is skip
// (do not invent a boot filesystem).
func CanStageBootKit(destDir, bootHawkeye string, lstat LstatFunc) bool {
	if strings.TrimSpace(destDir) != "" {
		return true
	}
	if strings.TrimSpace(bootHawkeye) == "" {
		return false
	}
	if realDir(bootHawkeye, lstat) {
		return true
	}
	return realDir(filepath.Dir(bootHawkeye), lstat)
}
