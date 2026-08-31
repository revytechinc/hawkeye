// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package cli

import "strings"

func isKnownCommand(s string) bool {
	switch s {
	case "help", "version", "init", "consult", "plan", "apply", "doctor", "mcp", "update":
		return true
	}
	return false
}

func isSessionQuit(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "quit", "exit", "q":
		return true
	}
	return false
}

func isSessionHelp(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "help", "?":
		return true
	}
	return false
}
