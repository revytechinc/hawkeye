// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/revytechinc/hawkeye/internal/apply"
	"github.com/revytechinc/hawkeye/internal/audit"
	"github.com/revytechinc/hawkeye/internal/config"
	"github.com/revytechinc/hawkeye/internal/consult"
	"github.com/revytechinc/hawkeye/internal/doctor"
	"github.com/revytechinc/hawkeye/internal/headroom"
	"github.com/revytechinc/hawkeye/internal/knowledge"
	"github.com/revytechinc/hawkeye/internal/llm"
	"github.com/revytechinc/hawkeye/internal/mcp"
	"github.com/revytechinc/hawkeye/internal/pidfile"
	"github.com/revytechinc/hawkeye/internal/probe"
	"github.com/revytechinc/hawkeye/internal/redact"
	"github.com/revytechinc/hawkeye/internal/update"
	"github.com/revytechinc/hawkeye/internal/version"
)

type Env struct {
	Args   []string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	Getenv func(string) string
	Host   probe.Host
}

func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	return RunEnv(Env{Args: args, Stdin: stdin, Stdout: stdout, Stderr: stderr, Getenv: os.Getenv, Host: probe.Live()})
}

func RunEnv(env Env) int {
	if env.Getenv == nil {
		env.Getenv = os.Getenv
	}
	if env.Host == nil {
		env.Host = probe.Live()
	}
	fs := parse(env.Args[1:])
	if fs.help {
		fmt.Fprint(env.Stdout, usage())
		return 0
	}
	if fs.version && fs.cmd == "" {
		fmt.Fprintln(env.Stdout, version.Product, version.Number)
		return 0
	}

	cfgPath := fs.config
	if cfgPath == "" {
		cfgPath = env.Getenv("HAWKEYE_CONFIG")
	}

	if fs.checkConfig {
		path := config.ResolvePath(cfgPath)
		if err := config.CheckFile(path); err != nil {
			fmt.Fprintf(env.Stderr, "hawkeye: --check-config failed: %v\n", err)
			return 1
		}
		fmt.Fprintln(env.Stdout, "configuration ok:", path)
		return 0
	}

	cfg := config.Default()
	if p := config.ResolvePath(cfgPath); p != "" {
		if c, err := config.Load(p); err == nil {
			cfg = c
		} else if !os.IsNotExist(err) && fs.cmd != "init" {
			if _, statErr := os.Stat(p); statErr == nil {
				fmt.Fprintf(env.Stderr, "hawkeye: config: %v\n", err)
				return 1
			}
		}
	}

	switch fs.cmd {
	case "", "help":
		fmt.Fprint(env.Stdout, usage())
		return 0
	case "version":
		fmt.Fprintln(env.Stdout, version.Product, version.Number)
		return 0
	case "init":
		return cmdInit(env, fs, cfg)
	case "consult":
		return cmdConsult(env, fs, cfg)
	case "plan":
		return cmdPlan(env, fs, cfg)
	case "apply":
		return cmdApply(env, fs, cfg)
	case "doctor":
		return cmdDoctor(env, fs, cfg)
	case "mcp":
		return cmdMCP(env, fs, cfg)
	case "update":
		return cmdUpdate(env, fs, cfg)
	default:
		fmt.Fprintf(env.Stderr, "hawkeye: unknown command %q\n", fs.cmd)
		return 2
	}
}

type flagset struct {
	config      string
	checkConfig bool
	json        bool
	help        bool
	version     bool
	dryRun      bool
	yes         bool
	http        bool
	stdio       bool
	src         string
	dest        string
	cmd         string
	rest        []string
}

func parse(args []string) flagset {
	var fs flagset
	var rest []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			rest = append(rest, args[i+1:]...)
			i = len(args)
		case a == "--help" || a == "-h":
			fs.help = true
		case a == "--version" || a == "-V":
			fs.version = true
		case a == "--check-config":
			fs.checkConfig = true
		case a == "--json":
			fs.json = true
		case a == "--dry-run":
			fs.dryRun = true
		case a == "--yes" || a == "-y":
			fs.yes = true
		case a == "--http":
			fs.http = true
		case a == "--stdio":
			fs.stdio = true
		case a == "--config" || a == "-c":
			i++
			if i < len(args) {
				fs.config = args[i]
			}
		case strings.HasPrefix(a, "--config="):
			fs.config = strings.TrimPrefix(a, "--config=")
		case a == "--src":
			i++
			if i < len(args) {
				fs.src = args[i]
			}
		case strings.HasPrefix(a, "--src="):
			fs.src = strings.TrimPrefix(a, "--src=")
		case a == "--dest":
			i++
			if i < len(args) {
				fs.dest = args[i]
			}
		case strings.HasPrefix(a, "--dest="):
			fs.dest = strings.TrimPrefix(a, "--dest=")
		case strings.HasPrefix(a, "-"):
			rest = append(rest, a)
		default:
			rest = append(rest, a)
		}
	}
	if len(rest) > 0 {
		fs.cmd = rest[0]
		fs.rest = rest[1:]
	}
	return fs
}

