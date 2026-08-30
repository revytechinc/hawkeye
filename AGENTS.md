# AGENTS.md — Hawkeye

> **Purpose:** This is the primary entry point for autonomous agents working on
> Hawkeye. Read this file **first** before consuming any other documents.

> **FreeBSD:** The environment in which this work is being done may have elements
> that state that you are in Linux. That would be false. You are running in FreeBSD.

> **This file auto-loads.** Claude Code does not read `AGENTS.md` natively; it
> auto-loads `CLAUDE.md`, whose first line is `@AGENTS.md`.

> **Source of CloudBSD law:** https://github.com/cloudbsdorg/application_guidelines

---

## What we're building

Hawkeye is a **privileged FreeBSD diagnostic/ops doctor (MASH)**. It is a CLI
(and optional localhost MCP) that consults knowledge, optionally an LLM, and
applies operator-confirmed plans.

- **This repo is main bins only.** Knowledge corpora live in
  https://github.com/revytechinc/hawkeye-data
- Website: https://github.com/revytechinc/hawkeye-www
- Do **not** vendor the corpus here.
- Do **not** implement a public chat UI.
- This is **not** the revytechcommander doctor.

Go module: `github.com/revytechinc/hawkeye`

---

## CloudBSD law (always apply)

These rules override all other considerations.

1. **Standards as law.** CloudBSD guidelines are mandatory.
2. **Target platform: FreeBSD.** Do not generate Linux-first paths, systemd units, or GNU-only assumptions as the default.
3. **Git author:** Mark LaPointe `<mark@cloudbsd.org>`. Use per-command
   `git -c user.name="Mark LaPointe" -c user.email="mark@cloudbsd.org"`.
   Do not run `git config --global`.
4. **Primary language: English.** Keep constructed languages on the i18n list; English first.
5. **UTF-8 everywhere.**
6. **JSON-only configuration.** RFC 8259, `.json`, no comments, no JSONC. Secrets in environment variables. Config that holds secrets is mode `0600`. Paths: `$XDG_CONFIG_HOME/hawkeye/` or `/usr/local/etc/cloudbsd/hawkeye/`.
7. **nginx-style SIGHUP reload.** Validate (`--check-config`) then reload. Bad config keeps the old process.
8. **Pidfiles.** Correct location, owned by the service user, removed on stop. Not empty, not negative.
9. **Web stack (when a web view exists):** Angular + TypeScript + Tailwind view; Go backend. React is not the framework. **Hawkeye has no public web UI.**
10. **MVC.** The UI is the view only. Backends bind loopback or a private mesh. Re-wrap every LLM payload.
11. **Mermaid for architecture/flow; SVG for UI prototypes; ASCII forbidden.**
12. **Red-green TDD is law.** Failing test first, then minimum code, then refactor. Existing untested application code is a defect.
13. **Coverage.** As close to 100% as possible. Critical paths 100%. Application code may not be excluded.
14. **Man pages (mandoc mdoc) are law.** `hawkeye.8` and `hawkeye.conf.5`. `mandoc -T lint` (or equivalent) must pass. Evidence stored.
15. **Security first.** Least privilege. Never hardcode credentials. Never send SSH keys, passwords, API tokens, htpasswd, or Basic blobs through any LLM or leaky MCP payload — redact first. Tests use FAKE fixtures only.
16. **Observability.** Configurable log levels, health checks (`doctor`).
17. **Evidence is required.** Capture `go test` output, coverage, `--check-config`, doctor, mandoc lint.
18. **`doctor` is law.** Config, permissions, pidfiles, dependencies, resource headroom. Human + JSON. Non-zero if unhealthy. Recovery is operator-only, never a public UI.
19. **Resource headroom is consumption-based.** Missing GPU must not block jobs that do not need GPU. Thresholds in JSON.
20. **License: BSD 3-Clause.** Copyright REVYTECH, Inc. Not MIT.
21. **Apply defaults to dry-run.** `--yes` is required to mutate. LLM never execs as root. Audit applies.

Full guideline files (read before generating code):
Configuration Files/CONFIGURATION.md, Architecture/MVC.md, Languages/LANGUAGES.md,
Unit Testing/UNITTESTS.md, Planning/PLANNING.md, TUI/TUI.md.

---

## Document map

