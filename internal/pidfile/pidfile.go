// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package pidfile

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func Write(path string, pid int) error {
	if pid <= 0 {
		return fmt.Errorf("pid must be positive, got %d", pid)
	}
	return os.WriteFile(path, []byte(strconv.Itoa(pid)+"\n"), 0o644)
}

func Read(path string) (int, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	s := strings.TrimSpace(string(b))
	if s == "" {
		return 0, fmt.Errorf("pidfile is empty")
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	if n < 0 {
		return 0, fmt.Errorf("pidfile must not be negative")
	}
	if n == 0 {
		return 0, fmt.Errorf("pidfile must not be zero")
	}
	return n, nil
}

func Remove(path string) error {
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
