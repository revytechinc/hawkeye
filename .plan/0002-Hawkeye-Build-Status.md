# Hawkeye — Build Status

**Document ID:** HAWKEYE-0002
**Version:** 0.1.0
**Last Updated:** 2026-08-30
**Status:** ACTIVE

CI: `.github/workflows/test.yml` runs `go test ./...` and a `CGO_ENABLED=0` build.

Local:

```
make test
make cover
make build
```

Evidence: `docs/TEST-EVIDENCE.md`.
