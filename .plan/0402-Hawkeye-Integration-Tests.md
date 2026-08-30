# Hawkeye — Integration Tests

**Document ID:** HAWKEYE-0402
**Version:** 0.1.0
**Last Updated:** 2026-08-30
**Status:** ACTIVE

Seams covered:

- CLI `RunEnv` for init, check-config, consult, plan, apply, doctor, mcp stdio
- SIGHUP-equivalent `reload.Holder.ReloadFile` (bad config keeps old)
- Knowledge FTS open + search on a real SQLite file
- MCP HTTP handler via `httptest`
