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
	// Sources is the host first-look view. Tests inject FAKE fixtures.
	// Nil/zero with a live Host uses probe.LiveSources in Run().
	Sources probe.Sources
	// TTY is true when stdin is an interactive terminal. Tests set this;
	// Run() detects it from stdin. Non-TTY, --json, and MCP never prompt.
	TTY bool
	// Editor, if set, edits a temp plan file (tests). Production uses VISUAL/EDITOR/vi.
	Editor func(path string) error
	// Exec overrides apply.SysExecutor (tests). Mutation still requires --yes / second y.
	Exec apply.Executor
}

func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	return RunEnv(Env{
		Args:    args,
		Stdin:   stdin,
		Stdout:  stdout,
		Stderr:  stderr,
		Getenv:  os.Getenv,
		Host:    probe.Live(),
		Sources: probe.LiveSources(),
		TTY:     readerIsTTY(stdin),
	})
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
		if _, err := os.Stat(path); os.IsNotExist(err) {
			fmt.Fprintln(env.Stdout, "configuration ok: defaults (no file at "+path+")")
			return 0
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
	cfg = config.ApplyEnv(cfg, env.Getenv)

	switch fs.cmd {
	case "":
		return cmdSession(env, fs, cfg)
	case "help":
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
	case "inspect":
		return cmdInspect(env, fs)
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
		if isKnownCommand(rest[0]) {
			fs.cmd = rest[0]
			fs.rest = rest[1:]
		} else {
			fs.rest = rest
		}
	}
	return fs
}

