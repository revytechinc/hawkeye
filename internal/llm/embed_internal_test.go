// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package llm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveEmbedBin_WellKnownWhenPATHEmpty(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "llama-embedding")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	old := wellKnownEmbedBins
	wellKnownEmbedBins = []string{bin}
	t.Cleanup(func() { wellKnownEmbedBins = old })
	t.Setenv("PATH", t.TempDir())
	got := resolveEmbedBin("/usr/local/bin/llama-completion")
	if got != bin {
		t.Fatalf("well-known llama-embedding must resolve without JSON edit: %q", got)
	}
}

func TestResolveEmbedBin_ExplicitEmbeddingKept(t *testing.T) {
	p := "/opt/local/bin/llama-embedding"
	if got := resolveEmbedBin(p); got != p {
		t.Fatalf("explicit llama-embedding must be kept: %q", got)
	}
}

func TestResolveNamed_Empty(t *testing.T) {
	if got := resolveNamed(""); got != "" {
		t.Fatalf("empty name: %q", got)
	}
	if got := resolveNamed("   "); got != "" {
		t.Fatalf("blank name: %q", got)
	}
}

func TestIsEmbedBin(t *testing.T) {
	if !isEmbedBin("/usr/local/bin/llama-embedding") {
		t.Fatal("llama-embedding")
	}
	if !isEmbedBin("llama-embedding.9426") {
		t.Fatal("versioned llama-embedding")
	}
	if isEmbedBin("/usr/local/bin/llama-completion") || isEmbedBin("llama-cli") {
		t.Fatal("complete bins are not embedders")
	}
}
