// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package cli

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/revytechinc/hawkeye/internal/config"
	"github.com/revytechinc/hawkeye/internal/consult"
	"github.com/revytechinc/hawkeye/internal/headroom"
	"github.com/revytechinc/hawkeye/internal/llm"
	"github.com/revytechinc/hawkeye/internal/probe"
)

const (
	sessionBanner = "hawkeye"
	sessionPrompt = "> "
)

func isKnownCommand(s string) bool {
	switch s {
	case "help", "version", "init", "consult", "plan", "apply", "doctor", "inspect", "mcp", "update":
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

func sessionNeedTTY() string {
	return "hawkeye: run hawkeye on a terminal (type the problem at >, then y/N/e).\n"
}

func sessionHelpText() string {
	return `Type the problem (one line). Then Apply these steps? [y/N/e]
  y  dry-run, then confirm to land (Enter is N; never mutates by accident)
  e  edit the plan in $EDITOR
  n  skip and return to >
quit, exit, q, or Ctrl-D leaves.
`
}

func cmdSession(env Env, fs flagset, cfg config.Config) int {
	q := strings.Join(fs.rest, " ")
	if wantJSON(fs, env.Getenv) {
		if q == "" {
			return cmdInspect(env, fs)
		}
		return runConsultQuery(env, fs, cfg, q, nil)
	}
	if !env.TTY {
		if q == "" {
			fmt.Fprint(env.Stdout, sessionNeedTTY())
			return 0
		}
		return runConsultQuery(env, fs, cfg, q, nil)
	}
	return runREPL(env, fs, cfg, q)
}

func runREPL(env Env, fs flagset, cfg config.Config, first string) int {
	in := bufio.NewReader(env.Stdin)
	fmt.Fprintln(env.Stdout, sessionBanner)
	writeFirstLook(env)
	if first != "" {
		fmt.Fprint(env.Stdout, sessionPrompt)
		fmt.Fprintln(env.Stdout, first)
		_ = runConsultQuery(env, fs, cfg, first, in)
	}
	for {
		fmt.Fprint(env.Stdout, sessionPrompt)
		line, err := readPromptLine(in)
		if err != nil {
			return 0
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if isSessionQuit(line) {
			return 0
		}
		if isSessionHelp(line) {
			fmt.Fprint(env.Stdout, sessionHelpText())
			continue
		}
		_ = runConsultQuery(env, fs, cfg, line, in)
	}
}

func runConsultQuery(env Env, fs flagset, cfg config.Config, q string, in *bufio.Reader) int {
	snap := probe.Probe(env.Host)
	st := openKnowledge(env, cfg, snap)
	if st != nil {
		defer st.Close()
	}
	var comp llm.Completer
	if snap.Tier >= 1 && cfg.LLM.Local.Backend != "" {
		hr := headroom.Live(snap.GPUPresent)
		comp = llm.Local{
			Backend:    cfg.LLM.Local.Backend,
			ModelPath:  cfg.LLM.Local.ModelPath,
			PreferGPU:  cfg.LLM.Local.PreferGPU,
			RequireGPU: cfg.LLM.Local.RequireGPU,
			GPUPresent: snap.GPUPresent,
			Headroom:   hr,
			RAMMin:     cfg.Resources.RAMMinFreeBytes,
			VRAMMin:    cfg.Resources.GPUVRAMMinFreeBytes,
		}
	}
	res, err := consult.Run(q, snap, st, comp)
	if err != nil {
		fmt.Fprintln(env.Stderr, err)
		return 1
	}
	if wantJSON(fs, env.Getenv) {
		b, err := res.JSON()
		if err != nil {
			fmt.Fprintln(env.Stderr, err)
			return 1
		}
		_, _ = env.Stdout.Write(append(b, '\n'))
		return 0
	}
	fmt.Fprint(env.Stdout, res.Human())
	if !env.TTY {
		return 0
	}
	fmt.Fprintln(env.Stdout)
	plan := res.Plan(snap)
	if in == nil {
		return promptConsultApply(env, fs, cfg, plan)
	}
	return promptConsultApplyReader(env, fs, cfg, plan, in)
}
