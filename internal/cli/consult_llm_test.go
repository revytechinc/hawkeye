// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/revytechinc/hawkeye/internal/knowledge"
)

func writeFakeLlamaCLI(t *testing.T, canned string) (bin, model string) {
	t.Helper()
	dir := t.TempDir()
	model = filepath.Join(dir, "fake.gguf")
	if err := os.WriteFile(model, []byte("not-a-real-gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin = filepath.Join(dir, "llama-cli")
	script := "#!/bin/sh\nprintf '%s\\n' '" + canned + "'\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin, model
}

func TestConsult_NoModelQuietTTY(t *testing.T) {
	dir := t.TempDir()
	if err := knowledge.CreatePlaybookTestDB(filepath.Join(dir, "knowledge.sqlite")); err != nil {
		t.Fatal(err)
	}
	code, out, err := run(t, []string{"consult", "ZFS", "root", "is", "read-only", "after", "boot"}, "", fakeHost{usr: true, varp: true}, map[string]string{
		"HAWKEYE_KNOWLEDGE_PATH": dir,
	})
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	for _, junk := range []string{"llm skipped", "skeleton", "local llama.cpp"} {
		if strings.Contains(out, junk) || strings.Contains(err, junk) {
			t.Fatalf("TTY must skip quietly when no model: out=%s err=%s", out, err)
		}
	}
	if !strings.Contains(out, knowledge.RemountPlaybookTitle) {
		t.Fatalf("playbook missing:\n%s", out)
	}
}

func TestConsult_NoModelJSONNotesSkip(t *testing.T) {
	dir := t.TempDir()
	if err := knowledge.CreateTestDB(filepath.Join(dir, "knowledge.sqlite")); err != nil {
		t.Fatal(err)
	}
	code, out, err := run(t, []string{"consult", "--json", "zfs", "readonly"}, "", fakeHost{usr: true, varp: true}, map[string]string{
		"HAWKEYE_KNOWLEDGE_PATH": dir,
	})
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if !json.Valid([]byte(strings.TrimSpace(out))) {
		t.Fatalf("consult --json must be JSON:\n%s", out)
	}
	if !strings.Contains(out, "llm skipped") {
		t.Fatalf("--json may still note the skip:\n%s", out)
	}
}

func TestConsult_LocalCompletionAfterPlaybook(t *testing.T) {
	dir := t.TempDir()
	if err := knowledge.CreatePlaybookTestDB(filepath.Join(dir, "knowledge.sqlite")); err != nil {
		t.Fatal(err)
	}
	canned := "canned local completion: check zpool status"
	bin, model := writeFakeLlamaCLI(t, canned)
	code, out, err := run(t, []string{"consult", "ZFS", "root", "is", "read-only", "after", "boot"}, "", fakeHost{usr: true, varp: true}, map[string]string{
		"HAWKEYE_KNOWLEDGE_PATH": dir,
		"HAWKEYE_LLM_MODEL":      model,
		"HAWKEYE_LLM_BIN":        bin,
	})
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if strings.Contains(out, "llm skipped") || strings.Contains(out, "skeleton") {
		t.Fatalf("TTY leaked guts:\n%s", out)
	}
	if !strings.Contains(out, knowledge.RemountPlaybookTitle) {
		t.Fatalf("playbook missing:\n%s", out)
	}
	if !strings.Contains(out, canned) {
		t.Fatalf("local completion missing after playbook hits:\n%s", out)
	}
	idxPlay := strings.Index(out, knowledge.RemountPlaybookTitle)
	idxLLM := strings.Index(out, canned)
	if idxLLM < idxPlay {
		t.Fatalf("completion must follow playbook:\n%s", out)
	}
}
