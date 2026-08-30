# Hawkeye skeleton — test evidence

Captured 2026-08-30 (America/Chicago). Host `uname` may say Linux; CloudBSD target is FreeBSD.

## 1. Red (failing stubs)

TDD red for redact, apply dry-run default, probe RO/`unlock-rw`, `--check-config`, doctor, knowledge RO open:

```
# github.com/revytechinc/hawkeye/internal/knowledge
internal/knowledge/knowledge_test.go:27:32: string literal not terminated
FAIL	github.com/revytechinc/hawkeye/internal/knowledge [setup failed]
# github.com/revytechinc/hawkeye/internal/doctor
internal/doctor/doctor.go:49:33: newline in string
internal/doctor/doctor.go:50:2: newline in string
internal/doctor/doctor.go:51:1: missing return
--- FAIL: TestString_RedactsFakeOpenSSHKey (0.00s)
    redact_test.go:19: private key material leaked: "-----BEGIN OPENSSH PRIVATE KEY-----\nFAKE_TEST_KEY_NOT_A_REAL_SECRET_aaaaaaaa\n-----END OPENSSH PRIVATE KEY-----"
--- FAIL: TestString_RedactsFakePasswordJSON (0.00s)
    redact_test.go:30: password leaked: "{\"username\":\"operator\",\"password\":\"fake-password-for-tests-only\"}"
--- FAIL: TestString_RedactsFakeAPIToken (0.00s)
    redact_test.go:38: api token leaked: "api_token=fake_api_token_AAAAAAAAAAAAAAAA"
--- FAIL: TestString_RedactsFakeHtpasswd (0.00s)
    redact_test.go:46: htpasswd blob leaked: "testuser:$apr1$faketest$abcdefghijklmnopqrstuv"
--- FAIL: TestString_RedactsFakeBasicBlob (0.00s)
    redact_test.go:54: Basic blob leaked: "Authorization: Basic ZmFrZXVzZXI6ZmFrZXBhc3M="
--- FAIL: TestString_RedactsFakeBearerAndPATs (0.00s)
    redact_test.go:69: github pat leaked: "Bearer FAKESECRET_a3b4c5d6e7f8g9h0i1j2\nghp_FakeGitHubTokenForTestsOnly1234567890\nsk-fakeOpenAIKeyForTestsOnly1234567890abcd"
--- FAIL: TestContainsSecret_FakeFixtureFile (0.00s)
    redact_test.go:90: expected fake fixture to be detected as containing secrets
FAIL
FAIL	github.com/revytechinc/hawkeye/internal/redact	0.003s
--- FAIL: TestResolveMode_DefaultIsDryRun (0.00s)
    apply_test.go:30: default mode = apply, want dry-run
--- FAIL: TestResolveMode_YesMutatesUnlessDryRun (0.00s)
    apply_test.go:39: --dry-run wins over --yes: got apply
--- FAIL: TestExecute_DefaultDryRunDoesNotCallExecutor (0.00s)
    apply_test.go:57: result {DryRun:false Applied:true Steps:[]}
--- FAIL: TestExecute_YesCallsExecutorForOperator (0.00s)
    apply_test.go:68: calls = 0
--- FAIL: TestExecute_LLMActorNeverExecsPrivileged (0.00s)
    apply_test.go:79: expected error: LLM must not exec as root
--- FAIL: TestExecute_MCPActorPrivilegedIsDryRunOnly (0.00s)
    apply_test.go:99: MCP apply must remain dry-run for privileged steps
FAIL
FAIL	github.com/revytechinc/hawkeye/internal/apply	0.002s
--- FAIL: TestProbe_RescueROIsTier0AndUnlockRW (0.00s)
    probe_test.go:42: expected root RO: {Securelevel:0 SecurelevelOK:false RootRO:false UsrPresent:false VarPresent:false NetworkUp:false GPUPresent:false RescuePresent:false ZFSReadOnly:false Tier:2}
--- FAIL: TestProbe_WritableIslandedIsTier1 (0.00s)
    probe_test.go:69: tier = 2, want 1
--- FAIL: TestProbe_NetworkUpIsTier2 (0.00s)
    probe_test.go:88: {Securelevel:0 SecurelevelOK:false RootRO:false UsrPresent:false VarPresent:false NetworkUp:false GPUPresent:false RescuePresent:false ZFSReadOnly:false Tier:2}
--- FAIL: TestProbe_ZFSReadOnlyCountsAsRootRO (0.00s)
    probe_test.go:100: {Securelevel:0 SecurelevelOK:false RootRO:false UsrPresent:false VarPresent:false NetworkUp:false GPUPresent:false RescuePresent:false ZFSReadOnly:false Tier:2}
FAIL
FAIL	github.com/revytechinc/hawkeye/internal/probe	0.002s
--- FAIL: TestCheckFile_RejectsInvalidJSON (0.00s)
    config_test.go:58: expected invalid JSON to fail --check-config
--- FAIL: TestCheckFile_RejectsComments (0.00s)
    config_test.go:69: JSONC must be rejected
--- FAIL: TestValidate_RejectsPublicBind (0.00s)
    config_test.go:77: public bind must be rejected by default validator
--- FAIL: TestValidate_RejectsUnknownLogLevel (0.00s)
    config_test.go:85: expected log_level error
FAIL
FAIL	github.com/revytechinc/hawkeye/internal/config	0.004s
FAIL	github.com/revytechinc/hawkeye/internal/doctor [build failed]
--- FAIL: TestAllow_ExhaustedRAMBlocks (0.00s)
    headroom_test.go:26: expected RAM headroom failure
--- FAIL: TestAllow_GPUJobFailsWhenRequiredVRAMExhausted (0.00s)
    headroom_test.go:36: expected VRAM headroom failure
FAIL
FAIL	github.com/revytechinc/hawkeye/internal/headroom	0.002s
FAIL
```

