// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package cli_test

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/revytechinc/hawkeye/internal/knowledge"
)

func TestConsult_DefaultIsHumanNotJSON(t *testing.T) {
	dir := t.TempDir()
	if err := knowledge.CreateTestDB(filepath.Join(dir, "knowledge.sqlite")); err != nil {
		t.Fatal(err)
	}
	code, out, err := run(t, []string{"consult", "ZFS", "root", "is", "read-only", "after", "boot"}, "", fakeHost{ro: true, rescue: true}, map[string]string{"HAWKEYE_KNOWLEDGE_PATH": dir})
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	trim := strings.TrimSpace(out)
	if trim == "" || strings.HasPrefix(trim, "{") {
		t.Fatalf("default consult must be operator prose, not JSON:\n%s", out)
	}
	if !json.Valid([]byte(trim)) {
		// expected: human text is not a JSON document
	} else {
		t.Fatalf("default consult dumped JSON:\n%s", out)
	}
	for _, key := range []string{`"Title"`, `"when_to_use"`, `"query":`, `"hits"`, "llm skipped", "tier "} {
		if strings.Contains(out, key) {
			t.Fatalf("default consult leaked %s:\n%s", key, out)
		}
	}
	if !strings.Contains(out, "ZFS readonly pool") {
		t.Fatalf("hit title missing:\n%s", out)
	}
	if !strings.Contains(out, "unlock-rw") {
		t.Fatalf("playbook text missing:\n%s", out)
	}
}

func TestConsult_JSONFlagDumpsMachineObject(t *testing.T) {
	dir := t.TempDir()
	if err := knowledge.CreateTestDB(filepath.Join(dir, "knowledge.sqlite")); err != nil {
		t.Fatal(err)
	}
	code, out, err := run(t, []string{"consult", "--json", "zfs", "readonly"}, "", fakeHost{ro: true, rescue: true}, map[string]string{"HAWKEYE_KNOWLEDGE_PATH": dir})
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if !json.Valid([]byte(strings.TrimSpace(out))) {
		t.Fatalf("consult --json must be JSON:\n%s", out)
	}
	if !strings.Contains(out, `"query"`) || !strings.Contains(out, `"hits"`) {
		t.Fatalf("consult --json missing machine keys:\n%s", out)
	}
}

func TestConsult_HAWKEYE_JSON(t *testing.T) {
	dir := t.TempDir()
	if err := knowledge.CreateTestDB(filepath.Join(dir, "knowledge.sqlite")); err != nil {
		t.Fatal(err)
	}
	code, out, err := run(t, []string{"consult", "zfs"}, "", fakeHost{ro: true, rescue: true}, map[string]string{
		"HAWKEYE_KNOWLEDGE_PATH": dir,
		"HAWKEYE_JSON":           "1",
	})
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if !json.Valid([]byte(strings.TrimSpace(out))) {
		t.Fatalf("HAWKEYE_JSON=1 must dump JSON:\n%s", out)
	}
}

func TestPlan_DefaultIsHumanNotJSON(t *testing.T) {
	code, out, err := run(t, []string{"plan", "restart", "sshd"}, "", fakeHost{ro: true, rescue: true}, nil)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	trim := strings.TrimSpace(out)
	if json.Valid([]byte(trim)) || strings.HasPrefix(trim, "{") {
		t.Fatalf("default plan must be operator prose, not JSON:\n%s", out)
	}
	if !strings.Contains(out, "unlock-rw") {
		t.Fatalf("plan action missing:\n%s", out)
	}
	if !strings.Contains(out, "zfs set readonly=off") {
		t.Fatalf("plan command missing:\n%s", out)
	}
}

func TestPlan_JSONFlagDumpsMachineObject(t *testing.T) {
	code, out, err := run(t, []string{"plan", "--json", "pkg", "install", "foo"}, "", fakeHost{ro: true, rescue: true}, nil)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if !json.Valid([]byte(strings.TrimSpace(out))) {
		t.Fatalf("plan --json must be JSON:\n%s", out)
	}
	if !strings.Contains(out, `"steps"`) || !strings.Contains(out, "unlock-rw") {
		t.Fatalf("plan --json missing machine object:\n%s", out)
	}
}

func TestPlan_HAWKEYE_JSONYes(t *testing.T) {
	code, out, err := run(t, []string{"plan", "hello"}, "", fakeHost{usr: true, varp: true}, map[string]string{"HAWKEYE_JSON": "yes"})
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if !json.Valid([]byte(strings.TrimSpace(out))) {
		t.Fatalf("HAWKEYE_JSON=yes must dump plan JSON:\n%s", out)
	}
}

func TestConsult_JSONDoesNotBreakDoctorOrApply(t *testing.T) {
	plan := `{"id":"p","source":"operator","steps":[{"id":"1","action":"echo","argv":["echo","hi"],"privileged":false}]}`
	code, out, err := run(t, []string{"apply"}, plan, fakeHost{usr: true, varp: true}, nil)
	if code != 0 {
		t.Fatalf("apply %d %s %s", code, out, err)
	}
	if !strings.Contains(out, `"dry_run"`) {
		t.Fatalf("apply still JSON: %s", out)
	}
	code, out, _ = run(t, []string{"doctor"}, "", fakeHost{usr: true, varp: true}, nil)
	if !strings.Contains(strings.ToLower(out), "doctor") {
		t.Fatalf("doctor human broken: %s", out)
	}
}
