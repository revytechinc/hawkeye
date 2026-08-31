// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package llm_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/revytechinc/hawkeye/internal/knowledge"
	"github.com/revytechinc/hawkeye/internal/llm"
)

func writeGGUF(t *testing.T, dir, name string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("not-a-real-gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestDiscoverModel_EmptyDirsSkip(t *testing.T) {
	if got := llm.DiscoverModel(nil, false); got != "" {
		t.Fatalf("empty dirs must skip: %q", got)
	}
	if got := llm.DiscoverModel([]string{t.TempDir()}, false); got != "" {
		t.Fatalf("empty models dir must skip: %q", got)
	}
	if got := llm.DiscoverModel([]string{""}, false); got != "" {
		t.Fatalf("empty path must skip: %q", got)
	}
	missing := filepath.Join(t.TempDir(), "nope")
	if got := llm.DiscoverModel([]string{missing}, false); got != "" {
		t.Fatalf("missing dir must skip: %q", got)
	}
}

func TestDiscoverModel_WellKnownModelsDir(t *testing.T) {
	root := t.TempDir()
	models := filepath.Join(root, "models")
	want := writeGGUF(t, models, "tiny-chat.gguf")
	writeGGUF(t, models, "nomic-embed.gguf")
	got := llm.DiscoverModel([]string{models}, false)
	if got != want {
		t.Fatalf("chat discover = %q want %q (prefer non-embed)", got, want)
	}
}

func TestDiscoverModel_OnlyEmbedNamedStillFindsChat(t *testing.T) {
	dir := t.TempDir()
	want := writeGGUF(t, dir, "nomic-embed-text.gguf")
	got := llm.DiscoverModel([]string{dir}, false)
	if got != want {
		t.Fatalf("sole GGUF must be the obvious drop: %q want %q", got, want)
	}
}

func TestDiscoverEmbedModel_PrefersEmbedName(t *testing.T) {
	dir := t.TempDir()
	writeGGUF(t, dir, "tiny-chat.gguf")
	want := writeGGUF(t, dir, "nomic-embed.gguf")
	got := llm.DiscoverModel([]string{dir}, true)
	if got != want {
		t.Fatalf("embed discover = %q want %q", got, want)
	}
}

func TestDiscoverEmbedModel_NoEmbedNameSkipsChat(t *testing.T) {
	dir := t.TempDir()
	writeGGUF(t, dir, "tiny-chat.gguf")
	if got := llm.DiscoverModel([]string{dir}, true); got != "" {
		t.Fatalf("chat GGUF is not an embedder: %q", got)
	}
}

func TestResolveModel_ExplicitFileWins(t *testing.T) {
	dir := t.TempDir()
	writeGGUF(t, dir, "other.gguf")
	explicit := writeGGUF(t, dir, "explicit.gguf")
	got := llm.ResolveModel(explicit, []string{dir})
	if got != explicit {
		t.Fatalf("explicit file must win: %q", got)
	}
}

func TestResolveModel_ExplicitDirScans(t *testing.T) {
	dir := t.TempDir()
	want := writeGGUF(t, dir, "dropped.gguf")
	got := llm.ResolveModel(dir, nil)
	if got != want {
		t.Fatalf("model_path directory must scan: %q want %q", got, want)
	}
}

func TestResolveModel_EmptyDiscovers(t *testing.T) {
	dir := t.TempDir()
	want := writeGGUF(t, dir, "dropped.gguf")
	got := llm.ResolveModel("", []string{dir})
	if got != want {
		t.Fatalf("empty model_path must discover: %q want %q", got, want)
	}
}

func TestResolveModel_MissingExplicitKept(t *testing.T) {
	got := llm.ResolveModel("/no/such/model.gguf", []string{t.TempDir()})
	if got != "/no/such/model.gguf" {
		t.Fatalf("explicit missing path must not be replaced: %q", got)
	}
}

func TestResolveEmbedModel_EmptyDiscoversEmbedName(t *testing.T) {
	dir := t.TempDir()
	writeGGUF(t, dir, "tiny-chat.gguf")
	want := writeGGUF(t, dir, "nomic-embed.gguf")
	got := llm.ResolveEmbedModel("", []string{dir})
	if got != want {
		t.Fatalf("embed resolve = %q want %q", got, want)
	}
}

func TestDefaultModelDirs_MatchKnowledgeLayout(t *testing.T) {
	dirs := llm.DefaultModelDirs("/xdg/share", "/home/operator")
	wantModels := filepath.Join(knowledge.SystemDir, "models")
	wantRescue := filepath.Join(knowledge.RescueDir, "models")
	foundSys, foundRescue := false, false
	for _, d := range dirs {
		if d == wantModels {
			foundSys = true
		}
		if d == wantRescue {
			foundRescue = true
		}
	}
	if !foundSys || !foundRescue {
		t.Fatalf("default model dirs must include well-known GGUF drops: %v", dirs)
	}
}