## 2. Green

`go test ./internal/... ./cmd/hawkeye -count=1`

```
ok  	github.com/revytechinc/hawkeye/internal/apply	0.008s	coverage: 98.2% of statements
ok  	github.com/revytechinc/hawkeye/internal/audit	0.005s	coverage: 86.8% of statements
ok  	github.com/revytechinc/hawkeye/internal/cli	0.039s	coverage: 76.0% of statements
ok  	github.com/revytechinc/hawkeye/internal/config	0.007s	coverage: 82.3% of statements
ok  	github.com/revytechinc/hawkeye/internal/consult	0.018s	coverage: 89.5% of statements
ok  	github.com/revytechinc/hawkeye/internal/doctor	0.004s	coverage: 95.7% of statements
ok  	github.com/revytechinc/hawkeye/internal/headroom	0.004s	coverage: 97.1% of statements
ok  	github.com/revytechinc/hawkeye/internal/knowledge	0.044s	coverage: 93.4% of statements
ok  	github.com/revytechinc/hawkeye/internal/llm	0.006s	coverage: 84.6% of statements
ok  	github.com/revytechinc/hawkeye/internal/mcp	0.006s	coverage: 84.0% of statements
ok  	github.com/revytechinc/hawkeye/internal/pidfile	0.005s	coverage: 85.7% of statements
ok  	github.com/revytechinc/hawkeye/internal/probe	0.004s	coverage: 86.2% of statements
ok  	github.com/revytechinc/hawkeye/internal/redact	0.003s	coverage: 100.0% of statements
ok  	github.com/revytechinc/hawkeye/internal/reload	0.005s	coverage: 92.9% of statements
ok  	github.com/revytechinc/hawkeye/internal/update	0.004s	coverage: 69.6% of statements
ok  	github.com/revytechinc/hawkeye/internal/version	0.002s	coverage: [no statements]
ok  	github.com/revytechinc/hawkeye/cmd/hawkeye	0.004s	coverage: 0.0% of statements
```

All packages PASS. `CGO_ENABLED=0 go build ./cmd/hawkeye` succeeded (rescue-oriented static binary path).

## 3. Coverage (critical paths)

Application-code total from this run is in the cover profile. Critical paths:

