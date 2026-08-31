// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package cli

import (
	"testing"

	"github.com/revytechinc/hawkeye/internal/config"
	"github.com/revytechinc/hawkeye/internal/knowledge"
	"github.com/revytechinc/hawkeye/internal/probe"
)

func TestAttachSearch_NilStore(t *testing.T) {
	attachSearch(nil, config.Default(), probe.Snapshot{})
}

func TestAttachSearch_NoEmbedModelIsFTSOnly(t *testing.T) {
	st := &knowledge.Store{}
	cfg := config.Default()
	attachSearch(st, cfg, probe.Snapshot{GPUPresent: false})
	if st.Embedder != nil {
		t.Fatal("empty embed_model_path must leave FTS-only")
	}
	if st.RAMMin == nil || *st.RAMMin != *cfg.Resources.RAMMinFreeBytes {
		t.Fatal("consumption headroom must still be attached")
	}
}

func TestAttachSearch_ConfiguredEmbedder(t *testing.T) {
	st := &knowledge.Store{}
	cfg := config.Default()
	cfg.LLM.Local.EmbedModelPath = "/models/fake-embed.gguf"
	cfg.LLM.Local.Bin = "/usr/local/bin/llama-cli"
	attachSearch(st, cfg, probe.Snapshot{GPUPresent: true})
	if st.Embedder == nil {
		t.Fatal("configured embed model must attach a local embedder")
	}
	if st.Embedder.Model() != "/models/fake-embed.gguf" {
		t.Fatalf("model %q", st.Embedder.Model())
	}
}
