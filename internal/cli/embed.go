// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/revytechinc/hawkeye/internal/apply"
	"github.com/revytechinc/hawkeye/internal/config"
	"github.com/revytechinc/hawkeye/internal/headroom"
	"github.com/revytechinc/hawkeye/internal/knowledge"
	"github.com/revytechinc/hawkeye/internal/llm"
	"github.com/revytechinc/hawkeye/internal/probe"
	"github.com/revytechinc/hawkeye/internal/redact"
)

func resolveLocalModels(cfg config.Config, env Env) config.Config {
	xdg := ""
	extra := ""
	if env.Getenv != nil {
		xdg = env.Getenv("XDG_DATA_HOME")
		extra = strings.TrimSpace(env.Getenv("HAWKEYE_MODELS_DIR"))
	}
	home, _ := os.UserHomeDir()
	dirs := llm.DefaultModelDirs(xdg, home)
	if extra != "" {
		dirs = append([]string{extra}, dirs...)
	}
	cfg.LLM.Local.ModelPath = llm.ResolveModel(cfg.LLM.Local.ModelPath, dirs)
	cfg.LLM.Local.EmbedModelPath = llm.ResolveEmbedModel(cfg.LLM.Local.EmbedModelPath, dirs)
	return cfg
}

func embedDest(fs flagset, cfg config.Config, env Env) string {
	if strings.TrimSpace(fs.dest) != "" {
		return kitPath(fs.dest)
	}
	if len(fs.rest) > 0 {
		return kitPath(fs.rest[0])
	}
	if env.Getenv != nil {
		if extra := strings.TrimSpace(env.Getenv("HAWKEYE_KNOWLEDGE_PATH")); extra != "" {
			return kitPath(extra)
		}
	}
	if strings.TrimSpace(cfg.Update.Dest) != "" {
		return kitPath(cfg.Update.Dest)
	}
	return ""
}

func kitPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if fi, err := os.Stat(p); err == nil && fi.IsDir() {
		return filepath.Join(p, knowledge.DBName)
	}
	return p
}

func writableKit(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("embed dest is required")
	}
	fi, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("embed dest: %w", err)
	}
	if fi.IsDir() {
		return fmt.Errorf("embed dest must be a knowledge.sqlite file")
	}
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("embeddings fill refused: store is read-only")
		}
		return fmt.Errorf("embeddings fill refused: store is read-only: %w", err)
	}
	_ = f.Close()
	return nil
}

func cmdEmbed(env Env, fs flagset, cfg config.Config) int {
	dest := embedDest(fs, cfg, env)
	if dest == "" {
		fmt.Fprintln(env.Stderr, redact.String("hawkeye embed: dest is required (--dest or a writable knowledge.sqlite)"))
		return 1
	}
	if err := writableKit(dest); err != nil {
		fmt.Fprintln(env.Stderr, redact.String("hawkeye embed: "+err.Error()))
		return 1
	}
	mode := apply.ResolveMode(fs.dryRun, fs.yes)
	if mode == apply.ModeDryRun {
		fmt.Fprintln(env.Stdout, redact.String("dry-run: would embed playbook and document chunks into "+dest+" (pass --yes to write)"))
		return 0
	}
	emb := env.Embedder
	if emb == nil {
		if strings.TrimSpace(cfg.LLM.Local.EmbedModelPath) == "" {
			fmt.Fprintln(env.Stderr, "hawkeye embed: local embedder is not configured")
			return 1
		}
		snap := probe.Probe(env.Host)
		hr := headroom.Live(snap.GPUPresent)
		emb = llm.Local{
			Backend:        cfg.LLM.Local.Backend,
			Bin:            cfg.LLM.Local.Bin,
			EmbedModelPath: cfg.LLM.Local.EmbedModelPath,
			PreferGPU:      cfg.LLM.Local.PreferGPU,
			RequireGPU:     false,
			GPUPresent:     snap.GPUPresent,
			Headroom:       hr,
			RAMMin:         cfg.Resources.RAMMinFreeBytes,
			VRAMMin:        cfg.Resources.GPUVRAMMinFreeBytes,
		}
	}
	rw, err := knowledge.OpenRW(dest)
	if err != nil {
		fmt.Fprintln(env.Stderr, redact.String("hawkeye embed: "+err.Error()))
		return 1
	}
	defer rw.Close()
	rw.Headroom = headroom.Live(probe.Probe(env.Host).GPUPresent)
	rw.RAMMin = cfg.Resources.RAMMinFreeBytes
	if err := rw.FillEmbeddings(context.Background(), emb); err != nil {
		fmt.Fprintln(env.Stderr, redact.String("hawkeye embed: "+err.Error()))
		return 1
	}
	fmt.Fprintln(env.Stdout, redact.String("embedded chunks into "+dest))
	return 0
}