```
github.com/revytechinc/hawkeye/internal/apply/apply.go:34:		ResolveMode		100.0%
github.com/revytechinc/hawkeye/internal/apply/apply.go:113:		Execute			96.7%
github.com/revytechinc/hawkeye/internal/config/config.go:93:		Validate		100.0%
github.com/revytechinc/hawkeye/internal/config/config.go:136:		CheckFile		80.0%
github.com/revytechinc/hawkeye/internal/knowledge/knowledge.go:53:	fileDSN			100.0%
github.com/revytechinc/hawkeye/internal/probe/probe.go:32:		Probe			100.0%
github.com/revytechinc/hawkeye/internal/probe/probe.go:57:		FirstSkill		100.0%
github.com/revytechinc/hawkeye/internal/redact/redact.go:74:		String			100.0%
github.com/revytechinc/hawkeye/internal/redact/redact.go:83:		Bytes			100.0%
github.com/revytechinc/hawkeye/internal/redact/redact.go:88:		ContainsSecret		100.0%
total:									(statements)		84.8%
```

| Path | Result |
|------|--------|
| redact | 100% |
| apply.ResolveMode (dry-run default) | 100% |
| knowledge.Open (RO/immutable) | 100% |
| probe.Probe / FirstSkill | 100% |
| config.Validate | 100% |

## 4. `--check-config`

Valid sample:

```
$ hawkeye --config configs/config.example.json --check-config
configuration ok: configs/config.example.json
```

exit 0

Missing file:

```
$ hawkeye --config /tmp/nope-hawkeye.json --check-config
hawkeye: --check-config failed: open /tmp/nope-hawkeye.json: no such file or directory
```

exit 1

## 5. `hawkeye apply` default dry-run

```
{
  "dry_run": true,
  "applied": false,
  "steps": [
    {
      "id": "1",
      "skipped": true,
      "dry_run": true,
      "output": "dry-run: true"
    }
  ]
}
```

`applied` is false. Executor is not invoked.

## 6. `hawkeye doctor` (human+JSON, non-zero if unhealthy)

This CI/dev host has no hawkeye-data artifact, so doctor is **unhealthy** (dependencies). That is correct:

```
{
  "healthy": false,
  "checks": [
    {
      "name": "config",
      "ok": true,
      "detail": "configuration is valid"
    },
    {
      "name": "permissions",
      "ok": true,
      "detail": "config permissions acceptable"
    },
    {
      "name": "pidfile",
      "ok": true,
      "detail": "service is not running; pidfile not required"
    },
    {
      "name": "dependencies",
      "ok": false,
      "detail": "knowledge store missing"
    },
    {
      "name": "headroom",
      "ok": true,
      "detail": "gpu absent (ok; not required for this operation)"
    }
  ],
  "resources": {
    "ram_free_bytes": 13404315648,
    "ram_total_bytes": 16791228416,
    "cpu_pct": 0,
    "disk_free_bytes": 117258059776,
    "gpu_present": false,
    "gpu_vram_free_bytes": null
  },
  "tier": 0
}
```

exit 1 when knowledge is missing. GPU absent is reported and does not fail headroom for non-GPU work.

## 7. Man pages

`mandoc` was not installed (debian InRelease 502; mandoc.bsd.lv TLS EOF). Equivalent mdoc lint:

```
mdoc equivalent lint: OK
checked man/hawkeye.8 man/hawkeye.conf.5
required macros present: Dd Dt NAME SYNOPSIS DESCRIPTION COMMANDS OPTIONS SIGNALS FILES KEYS ENVIRONMENT SEE ALSO
```

Re-run `mandoc -T lint man/hawkeye.8 man/hawkeye.conf.5` on FreeBSD/CloudBSD.

## 8. Notes

- Tests use **FAKE** secret fixtures only (`internal/redact/testdata/fake_secrets.txt`).
- Knowledge corpus is not vendored; Open consumes a temp SQLite FTS DB in tests.
- No public chat UI is present.


## 9. Public token-authenticated MCP (2026-08-30)

`go test ./internal/... ./cmd/hawkeye -count=1` PASS.

HTTP MCP:

- missing token → 401 + `WWW-Authenticate: Bearer`
- wrong fixture token → 401
- fixture token → not 401 (GET 200 / POST initialize 200)
- stdio MCP still works without a token
- privileged MCP apply remains dry-run (`ActorMCP`)
- JSON config names `HAWKEYE_MCP_TOKEN`; tests use fixture tokens only

`--check-config` on `configs/config.example.json`: exit 0.

TLS is optional on the loopback listener (nginx terminates TLS). Public bind is still rejected.
