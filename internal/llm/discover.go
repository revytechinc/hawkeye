// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package llm

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/revytechinc/hawkeye/internal/knowledge"
)

// DefaultModelDirs are well-known places an operator may drop a GGUF
// without editing JSON. Layout matches knowledge prefixes: /boot/hawkeye
// then /usr/local/share/hawkeye, each with a models/ subdir first.
func DefaultModelDirs(xdgDataHome, home string) []string {
	var out []string
	for _, p := range knowledge.SearchPaths(xdgDataHome, home) {
		out = append(out, filepath.Join(p, "models"), p)
	}
	return out
}

// ResolveModel returns an explicit GGUF path, or scans explicit when it
// is a directory, or discovers the first chat GGUF under dirs.
// A missing explicit file path is kept so the operator's setting wins.
func ResolveModel(explicit string, dirs []string) string {
	return resolveGGUF(explicit, dirs, false)
}

// ResolveEmbedModel is ResolveModel for an embedding GGUF (*embed*).
func ResolveEmbedModel(explicit string, dirs []string) string {
	return resolveGGUF(explicit, dirs, true)
}

func resolveGGUF(explicit string, dirs []string, wantEmbed bool) string {
	if p := strings.TrimSpace(explicit); p != "" {
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			if got := firstGGUF(p, wantEmbed); got != "" {
				return got
			}
		}
		return p
	}
	return DiscoverModel(dirs, wantEmbed)
}

// DiscoverModel walks dirs for *.gguf. wantEmbed prefers *embed* names
// and skips a chat-only drop. Chat discovery prefers non-embed names,
// then any sole GGUF (the obvious operator drop).
func DiscoverModel(dirs []string, wantEmbed bool) string {
	for _, d := range dirs {
		if got := firstGGUF(d, wantEmbed); got != "" {
			return got
		}
	}
	return ""
}

func firstGGUF(dir string, wantEmbed bool) string {
	if strings.TrimSpace(dir) == "" {
		return ""
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	var chat, embed []string
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".gguf") {
			continue
		}
		p := filepath.Join(dir, name)
		if looksEmbed(name) {
			embed = append(embed, p)
		} else {
			chat = append(chat, p)
		}
	}
	sort.Strings(chat)
	sort.Strings(embed)
	if wantEmbed {
		if len(embed) > 0 {
			return embed[0]
		}
		return ""
	}
	if len(chat) > 0 {
		return chat[0]
	}
	if len(embed) > 0 {
		return embed[0]
	}
	return ""
}

func looksEmbed(name string) bool {
	return strings.Contains(strings.ToLower(name), "embed")
}
