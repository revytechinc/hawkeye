# Hawkeye — Security Threat Model

**Document ID:** HAWKEYE-0101
**Version:** 0.1.0
**Last Updated:** 2026-08-30
**Status:** ACTIVE

```mermaid
flowchart LR
    T1["T1 LLM exec as root"] --> C1["apply gate ActorLLM refuse"]
    T2["T2 secret in prompt"] --> C2["redact before LLM/MCP"]
    T3["T3 public MCP"] --> C3["loopback + TLS required"]
    T4["T4 write on RO root"] --> C4["consult RO/immutable; update refused"]
```

STRIDE focus: Tampering (apply), Information disclosure (secrets), Elevation (root exec).
