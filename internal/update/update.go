// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package update

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/revytechinc/hawkeye/internal/probe"
)

const (
	ArtifactName    = "knowledge.sqlite"
	DefaultDest     = "/usr/local/share/hawkeye/knowledge.sqlite"
	SourceEnv       = "HAWKEYE_UPDATE_SOURCE"
	LegacySourceEnv = "HAWKEYE_DATA_ARTIFACT"
)

func ResolveDest(dest string) string {
	if dest == "" {
		return DefaultDest
	}
	return dest
}

func Run(src, dest string, snap probe.Snapshot) (string, error) {
	if src == "" {
		// Unset source is a healthy no-op so rc start does not fail.
		return "", nil
	}
	if snap.RootRO {
		return "", fmt.Errorf("root is read-only; refuse update (first skill is unlock-rw)")
	}
	dest = ResolveDest(dest)
	in, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", err
	}
	tmp := dest + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		_ = os.Remove(tmp)
		return "", err
	}
	if err := out.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, dest); err != nil {
		return "", err
	}
	return dest, nil
}
