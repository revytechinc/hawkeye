# Hawkeye architecture

**Document ID:** HAWKEYE-ARCH
**Version:** 0.1.0
**Last Updated:** 2026-08-30
**Status:** ACTIVE

This repository ships the Hawkeye **binary**. Knowledge lives in
[hawkeye-data](https://github.com/revytechinc/hawkeye-data). There is no public
chat UI.

## Layers

```mermaid
flowchart TD
    Operator["Operator CLI / stdio MCP"] -->|"consult / plan / apply / doctor"| CLI["Controller: cmd/hawkeye + internal/cli"]
    MCPHTTP["MCP Streamable HTTP TLS on 127.0.0.1"] --> CLI
    CLI --> Probe["probe: tier 0/1/2"]
    CLI --> Knowledge["knowledge client: SQLite FTS RO"]
    CLI --> PlanApply["plan + apply gate"]
    CLI --> Doctor["doctor + headroom"]
    CLI --> LLM["llm interface: llama.cpp GPU then CPU"]
    Knowledge -->|"consumes artifacts"| DataRepo["hawkeye-data artifacts"]
    PlanApply -->|"default dry-run; audit"| System["FreeBSD system"]
    LLM -->|"redact first"| Remote["optional Grok / FreeGrok / Claude"]
```

The UI, if any later, is a view only. This skeleton is operator CLI + MCP.
Backends bind loopback. Payloads are application DTOs; provider protocols are
never exposed. Secrets are redacted before any LLM or MCP payload.

## Apply gate

```mermaid
flowchart TD
    In["plan JSON"] --> Mode{"ResolveMode: default dry-run"}
    Mode -->|"no --yes"| Dry["dry-run: no exec"]
    Mode -->|"--yes and operator"| Actor{"actor"}
    Actor -->|"ActorLLM"| Refuse["refuse: LLM must not exec as root"]
    Actor -->|"ActorMCP privileged"| Dry
    Actor -->|"ActorOperator"| Exec["SysExecutor + audit log"]
```

## Knowledge search order

```mermaid
flowchart LR
    A["/boot/hawkeye"] --> B["/usr/local/share/hawkeye"]
    B --> C["XDG_DATA_HOME/hawkeye"]
```

Open is always read-only for consult. When the root is read-only, the DSN uses
`mode=ro` and `immutable=1` so SQLite will not try to create `-wal`/`-shm` files.

## Tiers

```mermaid
flowchart TD
    Start["Probe host"] --> RO{"root RO or missing /usr /var?"}
    RO -->|yes| T0["Tier 0 rescue: FTS, console, no LLM required"]
    RO -->|no| Net{"network carrier?"}
    Net -->|no| T1["Tier 1 islanded: CPU embedder if RAM allows"]
    Net -->|yes| T2["Tier 2: optional remote LLM, GPU if present"]
```

## Packages

| Package | Role |
|---------|------|
| `cmd/hawkeye` | Process entry |
| `internal/cli` | Command dispatcher |
| `internal/config` | JSON RFC 8259, XDG, `--check-config` |
| `internal/probe` | Tier classification |
| `internal/knowledge` | SQLite FTS client |
| `internal/apply` | Plan/apply gate |
| `internal/doctor` | Service health |
| `internal/headroom` | Consumption-based resources |
| `internal/llm` | Local/remote LLM interface |
| `internal/mcp` | stdio + Streamable HTTP |
| `internal/redact` | Secret scrubbing |
| `internal/audit` | Apply audit log |
| `internal/reload` | SIGHUP validate-then-swap |