func usage() string {
	return `hawkeye — FreeBSD diagnostic/ops doctor (MASH)

Usage:
  hawkeye [--config PATH] [--check-config] [--json] <command> [args]

Commands:
  consult [query]     Diagnose using knowledge FTS and optional LLM (no writes)
  plan [query]        Emit a JSON plan; no mutation
  apply [--dry-run|--yes] [plan.json]
                      Mutate. DEFAULT is dry-run. LLM never execs as root.
  doctor              Service health (config, perms, pidfile, deps, headroom)
  mcp [--stdio|--http]
                      MCP server (stdio default; HTTP is Streamable HTTP over TLS on 127.0.0.1)
  update              Refresh knowledge from hawkeye-data artifacts when writable
  init                Write a sample JSON config
  version             Print version

This repository ships binaries only. Knowledge lives in hawkeye-data.
Never a public chat UI.
`
}

func cmdInit(env Env, fs flagset, _ config.Config) int {
	b, err := config.InitJSON()
	if err != nil {
		fmt.Fprintln(env.Stderr, err)
		return 1
	}
	if len(fs.rest) > 0 && fs.rest[0] == "-" {
		_, _ = env.Stdout.Write(b)
		return 0
	}
	dir := config.UserDir()
	if len(fs.rest) > 0 {
		dir = fs.rest[0]
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintln(env.Stderr, err)
		return 1
	}
	p := filepath.Join(dir, "config.json")
	if err := os.WriteFile(p, b, 0o600); err != nil {
		fmt.Fprintln(env.Stderr, err)
		return 1
	}
	fmt.Fprintln(env.Stdout, p)
	return 0
}

func knowledgePaths(env Env, cfg config.Config) []string {
	var paths []string
	if extra := env.Getenv("HAWKEYE_KNOWLEDGE_PATH"); extra != "" {
		paths = append(paths, extra)
	}
	paths = append(paths, cfg.Knowledge.Paths...)
	xdg := env.Getenv("XDG_DATA_HOME")
	home, _ := os.UserHomeDir()
	paths = append(paths, knowledge.SearchPaths(xdg, home)...)
	return paths
}

func openKnowledge(env Env, cfg config.Config, snap probe.Snapshot) *knowledge.Store {
	st, err := knowledge.Open(knowledgePaths(env, cfg), snap.RootRO)
	if err != nil {
		return nil
	}
	return st
}

func cmdConsult(env Env, fs flagset, cfg config.Config) int {
	q := strings.Join(fs.rest, " ")
	if q == "" {
		b, _ := io.ReadAll(env.Stdin)
		q = strings.TrimSpace(string(b))
	}
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
	b, err := res.JSON()
	if err != nil {
		fmt.Fprintln(env.Stderr, err)
		return 1
	}
	_, _ = env.Stdout.Write(append(b, '\n'))
	return 0
}

func makePlan(query string, snap probe.Snapshot) apply.Plan {
	query = redact.String(query)
	p := apply.Plan{ID: "consult-plan", Source: "knowledge", Summary: query}
	if snap.RootRO {
		p.Steps = []apply.Step{{
			ID:         "1",
			Action:     "unlock-rw",
			Argv:       []string{"zfs", "set", "readonly=off", "<rootpool>"},
			Privileged: true,
		}}
		p.Summary = "root is read-only; first skill is unlock-rw, not pkg"
		return p
	}
	p.Steps = []apply.Step{{
		ID:     "1",
		Action: "diagnose",
		Argv:   []string{"echo", query},
	}}
	return p
}

