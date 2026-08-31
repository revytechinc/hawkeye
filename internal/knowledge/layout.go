// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package knowledge

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// MkdirAllFunc is os.MkdirAll. Tests inject fixtures; production never remounts.
type MkdirAllFunc func(path string, perm os.FileMode) error

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
// (do not invent a boot filesystem). A present /boot that is read-only is
// decided at create time by StageBootKit (EROFS/EACCES/EPERM → skip).
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

// IsReadOnlyCreateError is true for EROFS, EACCES, and EPERM (and PathError
// wrappers). Other errors (ENOSPC, …) must fail the install target.
func IsReadOnlyCreateError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, syscall.EROFS) || errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EPERM) {
		return true
	}
	var pe *os.PathError
	if errors.As(err, &pe) && pe != nil && pe.Err != err {
		return IsReadOnlyCreateError(pe.Err)
	}
	return false
}

// BootKitSkipMessage is the live-install note when /boot/hawkeye cannot be
// created because the filesystem is read-only. Same style as
// "install-rescue: skip /rescue (not a real directory)".
func BootKitSkipMessage(bootHawkeye string) string {
	return "install-rescue: skip " + bootHawkeye + " (read-only)"
}

// StageBootKit creates bootHawkeye (typically /boot/hawkeye). DESTDIR is a
// flag: when set, create errors propagate (package/chroot must still stage
// both prefixes). Live: EROFS/EACCES/EPERM is skip with no error — do not
// remount a bastille RO /boot. mkdir is os.MkdirAll when nil.
func StageBootKit(destDir, bootHawkeye string, mkdir MkdirAllFunc) (skipped bool, err error) {
	if mkdir == nil {
		mkdir = os.MkdirAll
	}
	if strings.TrimSpace(bootHawkeye) == "" {
		return true, nil
	}
	err = mkdir(bootHawkeye, 0o755)
	if err == nil {
		return false, nil
	}
	if strings.TrimSpace(destDir) == "" && IsReadOnlyCreateError(err) {
		return true, nil
	}
	return false, err
}
