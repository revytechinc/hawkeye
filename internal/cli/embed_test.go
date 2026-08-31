// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package cli_test

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/revytechinc/hawkeye/internal/cli"
	"github.com/revytechinc/hawkeye/internal/knowledge"
	"github.com/revytechinc/hawkeye/internal/probe"
	_ "modernc.org/sqlite"
)

func countEmbeddings(t *testing.T, path string) int {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var n int
	err = db.QueryRow(`SELECT COUNT(*) FROM embeddings WHERE vector IS NOT NULL`).Scan(&n)
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return 0
		}
		t.Fatal(err)
	}
	return n
}

func runEmbed(t *testing.T, args []string, dest string, fake *knowledge.FakeEmbedder, host probe.Host, env map[string]string) (int, string, string) {
	t.Helper()
	if env == nil {
		env = map[string]string{}
	}
	var out, errb bytes.Buffer
	e := cli.Env{
		Args:   append([]string{"hawkeye"}, args...),
		Stdin:  bytes.NewReader(nil),
		Stdout: &out,
		Stderr: &errb,
		Getenv: func(k string) string { return env[k] },
		Host:   host,
	}
	if fake != nil {
		e.Embedder = fake
	}
	code := cli.RunEnv(e)
	return code, out.String(), errb.String()
}

func TestEmbed_DefaultDryRunDoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "knowledge.sqlite")
	if err := knowledge.CreatePlaybookTestDB(dest); err != nil {
		t.Fatal(err)
	}
	fake := &knowledge.FakeEmbedder{Name: "fake-test", Default: []float32{1, 0, 0}}
	code, out, err := runEmbed(t, []string{"embed", "--dest", dest}, dest, fake, fakeHost{usr: true, varp: true}, nil)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if !strings.Contains(strings.ToLower(out), "dry-run") {
		t.Fatalf("default embed must say dry-run:\n%s", out)
	}
	if strings.Contains(out, "fake-password-for-tests-only") {
		t.Fatal("dry-run leaked a secret")
	}
	if n := countEmbeddings(t, dest); n != 0 {
		t.Fatalf("dry-run must not fill embeddings: %d", n)
	}
	if fake.LastText != "" {
		t.Fatal("dry-run must not invoke the embedder")
	}
}

func TestEmbed_YesRestPathFills(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "knowledge.sqlite")
	if err := knowledge.CreatePlaybookTestDB(dest); err != nil {
		t.Fatal(err)
	}
	fake := &knowledge.FakeEmbedder{Name: "fake-test", Default: []float32{1, 0, 0}}
	code, out, err := runEmbed(t, []string{"embed", "--yes", dest}, dest, fake, fakeHost{usr: true, varp: true}, nil)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if n := countEmbeddings(t, dest); n < 1 {
		t.Fatalf("rest dest must fill: %d\n%s", n, out)
	}
}

func TestEmbed_YesFillsWritableKit(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "knowledge.sqlite")
	if err := knowledge.CreatePlaybookTestDB(dest); err != nil {
		t.Fatal(err)
	}
	fake := &knowledge.FakeEmbedder{Name: "fake-test", Default: []float32{1, 0, 0}}
	code, out, err := runEmbed(t, []string{"embed", "--yes", "--dest", dest}, dest, fake, fakeHost{usr: true, varp: true}, nil)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if strings.Contains(strings.ToLower(out), "dry-run") {
		t.Fatalf("--yes must write, not dry-run:\n%s", out)
	}
	if n := countEmbeddings(t, dest); n < 1 {
		t.Fatalf("--yes must fill embeddings, got %d\n%s", n, out)
	}
}

func TestEmbed_RefusesReadOnlyStore(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("chmod 0444 is still writable as root")
	}
	dir := t.TempDir()
	dest := filepath.Join(dir, "knowledge.sqlite")
	if err := knowledge.CreatePlaybookTestDB(dest); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dest, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dest, 0o644) })
	fake := &knowledge.FakeEmbedder{Name: "fake-test", Default: []float32{1, 0, 0}}
	code, out, err := runEmbed(t, []string{"embed", "--yes", "--dest", dest}, dest, fake, fakeHost{usr: true, varp: true}, nil)
	if code == 0 {
		t.Fatalf("RO kit must be refused: %s %s", out, err)
	}
	if !strings.Contains(strings.ToLower(out+err), "read-only") {
		t.Fatalf("must say read-only: out=%s err=%s", out, err)
	}
	if n := countEmbeddings(t, dest); n != 0 {
		t.Fatalf("RO refuse must not write: %d", n)
	}
}