func usage() string {
	return `hawkeye — FreeBSD diagnostic/ops doctor (MASH)

Usage:
  hawkeye                      Panic-path session on a terminal (host first-look, then >)
  hawkeye [query]              Consult, then stay in the session on a TTY
  hawkeye [--config PATH] [--check-config] [--json] <command> [args]

Commands:
  inspect             Host first-look (fstab, rc, zpool, disks, net). Diagnose only.
                      Human text; --json for the machine object. Not doctor.
  consult [query]     Diagnose using knowledge FTS, optional sqlite-vec, optional LLM.
                      Human session on stdout; --json or HAWKEYE_JSON=1 for scripts.
                      TTY: Apply these steps? [y/N/e]. Default N.
                      Landing still needs --yes or a second y. No prompt
                      on --json, non-TTY, or MCP.
  plan [query]        Propose steps; no mutation.
                      Human session on stdout; --json or HAWKEYE_JSON=1 for apply.
                      Steps are the lead playbook's stored commands, not a stub.
  apply [--dry-run|--yes] [plan.json]
                      Mutate. DEFAULT is dry-run. LLM never execs as root.
  doctor              Service health (config, perms, pidfile, deps, headroom)
  mcp [--stdio|--http]
                      MCP server (stdio default; HTTP is Streamable HTTP on 127.0.0.1, bearer token required)
  update              Refresh knowledge from hawkeye-data artifacts when writable
  init                Write a sample JSON config
  version             Print version

This repository ships binaries only. Knowledge lives in hawkeye-data.
Public MCP URL is https://hawkeye.revytechinc.com/mcp (token required). Not a public chat UI.
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
	if extra := env.Getenv("HAWKEYE_KNOWLEDGE_PATH"); extra != "" {
		return []string{extra}
	}
	var paths []string
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
	attachSearch(st, cfg, snap)
	return st
}

func attachSearch(st *knowledge.Store, cfg config.Config, snap probe.Snapshot) {
	if st == nil {
		return
	}
	hr := headroom.Live(snap.GPUPresent)
	st.Headroom = hr
	st.RAMMin = cfg.Resources.RAMMinFreeBytes
	if strings.TrimSpace(cfg.LLM.Local.EmbedModelPath) == "" {
		return
	}
	st.Embedder = llm.Local{
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

func cmdConsult(env Env, fs flagset, cfg config.Config) int {
	q := strings.Join(fs.rest, " ")
	if q == "" {
		b, _ := io.ReadAll(env.Stdin)
		q = strings.TrimSpace(string(b))
	}
	return runConsultQuery(env, fs, cfg, q, nil)
}

func wantJSON(fs flagset, getenv func(string) string) bool {
	if fs.json {
		return true
	}
	if getenv == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(getenv("HAWKEYE_JSON"))) {
	case "1", "true", "yes":
		return true
	}
	return false
}

func makePlan(query string, snap probe.Snapshot, st *knowledge.Store) apply.Plan {
	res, err := consult.Run(query, snap, st, llm.None{})
	if err != nil {
		return consult.Result{Query: redact.String(query)}.Plan(snap)
	}
	return res.Plan(snap)
}

func cmdPlan(env Env, fs flagset, cfg config.Config) int {
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
	p := makePlan(q, snap, st)
	if wantJSON(fs, env.Getenv) {
		enc := json.NewEncoder(env.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(p); err != nil {
			fmt.Fprintln(env.Stderr, err)
			return 1
		}
		return 0
	}
	fmt.Fprint(env.Stdout, p.Human())
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
	if len(strings.TrimSpace(string(raw))) == 0 {
		fmt.Fprintln(env.Stderr, "hawkeye apply: empty plan")
		return 1
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		fmt.Fprintln(env.Stderr, "hawkeye apply: plan JSON:", err)
		return 1
	}
	p = redactPlan(p)
	res, err := executePlan(env, cfg, p, mode)
	if err != nil && len(res.Steps) == 0 {
		fmt.Fprintln(env.Stderr, err)
		return 1
	}
	enc := json.NewEncoder(env.Stdout)
	enc.SetIndent("", "  ")
	if encErr := enc.Encode(res); encErr != nil {
		return 1
	}
	if err != nil {
		fmt.Fprintln(env.Stderr, err)
		return 1
	}
	if mode == apply.ModeApply && !res.DryRun && !res.Applied {
		return 1
	}
	return 0
}

func hostInspect(env Env) probe.Report {
	return probe.Inspect(env.Host, env.Sources)
}

func writeFirstLook(env Env) {
	text := hostInspect(env).Human()
	if text == "" {
		return
	}
	fmt.Fprint(env.Stdout, text)
	if !strings.HasSuffix(text, "\n") {
		fmt.Fprintln(env.Stdout)
	}
}

func cmdInspect(env Env, fs flagset) int {
	rep := hostInspect(env)
	if wantJSON(fs, env.Getenv) {
		b, err := rep.JSON()
		if err != nil {
			fmt.Fprintln(env.Stderr, err)
			return 1
		}
		_, _ = env.Stdout.Write(append(b, '\n'))
		return 0
	}
	fmt.Fprint(env.Stdout, rep.Human())
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
	pidReadErr := ""
	pidOK := true
	pidMode := 0
	if fi, err := os.Stat(cfg.PidFile); err == nil {
		pidRunning = true
		pidMode = int(fi.Mode().Perm())
		b, err := os.ReadFile(cfg.PidFile)
		if err != nil {
			pidReadErr = "pidfile is unreadable: " + err.Error()
			pidOK = false
		} else {
			pidContent = string(b)
			if _, err := pidfile.Read(cfg.PidFile); err != nil {
				pidOK = false
			}
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
		PidReadErr:      pidReadErr,
		PidOwnerOK:      pidOK,
		PidMode:         pidMode,
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
			st := openKnowledge(env, cfg, snap)
			if st != nil {
				defer st.Close()
			}
			return makePlan(q, snap, st), nil
		},
		Apply: func(p apply.Plan, yes bool) (any, error) {
			return mcpApply(env, cfg, p, yes)
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
		Inspect: func() (any, error) {
			return hostInspect(env), nil
		},
	})
	if fs.http {
		addr := cfg.Listen.MCPHTTP
		if addr == "" {
			addr = mcp.DefaultAddr()
		}
		token, err := mcp.ResolveToken(env.Getenv, cfg.Listen.MCPTokenEnv)
		if err != nil {
			fmt.Fprintln(env.Stderr, "hawkeye mcp:", err)
			return 1
		}
		s.Token = token
		fmt.Fprintf(env.Stderr, "hawkeye mcp: streamable HTTP on %s (loopback; bearer token required)\n", addr)
		if err := mcp.ListenAndServe(addr, cfg.Listen.TLSCert, cfg.Listen.TLSKey, token, s); err != nil {
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
		src = cfg.Update.Source
	}
	dest := fs.dest
	if dest == "" {
		dest = cfg.Update.Dest
	}
	if dest == "" {
		dest = update.DefaultDest
	}
	got, err := update.Run(src, dest, snap)
	if err != nil {
		fmt.Fprintln(env.Stderr, err)
		return 1
	}
	if got != "" {
		fmt.Fprintln(env.Stdout, got)
	}
	return 0
}
