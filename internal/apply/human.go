// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package apply

import (
	"fmt"
	"strings"

	"github.com/revytechinc/hawkeye/internal/redact"
)

// Human returns the operator TTY view of a plan: summary, then each step
// as an action and the command as it would be typed. JSON keys are omitted.
func (p Plan) Human() string {
	var b strings.Builder
	sum := strings.TrimSpace(p.Summary)
	if sum == "" {
		sum = strings.TrimSpace(p.ID)
	}
	if sum == "" {
		sum = "(empty plan)"
	}
	fmt.Fprintf(&b, "plan  %s\n", sum)
	if len(p.Steps) == 0 {
		b.WriteString("\nno steps\n")
		return redact.String(b.String())
	}
	b.WriteByte('\n')
	for i, s := range p.Steps {
		title := strings.TrimSpace(s.Action)
		if title == "" {
			title = strings.TrimSpace(s.ID)
		}
		if title == "" {
			title = "step"
		}
		if s.Privileged {
			fmt.Fprintf(&b, "%d. %s  (privileged)\n", i+1, title)
		} else {
			fmt.Fprintf(&b, "%d. %s\n", i+1, title)
		}
		if len(s.Argv) > 0 {
			fmt.Fprintf(&b, "   %s\n", strings.Join(s.Argv, " "))
		}
		if i+1 < len(p.Steps) {
			b.WriteByte('\n')
		}
	}
	return redact.String(b.String())
}
