# Hawkeye

Meatball surgery on servers and desktops. Trench-warfare medicine for FreeBSD.

Privileged CLI that can modify the system. Optional `rc.d` for knowledge-artifact
updates and a token-authenticated MCP listener (loopback; nginx TLS).

![Hawkeye Pierce, 1975 CBS still (public domain in the United States; restoration REVYTECH)](docs/images/hawkeye-pierce-hero.png)


**This repository is binaries only.** Knowledge corpora live in
[hawkeye-data](https://github.com/revytechinc/hawkeye-data). The website lives
in [hawkeye-www](https://github.com/revytechinc/hawkeye-www). Do not vendor the
corpus here.

Hawkeye is **not** a public chat UI. It is **not** the revytechcommander doctor.

Public Streamable HTTP MCP: `https://hawkeye.revytechinc.com/mcp`.
A bearer token is required (`HAWKEYE_MCP_TOKEN` or `HAWKEYE_MCP_TOKEN_FILE`,
mode `0600`). Missing or wrong token returns HTTP 401. Do not put a token in
this README, man pages, or JSON config — JSON names the env var only.
stdio MCP is for local use and does not require a token.
nginx terminates TLS; Hawkeye still validates the bearer token.
Apply/consult through MCP use the same dry-run / `--yes` gate; the LLM never
execs as root.

## Commands

| Command | What it does |
|---------|----------------|
| `hawkeye` | Panic path. On a TTY, type the problem at `>`. Each line is a consult, then `Apply these steps? [y/N/e]`. `quit`/`exit`/`q`/Ctrl-D leave. Non-TTY with no args prints a reminder to run on a terminal. `--json` never enters the session. |
| `hawkeye consult` | Diagnose using knowledge FTS + optional LLM. Operator session on stdout; `--json` / `HAWKEYE_JSON=1` for the machine object. TTY asks `[y/N/e]` to apply or edit; default N. `--json`, pipes, and MCP do not prompt. Mutation still needs `--yes` or a second y. |
| `hawkeye plan` | Propose steps. No mutation. Operator session on stdout; `--json` / `HAWKEYE_JSON=1` for the JSON plan. |
| `hawkeye apply [--dry-run\|--yes]` | Mutate. **Default is dry-run.** LLM never execs as root. Audited. |
| `hawkeye doctor` | Service health: config, perms, pidfile, deps, headroom. Human + JSON. Non-zero if unhealthy. |
| `hawkeye mcp` | MCP server (stdio default; Streamable HTTP on `127.0.0.1`, bearer token required). |
| `hawkeye update` | Refresh knowledge from hawkeye-data artifacts when writable. |
| `hawkeye init` | Write sample JSON config (mode `0600`). |
| `hawkeye --check-config` | Validate JSON config and exit. Missing file uses compiled defaults (same as doctor). |

Manual pages: `hawkeye(8)`, `hawkeye.conf(5)`.

## Tiers

| Tier | Meaning | LLM |
|------|---------|-----|
| 0 | Rescue: RO root, maybe no `/usr` `/var`, no net, no GPU. Knowledge from `/boot/hawkeye`. | FTS only; not required |
| 1 | Root writable, islanded | CPU embedder if RAM allows |
| 2 | Network up | May escalate to Grok / FreeGrok / Claude. GPU if it works. |

On start Hawkeye probes `kern.securelevel`, mounts, ZFS readonly, `/usr`, `/var`,
carrier, and `/rescue`. If the root is read-only, the **first skill is
`unlock-rw`**, not `pkg`.

Consult needs no write. Knowledge opens SQLite `mode=ro` and `immutable=1` when
the root is RO.

## Install

```sh
pkg install hawkeye
```

That package depends on hawkeye-data, so the knowledge kit is installed
with the binary. You do not need `pkg install hawkeye hawkeye-data`.

## Build

```sh
make build
make test
```

Static /rescue-oriented build: `CGO_ENABLED=0 make build`.

Configuration is JSON (RFC 8259) under `/usr/local/etc/cloudbsd/hawkeye/` or
XDG. Install ships `config.json.sample` (mode `0644`, no secrets). A live
`config.json` is optional: missing uses compiled defaults, so `hawkeye doctor`
and `hawkeye --check-config` both succeed after pkg/make install. A present
file is still validated; invalid JSON fails. Secrets are environment variables
(`HAWKEYE_LLM_API_KEY`). Knowledge is
`knowledge.sqlite` from hawkeye-data (FTS5 tables `documents_fts` and
`playbooks_fts`, with fallback to legacy `knowledge_fts`). Override the search
path with `HAWKEYE_KNOWLEDGE_PATH` (directory or sqlite file). See
`hawkeye.conf(5)`.

## License

BSD 3-Clause. Copyright (c) 2026 REVYTECH, Inc.
