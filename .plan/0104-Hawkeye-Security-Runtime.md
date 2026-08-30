# Hawkeye — Security Runtime

**Document ID:** HAWKEYE-0104
**Version:** 0.1.0
**Last Updated:** 2026-08-30
**Status:** ACTIVE

- Tier 0: no writes, console logs, knowledge from `/boot/hawkeye`.
- SQLite URI `mode=ro` and `immutable=1` on RO root (no WAL sidecars).
- Pidfile non-empty, non-negative, removed on stop.
- Headroom refuses jobs that would exhaust consumed resources.
