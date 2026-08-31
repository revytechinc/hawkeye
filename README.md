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
| `hawkeye` | Panic path. On a TTY, print a host first-look (fstab, rc, zpool, disks, net — not `doctor`), then type the problem at `>`. Each line is a consult, then `Apply these steps? [y/N/e]`. `quit`/`exit`/`q`/Ctrl-D leave. Non-TTY with no args prints a reminder to run on a terminal. `--json` dumps inspect JSON and never enters the session. |
| `hawkeye inspect` | Host first-look only (same findings as the session banner). Diagnose only. `--json` for the machine object. Not `doctor`. |
| `hawkeye consult` | Diagnose using knowledge FTS, optional sqlite-vec rank when embeddings exist and RAM allows, and optional LLM. Operator session on stdout; `--json` / `HAWKEYE_JSON=1` for the machine object. TTY asks `[y/N/e]` to apply or edit; default N. `--json`, pipes, and MCP do not prompt. Mutation still needs `--yes` or a second y. Empty embeddings stay FTS-only (quiet). |
| `hawkeye plan` | Propose steps from the lead consult playbook (stored commands). No mutation. Operator session on stdout; `--json` / `HAWKEYE_JSON=1` for the JSON plan. Dry-run default; `--yes` to land via apply. |
| `hawkeye apply [--dry-run\|--yes]` | Mutate. **Default is dry-run.** LLM never execs as root. Audited. |
| `hawkeye doctor` | Service health: config, perms, pidfile, deps, headroom. Human + JSON. Non-zero if unhealthy. |
| `hawkeye mcp` | MCP server (stdio default; Streamable HTTP on `127.0.0.1`, bearer token required). |
| `hawkeye update` | Refresh knowledge from hawkeye-data artifacts when writable. Unset `HAWKEYE_UPDATE_SOURCE` skips (rc start stays healthy). Dest defaults to `/usr/local/share/hawkeye/knowledge.sqlite`. |
| `hawkeye embed [--dry-run\|--yes]` | Fill `embeddings` on a writable kit copy using the local embedder. **Default is dry-run.** `--yes` writes. Refuses a read-only store. Consult stays read-only. |
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
Stage `/rescue/hawkeye` and the `/boot/hawkeye` prefix (DESTDIR-friendly):

```sh
make install-rescue
# or: make install-rescue DESTDIR=/tmp/stage KNOWLEDGE_SRC=/path/from/hawkeye-data/knowledge.sqlite
```

The port option `RESCUE` is off by default (thin bastille jails often have
`/rescue` as a dangling symlink). Live `make install-rescue` installs
*into* a real `/rescue` (or a symlink to a writable rescue image) without
replacing FreeBSD tools. A dangling `/rescue` symlink is replaced with a
real directory so `/rescue/hawkeye` is a runnable binary. `/boot/hawkeye`
is created when `/boot` exists and is writable. A read-only `/rescue` or
`/boot` (EROFS/EACCES/EPERM) is skipped. The install does not remount.
Knowledge sqlite is not vendored here; hawkeye-data owns the dual
prefix. `misc/llama-cpp` is optional and not a package dependency.

Configuration is JSON (RFC 8259) under `/usr/local/etc/cloudbsd/hawkeye/` or
XDG. Install ships `config.json.sample` (mode `0644`, no secrets). A live
`config.json` is optional: missing uses compiled defaults, so `hawkeye doctor`
and `hawkeye --check-config` both succeed after pkg/make install. A present
file is still validated; invalid JSON fails. Secrets are environment variables
(`HAWKEYE_LLM_API_KEY`). Local inference uses a llama.cpp-style
binary (`HAWKEYE_LLM_BIN` / `llm.local.bin`, or `PATH`:
`llama-completion` then `llama-cli` then `llama.cpp`). `llama-cli`
b9426 is conversation-only; Complete adds `--single-turn --simple-io`
so the panic session does not hang on `>`. Trailing chat leftovers
(`> EOF by user`, `Exiting...`) are stripped. Embed resolves
`llama-embedding` from `PATH` or `/usr/local/bin/llama-embedding` and
does not reuse `llama-completion`. A GGUF
(`HAWKEYE_LLM_MODEL` / `llm.local.model_path`, or the first `*.gguf` under
`/usr/local/share/hawkeye/models`, `/boot/hawkeye/models`, or
`HAWKEYE_MODELS_DIR`). Missing GGUF is a quiet TTY skip; `doctor` notes it
and stays healthy. Optional embeddings use `HAWKEYE_EMBED_MODEL` /
`llm.local.embed_model_path` (local only; no cloud; no GGUF vendored).
`hawkeye embed --yes` fills a writable kit; consult never writes.
Empty embeddings stay FTS-only. When vectors exist and RAM allows,
consult ranks with sqlite-vec. GPU if present and VRAM is known, then CPU;
null VRAM (thin jail `/dev/nvidia0`) stays `-ngl 0` and does not skip
consult. Knowledge is
`knowledge.sqlite` from hawkeye-data (FTS5 tables `documents_fts` and
`playbooks_fts`, with fallback to legacy `knowledge_fts`, plus optional
`embeddings` FLOAT32 rows). Search order is
`HAWKEYE_KNOWLEDGE_PATH` (exclusive when set), then `/boot/hawkeye`, then
`/usr/local/share/hawkeye`, then XDG. See
`hawkeye.conf(5)`.

## License

BSD 3-Clause. Copyright (c) 2026 REVYTECH, Inc.