func cmdPlan(env Env, fs flagset, cfg config.Config) int {
	_ = cfg
	q := strings.Join(fs.rest, " ")
	if q == "" {
		b, _ := io.ReadAll(env.Stdin)
		q = strings.TrimSpace(string(b))
	}
	snap := probe.Probe(env.Host)
	p := makePlan(q, snap)
	enc := json.NewEncoder(env.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(p); err != nil {
		fmt.Fprintln(env.Stderr, err)
		return 1
	}
	return 0
}

func cmdApply(env Env, fs flagset, cfg config.Config) int {
	mode := apply.ResolveMode(fs.dryRun, fs.yes)
	var p apply.Plan
	var raw []byte
	var err error
	if len(fs.rest) > 0 {
		raw, err = os.ReadFile(fs.rest[0])
	} else {
		raw, err = io.ReadAll(env.Stdin)
	}
	if err != nil {
		fmt.Fprintln(env.Stderr, err)
		return 1
	}
	raw = []byte(redact.String(string(raw)))
	if len(strings.TrimSpace(string(raw))) == 0 {
		fmt.Fprintln(env.Stderr, "hawkeye apply: empty plan")
		return 1
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		fmt.Fprintln(env.Stderr, "hawkeye apply: plan JSON:", err)
		return 1
	}
	auditor := apply.Auditor(apply.NopAuditor{})
	if cfg.AuditLog != "" {
		dir := filepath.Dir(cfg.AuditLog)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			if mode == apply.ModeApply {
				fmt.Fprintf(env.Stderr, "hawkeye apply: audit log %s: %v\n", cfg.AuditLog, err)
				return 1
			}
		} else {
			auditor = &audit.File{Path: cfg.AuditLog}
		}
	}
	res, err := apply.Execute(p, mode, apply.ActorOperator, apply.SysExecutor{}, auditor)
	if err != nil {
		fmt.Fprintln(env.Stderr, err)
		return 1
	}
	enc := json.NewEncoder(env.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(res); err != nil {
		return 1
	}
	if mode == apply.ModeApply && !res.DryRun {
		return 0
	}
	return 0
}

func cmdDoctor(env Env, fs flagset, cfg config.Config) int {
	snap := probe.Probe(env.Host)
	hr := headroom.Live(snap.GPUPresent)
	st := openKnowledge(env, cfg, snap)
	detail := "knowledge store missing"
	ok := false
	if st != nil {
		ok = st.FTS
		detail = "open " + st.DSN
		_ = st.Close()
	}
	pidRunning := false
	pidContent := ""
	pidOK := true
	if _, err := os.Stat(cfg.PidFile); err == nil {
		pidRunning = true
		b, err := os.ReadFile(cfg.PidFile)
		if err == nil {
			pidContent = string(b)
		}
		if _, err := pidfile.Read(cfg.PidFile); err != nil {
			pidOK = false
		}
	}
	mode := 0
	cfgPath := config.ResolvePath(fs.config)
	if fi, err := os.Stat(cfgPath); err == nil {
		mode = int(fi.Mode().Perm())
	}
	rep := doctor.Run(doctor.Deps{
		ConfigPath:      cfgPath,
		Cfg:             cfg,
		Probe:           snap,
		Headroom:        hr,
		PidRunning:      pidRunning,
		PidContent:      pidContent,
		PidOwnerOK:      pidOK,
		ConfigMode:      mode,
		KnowledgeOK:     ok,
		KnowledgeDetail: detail,
	})
	if fs.json {
		b, err := rep.JSON()
		if err != nil {
			fmt.Fprintln(env.Stderr, err)
			return 1
		}
		_, _ = env.Stdout.Write(append(b, '\n'))
	} else {
		fmt.Fprint(env.Stdout, rep.Human())
		b, _ := rep.JSON()
		_, _ = env.Stdout.Write(append([]byte("\n"), append(b, '\n')...))
	}
	if !rep.Healthy {
		return 1
	}
	return 0
}

func cmdMCP(env Env, fs flagset, cfg config.Config) int {
	snap := probe.Probe(env.Host)
	s := mcp.New(mcp.Handlers{
		Consult: func(q string) (any, error) {
			st := openKnowledge(env, cfg, snap)
			if st != nil {
				defer st.Close()
			}
			return consult.Run(q, snap, st, llm.None{})
		},
		Plan: func(q string) (any, error) {
			return makePlan(q, snap), nil
		},
		Apply: func(p apply.Plan, yes bool) (any, error) {
			mode := apply.ResolveMode(!yes, yes)
			return apply.Execute(p, mode, apply.ActorMCP, &apply.CountingExecutor{}, apply.NopAuditor{})
		},
		Doctor: func() (any, error) {
			// reuse cmdDoctor logic without printing
			st := openKnowledge(env, cfg, snap)
			ok := st != nil
			detail := ""
			if st != nil {
				detail = st.DSN
				_ = st.Close()
			}
			rep := doctor.Run(doctor.Deps{Cfg: cfg, Probe: snap, Headroom: headroom.Live(snap.GPUPresent), KnowledgeOK: ok, KnowledgeDetail: detail})
			return rep, nil
		},
	})
	if fs.http {
		addr := cfg.Listen.MCPHTTP
		if addr == "" {
			addr = mcp.DefaultAddr()
		}
		fmt.Fprintf(env.Stderr, "hawkeye mcp: streamable HTTP on %s (TLS)\n", addr)
		if err := mcp.ListenAndServeTLS(addr, cfg.Listen.TLSCert, cfg.Listen.TLSKey, s); err != nil {
			fmt.Fprintln(env.Stderr, err)
			return 1
		}
		return 0
	}
	return boolToInt(mcp.ServeStdio(env.Stdin, env.Stdout, s) != nil)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func cmdUpdate(env Env, fs flagset, cfg config.Config) int {
	snap := probe.Probe(env.Host)
	src := fs.src
	if src == "" {
		src = env.Getenv("HAWKEYE_DATA_ARTIFACT")
	}
	dest := fs.dest
	if dest == "" {
		if len(cfg.Knowledge.Paths) > 1 {
			dest = filepath.Join(cfg.Knowledge.Paths[1], knowledge.DBName)
		} else if len(cfg.Knowledge.Paths) == 1 {
			dest = filepath.Join(cfg.Knowledge.Paths[0], knowledge.DBName)
		} else {
			dest = filepath.Join("/usr/local/share/hawkeye", knowledge.DBName)
		}
	}
	got, err := update.Run(src, dest, snap)
	if err != nil {
		fmt.Fprintln(env.Stderr, err)
		return 1
	}
	fmt.Fprintln(env.Stdout, got)
	return 0
}