func TestEmbed_RedactsSecrets(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "knowledge.sqlite")
	if err := knowledge.CreatePlaybookTestDB(dest); err != nil {
		t.Fatal(err)
	}
	fake := &knowledge.FakeEmbedder{
		Name:    "fake-test",
		Default: []float32{1, 0, 0},
	}
	code, out, err := runEmbed(t, []string{"embed", "--yes", "--dest", dest}, dest, fake, fakeHost{usr: true, varp: true}, map[string]string{
		"HAWKEYE_LLM_API_KEY": "fake-password-for-tests-only",
	})
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	joined := out + err
	if strings.Contains(joined, "fake-password-for-tests-only") {
		t.Fatal("embed must never print secrets")
	}
}

func TestEmbed_KnowledgePathDir(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "knowledge.sqlite")
	if err := knowledge.CreatePlaybookTestDB(dest); err != nil {
		t.Fatal(err)
	}
	fake := &knowledge.FakeEmbedder{Name: "fake-test", Default: []float32{1, 0, 0}}
	code, out, err := runEmbed(t, []string{"embed"}, dest, fake, fakeHost{usr: true, varp: true}, map[string]string{
		"HAWKEYE_KNOWLEDGE_PATH": dir,
	})
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if !strings.Contains(strings.ToLower(out), "dry-run") {
		t.Fatalf("dir knowledge path must dry-run: %s", out)
	}
	if n := countEmbeddings(t, dest); n != 0 {
		t.Fatalf("must not write: %d", n)
	}
}

func TestEmbed_DestDirectoryWithoutKit(t *testing.T) {
	fake := &knowledge.FakeEmbedder{Name: "fake-test", Default: []float32{1, 0, 0}}
	code, out, err := runEmbed(t, []string{"embed", "--dest", t.TempDir()}, "", fake, fakeHost{usr: true, varp: true}, nil)
	if code == 0 {
		t.Fatalf("dir without knowledge.sqlite must fail: %s %s", out, err)
	}
}

func TestEmbed_DestFileMissing(t *testing.T) {
	fake := &knowledge.FakeEmbedder{Name: "fake-test", Default: []float32{1, 0, 0}}
	code, out, err := runEmbed(t, []string{"embed", "--yes", "--dest", filepath.Join(t.TempDir(), "nope.sqlite")}, "", fake, fakeHost{usr: true, varp: true}, nil)
	if code == 0 {
		t.Fatalf("missing kit must fail: %s %s", out, err)
	}
}

func TestEmbed_YesEmbedderError(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "knowledge.sqlite")
	if err := knowledge.CreatePlaybookTestDB(dest); err != nil {
		t.Fatal(err)
	}
	fake := &knowledge.FakeEmbedder{Name: "fake-test", Default: []float32{1, 0, 0}, Err: context.Canceled}
	code, out, err := runEmbed(t, []string{"embed", "--yes", "--dest", dest}, dest, fake, fakeHost{usr: true, varp: true}, nil)
	if code == 0 {
		t.Fatalf("embedder error must fail: %s %s", out, err)
	}
	if n := countEmbeddings(t, dest); n != 0 {
		t.Fatalf("failed fill must not leave rows: %d", n)
	}
}

func TestEmbed_DestIsDirectoryNamedKit(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "knowledge.sqlite")
	if err := os.Mkdir(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	fake := &knowledge.FakeEmbedder{Name: "fake-test", Default: []float32{1, 0, 0}}
	code, out, err := runEmbed(t, []string{"embed", "--dest", dest}, dest, fake, fakeHost{usr: true, varp: true}, nil)
	if code == 0 {
		t.Fatalf("directory dest must fail: %s %s", out, err)
	}
}

func TestEmbed_YesWithoutEmbedder(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "knowledge.sqlite")
	if err := knowledge.CreatePlaybookTestDB(dest); err != nil {
		t.Fatal(err)
	}
	code, out, err := runEmbed(t, []string{"embed", "--yes", "--dest", dest}, dest, nil, fakeHost{usr: true, varp: true}, nil)
	if code == 0 {
		t.Fatalf("no embedder must fail: %s %s", out, err)
	}
	if !strings.Contains(strings.ToLower(out+err), "embedder") {
		t.Fatalf("want embedder message: out=%s err=%s", out, err)
	}
}

func TestEmbed_DryRunWinsOverYes(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "knowledge.sqlite")
	if err := knowledge.CreatePlaybookTestDB(dest); err != nil {
		t.Fatal(err)
	}
	fake := &knowledge.FakeEmbedder{Name: "fake-test", Default: []float32{1, 0, 0}}
	code, out, err := runEmbed(t, []string{"embed", "--yes", "--dry-run", "--dest", dest}, dest, fake, fakeHost{usr: true, varp: true}, nil)
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if n := countEmbeddings(t, dest); n != 0 {
		t.Fatalf("--dry-run must win over --yes: %d", n)
	}
}
