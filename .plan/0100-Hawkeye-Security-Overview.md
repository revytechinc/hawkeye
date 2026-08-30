# Hawkeye — Security Overview

**Document ID:** HAWKEYE-0100
**Version:** 0.1.0
**Last Updated:** 2026-08-30
**Status:** ACTIVE

Hawkeye is a privileged operator tool. Threats center on:

- accidental or LLM-driven mutation
- secret leakage into LLM/MCP payloads
- binding MCP on a public interface
- writing to a read-only rescue root

Controls: redact, dry-run default, loopback MCP + TLS, RO SQLite open, audit log, doctor.