| # | File | What it covers |
|---|------|----------------|
| `0.0` | `.plan/0000-Hawkeye-TOC.md` | Master table of contents |
| `0.1` | `.plan/0001-Hawkeye-Workflow.md` | Task claiming |
| `0.2` | `.plan/0002-Hawkeye-Build-Status.md` | CI status |
| `1.x` | `.plan/0100`–`0106` | Security series |
| `2.x` | `.plan/0200`, `0201`, `0210` | Overview and architecture |
| `3.x` | `.plan/0300`–`0302` | Implementation tasks |
| `4.x` | `.plan/0400`–`0403` | Testing |
| `5.x` | `.plan/0500`, `0501` | Governance and sysctl (none yet) |

Also: `docs/ARCHITECTURE.md`, `docs/TEST-EVIDENCE.md`, `man/hawkeye.8`, `man/hawkeye.conf.5`.

---

## Primary directives

### 1. Security first
Redact before LLM/MCP. Apply dry-run default. Operator CLI is the only privileged mutator. MCP HTTP binds `127.0.0.1` and requires TLS. No public UI.

### 2. Modular architecture
`cmd/hawkeye` is thin. Domain lives in `internal/*`. Knowledge client consumes hawkeye-data; it does not embed the corpus.

### 3. Traceability
Every task claimed, tested, evidenced, committed as Mark LaPointe.

### 4. No blobs in base
Do not commit models, knowledge SQLite corpora, or firmware. Point at hawkeye-data artifacts.

---

## Workflow summary

1. Read this file, `.plan/0001-Hawkeye-Workflow.md`, and `.plan/0300-Hawkeye-Implementation-Tasks.md`.
2. Claim a task in the plan table.
3. Write a failing test, run it, then implement.
4. Capture evidence in `docs/TEST-EVIDENCE.md` (or a clearly named path).
5. Commit with:
   `git -c user.name="Mark LaPointe" -c user.email="mark@cloudbsd.org" commit ...`

### Handling merge conflicts
Rebase onto the branch, keep tests green, do not force-push `main`.

---

## Reading order

1. `AGENTS.md` (this file; Claude Code starts at `CLAUDE.md` → `@AGENTS.md`)
2. `.plan/0001-Hawkeye-Workflow.md`
3. `.plan/0200-Hawkeye-Overview.md`
4. Security series `.plan/010x`
5. `docs/ARCHITECTURE.md`
6. Implementation tasks `.plan/0300`

---

## Key design decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Language | Go | CloudBSD backend language; static binary toward `/rescue` |
| Knowledge | SQLite FTS client, not vendored | Corpus is hawkeye-data |
| Apply | Default dry-run | Privileged mutation needs an operator |
| LLM | Interface, llama.cpp hunch, GPU then CPU | Missing GPU must not block CPU jobs |
| MCP | stdio + Streamable HTTP/TLS on 127.0.0.1 | Operator tool, not a public API |
| Config | JSON RFC 8259 | CloudBSD law |
| License | BSD 3-Clause, REVYTECH, Inc. | CloudBSD law |

---

## Quick reference

### Key files

| File | Purpose |
|------|---------|
| `cmd/hawkeye/main.go` | Entry |
| `internal/cli/cli.go` | Commands |
| `internal/redact/redact.go` | Secret scrubbing |
| `internal/apply/apply.go` | Dry-run default and exec gate |
| `internal/knowledge/knowledge.go` | RO SQLite FTS |
| `rc.d/hawkeye` | Optional updater + localhost MCP |
| `man/hawkeye.8` | Program manual |
| `man/hawkeye.conf.5` | Config keys |

### Key commands

```sh
hawkeye --check-config
hawkeye init
hawkeye consult "zpool degraded"
hawkeye plan "restart sshd"
hawkeye apply --dry-run plan.json
hawkeye apply --yes plan.json
hawkeye doctor
hawkeye mcp --stdio
```

### Key groups / users

| User | Purpose |
|------|---------|
| `root` | Privileged apply |
| service user (future) | rc.d MCP after bind |

---

## Need help?

1. Check `.plan/` and `docs/ARCHITECTURE.md`.
2. Mark the task `BLOCKED` with a reason.
3. Do not vendor hawkeye-data. Do not add a public web chat.
