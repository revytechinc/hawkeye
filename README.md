# Hawkeye

Meatball surgery on servers and desktops. Trench-warfare medicine for FreeBSD.

Privileged CLI that can modify the system. Optional `rc.d` for knowledge-artifact
updates and an optional localhost MCP listener.

![Hawkeye Pierce, 1975 CBS still (public domain in the United States; restoration REVYTECH)](docs/images/hawkeye-pierce-hero.png)


**This repository is binaries only.** Knowledge corpora live in
[hawkeye-data](https://github.com/revytechinc/hawkeye-data). The website lives
in [hawkeye-www](https://github.com/revytechinc/hawkeye-www). Do not vendor the
corpus here.

Hawkeye is **not** a public chat UI. It is **not** the revytechcommander doctor.

## Commands

| Command | What it does |
|---------|----------------|
| `hawkeye consult` | Diagnose using knowledge FTS + optional LLM. No writes. |
| `hawkeye plan` | JSON plan. No mutation. |
| `hawkeye apply [--dry-run\|--yes]` | Mutate. **Default is dry-run.** LLM never execs as root. Audited. |
| `hawkeye doctor` | Service health: config, perms, pidfile, deps, headroom. Human + JSON. Non-zero if unhealthy. |
| `hawkeye mcp` | MCP server (stdio default; Streamable HTTP over TLS on `127.0.0.1`). |
| `hawkeye update` | Refresh knowledge from hawkeye-data artifacts when writable. |
| `hawkeye init` | Write sample JSON config (mode `0600`). |
| `hawkeye --check-config` | Validate JSON config and exit. |

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

## Build

```sh
make build
make test
```

Static /rescue-oriented build: `CGO_ENABLED=0 make build`.

Configuration is JSON (RFC 8259) under `/usr/local/etc/cloudbsd/hawkeye/` or
XDG. Secrets are environment variables (`HAWKEYE_LLM_API_KEY`). Knowledge is
`knowledge.sqlite` from hawkeye-data (FTS5 tables `documents_fts` and
`playbooks_fts`, with fallback to legacy `knowledge_fts`). Override the search
path with `HAWKEYE_KNOWLEDGE_PATH` (directory or sqlite file). See
`hawkeye.conf(5)`.

## License

BSD 3-Clause. Copyright (c) 2026 REVYTECH, Inc.
