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

## 10. User config precedence and readable pidfile (2026-08-30)

Jail `make test` failed on `TestResolvePath_UserConfig` because
`/usr/local/etc/cloudbsd/hawkeye/config.json` exists and `ResolvePath`
preferred it over `$XDG_CONFIG_HOME`. Doctor as a non-root operator
reported `pidfile is empty` when `/var/run/hawkeye.pid` was mode 0600
(unreadable).

Fixes:

- `ResolvePath`: user config wins when both exist; system is fallback
- doctor reports `pidfile is unreadable` instead of `empty`
- rc.d `start_postcmd` sets pidfile mode 0644 (PID is not a secret)

`go test ./internal/... ./cmd/hawkeye -count=1` PASS on this branch.

`mandoc -T lint` STYLE only (Os CloudBSD, Xr not installed in PATH).

## 11. Consult/plan human TTY (2026-08-31)

T013: default `hawkeye consult` / `hawkeye plan` are an operator session, not a JSON blob.
`--json` and `HAWKEYE_JSON=1` still emit the machine object. MCP tools stay JSON on the wire.

Red (Human() missing; default CLI still dumped JSON):

```
# github.com/revytechinc/hawkeye/internal/apply_test
internal/apply/human_test.go:26:11: p.Human undefined
# github.com/revytechinc/hawkeye/internal/consult_test
internal/consult/human_test.go:39:11: r.Human undefined
--- FAIL: TestConsult_DefaultIsHumanNotJSON
    human_test.go:26: default consult must be operator prose, not JSON:
        { "query": "ZFS root is read-only after boot", "hits": [ { "Title": ... } ] }
--- FAIL: TestPlan_DefaultIsHumanNotJSON
```

Green:

`go test ./internal/... ./cmd/hawkeye -count=1 -coverprofile=coverage.out` PASS.

```
ok  github.com/revytechinc/hawkeye/internal/apply     coverage: 98.8%
ok  github.com/revytechinc/hawkeye/internal/cli       coverage: 77.1%
ok  github.com/revytechinc/hawkeye/internal/consult   coverage: 96.7%
ok  github.com/revytechinc/hawkeye/internal/doctor    coverage: 95.9%
ok  github.com/revytechinc/hawkeye/internal/mcp       coverage: 83.5%
ok  github.com/revytechinc/hawkeye/internal/redact    coverage: 100.0%
total: (statements) 86.1%
```

New formatters: `consult.Result.Human` 96.2%, `apply.Plan.Human` 100%, `wantJSON` 100%. Redact remains 100%.

`--check-config` on `configs/config.example.json`: exit 0 (`configuration ok`).

`hawkeye doctor` (no kit on this host): UNHEALTHY, dependencies FAIL, GPU absent is ok. Exit 1. Human + JSON both printed.

Operator session (fixture kit; this host is writable so tier 2):

```
$ hawkeye consult 'ZFS root is read-only after boot'
consult  ZFS root is read-only after boot
tier 2
llm skipped: local llm model is not configured

1. Remount ZFS root read-write
   Root is a ZFS dataset and is mounted read-only after boot.

   If the root pool is imported readonly, remount ZFS read-write.

   export PATH=/rescue:/sbin:/bin
   zfs set readonly=off "$ROOTDS"
   mount -u -o rw /
```

`--json` still dumps the machine object (`"query"`, `"hits"`, `"title"`).
`hawkeye plan 'restart sshd'` prints prose; `hawkeye plan --json` prints the plan object.

`mandoc` not installed here. Equivalent mdoc lint: required macros present; `HAWKEYE_JSON` documented in `hawkeye.conf(5)`.

`CGO_ENABLED=0 go build -buildvcs=false ./cmd/hawkeye` succeeded.

## 12. Consult TTY lead playbook (2026-08-31)

T014: default `hawkeye consult` leads with the most actionable playbook (title,
stored when-to-use, stored commands), then related titles under `also:`.
No query/tier keys, no Rank, no Tags-as-JSON, no `llm skipped`.
`--json` keeps the FTS machine object (remount may stay 4th).

Red: Human() still printed `consult  …`, `tier N`, `llm skipped`, and listed
hits in FTS order (`1. List, activate…` before remount).

Green: `go test ./internal/... ./cmd/hawkeye -count=1` PASS.

Fixture kit (playbooks_fts; query `ZFS root is read-only after boot`):

```
$ hawkeye consult 'ZFS root is read-only after boot'
Remount ZFS root read-write
  Root is a ZFS dataset and is mounted read-only (single-user, panic
  remount, zfs readonly=on, or a readonly pool import).

  export PATH=/rescue:/sbin:/bin:/usr/sbin:/usr/bin
  mount -p
  zfs set readonly=off "$ROOTDS"
  mount -u -o rw /

also:
  List, activate, or roll back a ZFS boot environment
  Import a ZFS pool (readonly first, then unlock)
```

`--json` still dumps `"query"` / `"hits"` with original FTS order.
`--check-config` on `configs/config.example.json`: exit 0.
`hawkeye doctor`: UNHEALTHY (knowledge missing), GPU absent ok.
`CGO_ENABLED=0 go build -buildvcs=false ./cmd/hawkeye` succeeded.

## 13. TTY consult apply/edit prompt (2026-08-31)

T015: after the operator TTY consult session (T013/T014), prompt
`Apply these steps? [y/N/e]`. `--json`, `HAWKEYE_JSON`, non-TTY, and MCP
do not prompt. Apply still defaults to dry-run; landing needs `--yes` or
a second `y`. Editor abort/empty does not apply. Secrets are redacted
per plan field (whole-document `redact.String` broke `password=` JSON).

Rebased onto `origin/main` so Human() lead-playbook output and the prompt
both remain.

`go test ./internal/... ./cmd/hawkeye -count=1` PASS after rebase (see
follow-up capture). Apply `ResolveMode` remains 100%. Tests use FAKE
secret fixtures only.

## 14. `--check-config` missing file uses defaults (2026-08-31)

T016: native pkg/make install ships `config.json.sample` only. `doctor` already
treated a missing live `config.json` as compiled defaults; `--check-config`
failed with `open …/config.json: no such file or directory`.

Red (`CheckFile` still required a live file):

```
--- FAIL: TestCheckFile_MissingUsesDefaults
    config_test.go:57: missing config must use compiled defaults (same as doctor): open …/config.json: no such file or directory
--- FAIL: TestCheckFile_SampleOnlyDirUsesDefaults
    config_test.go:79: sample-only dir must use defaults; operators must not be required to copy the sample: open …/config.json: no such file or directory
--- FAIL: TestResolvePath_SampleOnlyDoesNotSelectSample
    resolve_test.go:57: sample-only system dir must check as defaults: open …/config.json: no such file or directory
--- FAIL: TestCheckConfig_MissingUsesDefaults
    cli_test.go:92: missing config must be valid defaults: 1  hawkeye: --check-config failed: open …/config.json: no such file or directory
--- FAIL: TestCheckConfig_SampleOnlyDir
--- FAIL: TestCheckConfig_AgreesWithDoctorOnMissing
```

Green: `go test ./internal/... ./cmd/hawkeye -count=1 -coverprofile=coverage.out` PASS.

```
ok  github.com/revytechinc/hawkeye/internal/apply     coverage: 98.8%
ok  github.com/revytechinc/hawkeye/internal/cli       coverage: 77.3%
ok  github.com/revytechinc/hawkeye/internal/config    coverage: 85.3%
ok  github.com/revytechinc/hawkeye/internal/doctor    coverage: 95.9%
ok  github.com/revytechinc/hawkeye/internal/redact    coverage: 100.0%
total: (statements) 87.0%
```

`config.CheckFile` 85.7%. `config.Validate` 96.3%. `apply.ResolveMode` still 100%
(dry-run default not weakened).

Binary evidence (`CGO_ENABLED=0 go build -buildvcs=false ./cmd/hawkeye`):

```
$ hawkeye --config configs/config.example.json --check-config
configuration ok: configs/config.example.json
# exit 0

$ hawkeye --config /tmp/nope-hawkeye.json --check-config
configuration ok: defaults (no file at /tmp/nope-hawkeye.json)
# exit 0

$ # sample-only dir: config.json.sample present, config.json absent
$ hawkeye --config "$sodir/config.json" --check-config
configuration ok: defaults (no file at …/config.json)
# exit 0

$ hawkeye --config /tmp/hawkeye-bad.json --check-config
hawkeye: --check-config failed: not valid JSON (RFC 8259)
# exit 1
```

`hawkeye --json doctor` (no kit on this host): config check `ok: true`
(`configuration is valid`); dependencies FAIL (knowledge store missing);
GPU absent is ok. Exit 1.

`hawkeye apply` without `--yes`: `"dry_run": true`, `"applied": false`.

`mandoc` not installed here. Equivalent mdoc lint: required macros present
(Dd Dt NAME SYNOPSIS DESCRIPTION COMMANDS OPTIONS SIGNALS FILES KEYS
ENVIRONMENT SEE ALSO). `config.json.sample` documented in `hawkeye(8)` FILES
and `hawkeye.conf(5)`.

## 15. Bare hawkeye operator session (2026-08-31)

T017: `hawkeye` with no command on a TTY is the panic path (like
`ollama run`). Each line is a consult; then the T015 apply prompt
`Apply these steps? [y/N/e]`. N/Enter returns to `>`. `--yes` still
required to land. `--json` never enters the REPL. Non-TTY with no args
prints a reminder to run on a terminal (no pipe hang). Known
subcommands and MCP are unchanged. Positional words that are not a
command are the first query.

Red: no-args still dumped usage; `hawkeye ZFS …` was `unknown command`.

Green:

`go test ./internal/... ./cmd/hawkeye -count=1 -coverprofile=coverage.out` PASS.

```
ok  github.com/revytechinc/hawkeye/internal/apply     coverage: 98.8%
ok  github.com/revytechinc/hawkeye/internal/cli       coverage: 84.6%
ok  github.com/revytechinc/hawkeye/internal/consult   coverage: 96.2%
ok  github.com/revytechinc/hawkeye/internal/redact    coverage: 100.0%
total: (statements) 88.1%
```

`cmdSession`, `runREPL`, `isKnownCommand`, apply `ResolveMode` 100%.
`runConsultQuery` 84.6% (JSON/consult error paths). Redact 100%.
Tests use FAKE secret fixtures only.

`--check-config` on `configs/config.example.json`: exit 0.
`hawkeye --json` and `printf '' | hawkeye`: 
`hawkeye: run hawkeye on a terminal (type the problem at >, then y/N/e).`
`hawkeye doctor`: UNHEALTHY (knowledge missing), GPU absent ok. Exit 1.

`mandoc` not installed here. Equivalent mdoc lint: required macros
present; no-args documented as the panic path in `hawkeye(8)`.

`CGO_ENABLED=0 go build -buildvcs=false ./cmd/hawkeye` succeeded.

## 16. Plan from lead playbook stored commands (2026-08-31)

T018: `hawkeye plan` (and the TTY consult apply path) must turn the lead
consult playbook into apply steps using stored commands (playbooks.commands
JSON, else fenced body). No `echo <query>` stub. RootRO without a playbook
still keeps unlock-rw as the first skill. `--json` stays machine-shaped.
Apply remains dry-run by default; `--yes` (or TTY second y) to land.

Red (`makePlan` still stubbed):

```
# github.com/revytechinc/hawkeye/internal/consult_test
internal/consult/plan_test.go:23:9: r.Plan undefined
--- FAIL: TestSearch_PlaybookCommandsFromStore
    stored commands missing from hit: []string(nil)
--- FAIL: TestPlan_UsesPlaybookNotEchoStub
    plan must use stored remount command (got unlock-rw <rootpool>)
--- FAIL: TestPlan_HumanShowsStoredCommands
    1. diagnose
       echo ZFS root is read-only after boot
--- FAIL: TestConsult_TTY_ApplyDryRunsPlaybook
    dry-run: zfs set readonly=off <rootpool>
--- FAIL: TestSysExecutor_StoredShellLine
    exec: "echo hawkeye-shell && echo ok": executable file not found
```

Green: `go test ./internal/... ./cmd/hawkeye -count=1 -coverprofile=coverage.out` PASS.

```
ok  github.com/revytechinc/hawkeye/internal/apply      coverage: 98.8%
ok  github.com/revytechinc/hawkeye/internal/cli        coverage: 83.2%
ok  github.com/revytechinc/hawkeye/internal/consult    coverage: 96.8%
ok  github.com/revytechinc/hawkeye/internal/knowledge  coverage: 87.2%
ok  github.com/revytechinc/hawkeye/internal/redact     coverage: 100.0%
total: (statements) 87.7%
```

`consult.Result.Plan` 100%. `CommandLines` 100%. `apply.ResolveMode` 100%.
`SysExecutor.Run` 100% (stored playbook lines run via `/bin/sh -c`).
Redact remains 100%. Tests use FAKE fixtures only.

`--check-config` on `configs/config.example.json`: exit 0.
`hawkeye apply` without `--yes`: `"dry_run": true`, `"applied": false`.
`hawkeye --json doctor` (no kit): UNHEALTHY, GPU absent ok. Exit 1.

Fixture kit (query `ZFS root is read-only after boot`):

```
$ hawkeye plan --json 'ZFS root is read-only after boot'
{
  "id": "consult-plan",
  "source": "knowledge",
  "summary": "Remount ZFS root read-write",
  "steps": [
    {"id":"1","action":"export","argv":["export PATH=/rescue:/sbin:/bin:/usr/sbin:/usr/bin"],"privileged":true},
    {"id":"2","action":"mount","argv":["mount -p"],"privileged":true},
    {"id":"3","action":"zfs","argv":["zfs set readonly=off \"$ROOTDS\""],"privileged":true},
    {"id":"4","action":"mount","argv":["mount -u -o rw /"],"privileged":true}
  ]
}
```

TTY consult + `y` dry-runs those stored lines (not `echo <query>`).
`CGO_ENABLED=0 go build -buildvcs=false ./cmd/hawkeye` succeeded.

`mandoc` not installed here. Equivalent mdoc lint: required macros present
(Dd Dt NAME SYNOPSIS DESCRIPTION COMMANDS OPTIONS SIGNALS FILES KEYS
ENVIRONMENT SEE ALSO). `plan` documents stored playbook commands.

## 17. Reviewer blockers: playbook argv, FreeBSD carrier, MCP apply (2026-08-31)

T018 on main (PR 15, `c560e8e`) already builds TTY consult apply from
stored playbook commands. Added `TestConsult_TTY_YesAppliesPrintedPlaybookArgv`
so `y` must land `argv ==` the printed playbook lines, not `echo <query>`
and not `zfs set readonly=off <rootpool>`.

T019: `NetworkCarrier` no longer dies on missing Linux `/sys`. Tests
parse `ifconfig(8)` fixtures and inject `IfaceStatus` (no fake `/sys`).
`host_freebsd.go` uses getifaddrs + `SIOCGIFMEDIA`, then `ifconfig -a`.

T020: MCP `apply` uses CLI `SysExecutor` + auditor. Unprivileged + `yes`
must emit real `echo` output (CountingExecutor's `ok` fails the test).
Privileged MCP stays dry-run. Default dry-run; `--dry-run` wins over `--yes`.

Red (before the wiring):

```
--- FAIL: TestDefaultHost_FreeBSDCarrierWithoutSysfs
    absent /sys must still see FreeBSD em0 carrier
--- FAIL: TestParseIfconfig_Em0ActiveIgnoresLoopback
    ParseIfconfig undefined
--- FAIL: TestMCP_UnprivilegedYesUsesRealExecutor
    Apply claimed success without a real executor (CountingExecutor returns ok, no process)
```

Green: `CGO_ENABLED=0 go test ./internal/... ./cmd/hawkeye -count=1 -coverprofile=coverage.out` PASS.

```
ok  github.com/revytechinc/hawkeye/internal/apply      coverage: 98.8%
ok  github.com/revytechinc/hawkeye/internal/cli        coverage: 84.5%
ok  github.com/revytechinc/hawkeye/internal/consult    coverage: 96.8%
ok  github.com/revytechinc/hawkeye/internal/probe      coverage: 91.7%
ok  github.com/revytechinc/hawkeye/internal/redact     coverage: 100.0%
total: (statements) 88.5%
```

`consult.Result.Plan` 100%. `CommandLines` 100%. `mcpApply` 100%.
`executePlanActor` 100%. `CarrierUp` 100%. `IfmediaActive` 100%.
`ParseIfconfig` 100%. `NetworkCarrier` 100%. `sysfsCarrier` 100%.
`apply.ResolveMode` 100%. Redact 100%. Tests use FAKE fixtures only.

`--check-config` on `configs/config.example.json`: exit 0.
`hawkeye apply` without `--yes`: `"dry_run": true`, `"applied": false`.
`hawkeye apply --dry-run --yes`: `"dry_run": true` (`--dry-run` wins).
`hawkeye --json doctor` (no kit): UNHEALTHY (knowledge missing), GPU
absent ok. Exit 1. Doctor `tier` can be 2 when a non-loopback iface
has carrier (getifaddrs fallback; not stuck at 1 by missing `/sys`).

`CGO_ENABLED=0 go build -buildvcs=false ./cmd/hawkeye` succeeded.
`GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0 go build` succeeded.

`mandoc` not installed here. Equivalent mdoc lint: required macros
present (Dd Dt NAME SYNOPSIS DESCRIPTION COMMANDS OPTIONS SIGNALS
FILES SEE ALSO). Tier 2 documents getifaddrs / SIOCGIFMEDIA / ifconfig.

## 18. Panic-path apply blockers (2026-08-31)

T021. Claude Reviewer blocked pkg again after PR 16 (`3553f99`). Four
panic/rescue land bugs. Dry-run / `--yes` gates unchanged. PR 16 FreeBSD
carrier and real MCP apply executor kept.

### A. Playbook shell session (ROOTDS)

`stepsFromCommands` still emits one step per stored line
(`Argv []string{line}`). `SysExecutor` now keeps one `/bin/sh` session
so `ROOTDS=/export/...` is visible to a later `printf '%s\n' "$ROOTDS"`.
`TestSysExecutor_PersistsShellEnvAcrossSteps` and
`TestPrintApply_ShellEnvPersists` use a real executor, not
`CountingExecutor` (which returns `ok` and never evaluates the
assignment). Panic-path y/y can remount ZFS RO root.

### B. Failed land is not success

`apply.Execute` sets `Applied=false` and returns `ErrStepFailed` when any
step fails. `hawkeye apply` still prints the result JSON and exits
non-zero. TTY `printApply` does not print `applied`.
`TestExecute_AnyStepFailureIsNotApplied`,
`TestApply_StepFailureExitsNonZero`,
`TestConsult_TTY_FailedLandDoesNotClaimApplied`.

### C. RO missing /var must still land

`applyAuditor` ModeApply no longer refuses when `MkdirAll` of the audit
dir fails (default `/var/log/hawkeye/audit.log`). It degrades to
`NopAuditor` with a stderr note. Dry-run already did this.
`TestApply_ROMissingVarStillLands`,
`TestPrintApply_ROMissingVarStillLands`.

### D. File apply redacts after parse

`hawkeye apply plan.json` `json.Unmarshal`s first, then `redactPlan` by
field. A compact plan with `password: fake-password-for-tests-only` still
unmarshals. TTY consult was already field-safe.
`TestApply_PasswordKVStillUnmarshals` (FAKE fixture only).

Red (before the fix):

```
# apply.SysExecutor{}.Close undefined; applyAuditor arity
# TestSysExecutor_PersistsShellEnvAcrossSteps would fail:
#   ROOTDS gone after per-step /bin/sh -c
# TestExecute_StepErrorRecorded: Applied=true, err=nil
# TestApply_PasswordKVStillUnmarshals: plan JSON: unexpected end / invalid
# TestApplyAuditor_MkdirFail: ModeApply returned error (blocked land)
# TestPrintApply_StepError: code=0 and "applied"
```

Green: `CGO_ENABLED=0 go test ./internal/... ./cmd/hawkeye -count=1 -coverprofile=coverage.out` PASS.

```
ok  github.com/revytechinc/hawkeye/internal/apply      coverage: 98.0%
ok  github.com/revytechinc/hawkeye/internal/cli        coverage: 84.3%
ok  github.com/revytechinc/hawkeye/internal/consult    coverage: 96.8%
ok  github.com/revytechinc/hawkeye/internal/redact     coverage: 100.0%
ok  github.com/revytechinc/hawkeye/internal/probe      coverage: 91.7%
total: (statements) 88.7%
```

`apply.Execute` 100%. `apply.ResolveMode` 100%. `runShellLine` 100%.
`applyAuditor` 100%. `redactPlan` 100%. `mcpApply` 100%.
`consult.CommandLines` 100%. Redact 100%. FAKE fixtures only.

`--check-config` on `configs/config.example.json`: exit 0.
`hawkeye apply` without `--yes`: `"dry_run": true`, `"applied": false`.
`hawkeye apply --dry-run --yes`: `"dry_run": true` (`--dry-run` wins).
`hawkeye --json doctor` (no kit): UNHEALTHY, GPU absent ok. Exit 1.

`CGO_ENABLED=0 go build -buildvcs=false ./cmd/hawkeye` succeeded.
`GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0 go build` succeeded.

`mandoc` not installed here. Equivalent mdoc lint: required macros
present (Dd Dt NAME SYNOPSIS DESCRIPTION COMMANDS OPTIONS SIGNALS
FILES SEE ALSO). apply(8) documents one-shell playbooks, failed-step
exit, and RO audit degrade.

## 19. Host first-look before session `>` (2026-08-31)

T022: bare TTY `hawkeye` prints host first-look (human text) before `>`.
Not `doctor` (pidfile/config/knowledge stay service self-health).
Composed with T021 (A–D apply blockers): failed land still exits 1
and does not print `applied` or re-inspect.

Red (before `probe.Inspect`):

```
# github.com/revytechinc/hawkeye/internal/probe_test
internal/probe/inspect_test.go:21:15: undefined: probe.Inspect
undefined: probe.Sources
undefined: probe.DiskUse
FAIL	github.com/revytechinc/hawkeye/internal/probe [build failed]
```

Green: `CGO_ENABLED=0 go test ./internal/... ./cmd/hawkeye -count=1 -coverprofile=coverage.out` PASS.

`probe.Inspect` 100%. `Report.Human` 100%. `Report.JSON` 100%.
`inspectDisk` 100%. `hostInspect` 100%. Tests use FAKE fixtures only:
fake fstab+mount table, missing `*_enable` script/binary, RO root,
full disk + inodes, no carrier, degraded zpool, geli UNAVAIL.
Healthy fixtures are silent. Human output has no JSON keys and no
hawkeye pidfile.

Doctor still reports pidfile/config — not host first-look.
`hawkeye --json inspect` / bare `--json`: structured findings; no REPL.
Session documents host first-look before `>`.
`inspect` is diagnose-only and is not `doctor`.

## 20. inspect expands rc.subr `${name}` (2026-08-31)

T023: live jail first-look printed
`sshd_enable=YES but /usr/sbin/${name} is missing` while `/usr/sbin/sshd`
exists. Stock rc.d uses `command="/usr/sbin/${name}"`; rc.subr expands
`${name}` / `$name` from the script basename. Inspect did not.

Red (`TestInspect_RCExpandsNameLikeRCSubr` before expand):

```
--- FAIL: TestInspect_RCExpandsNameLikeRCSubr
    inspect_test.go: rc.subr expands name; must not report the unexpanded path:
        sshd_enable=YES but /usr/sbin/${name} is missing; restore the binary or disable sshd
        ntpd_enable=YES but /usr/sbin/$name is missing; restore the binary or disable ntpd
--- FAIL: TestInspect_HealthyIsSilent
    inspect_test.go: healthy host must be silent:
        sshd_enable=YES but /usr/sbin/${name} is missing; restore the binary or disable sshd
--- FAIL: TestKnowledgePaths_OverrideIsExclusive
    knowledge_paths_test.go: HAWKEYE_KNOWLEDGE_PATH must isolate from live kit paths:
        ["/tmp/kit-fixture" "/boot/hawkeye" "/usr/local/share/hawkeye" ...]
```

Green: `CGO_ENABLED=0 go test ./internal/... ./cmd/hawkeye -count=1 -coverprofile=coverage.out` PASS.

```
expandRCSubrName   100.0%
expandDollarName   100.0%
isRCIdent          100.0%
rcCommand          100.0%
rcBinaryMissing    100.0%
knowledgePaths     100.0%
probe package      87.6%
cli package        84.1%
redact             100.0%
total              88.0%
```

`$namespace` is not treated as `$name`. Unexpanded `$` vars do not
false-positive a missing binary. Diagnose only; services are not started.

Plan tests `TestPlan_RORootUnlockRW`, `TestPlan_DefaultIsHumanNotJSON`,
`TestPlan_JSONFlagDumpsMachineObject` open a FAKE `CreateTestDB` kit via
`HAWKEYE_KNOWLEDGE_PATH`. That env now overrides compiled/XDG paths so
live jail FTS cannot rank rc-enable-missing over unlock-rw.

`--check-config` on `configs/config.example.json`: exit 0.
`hawkeye doctor` (no kit): UNHEALTHY, dependencies FAIL, GPU absent ok.
Exit 1. Human + JSON.

`CGO_ENABLED=0 go build -buildvcs=false ./cmd/hawkeye` succeeded.
`GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0 go build` succeeded.

`mandoc` not installed here. Equivalent mdoc lint: required macros
present (Dd Dt NAME SYNOPSIS DESCRIPTION COMMANDS OPTIONS SIGNALS
FILES KEYS ENVIRONMENT SEE ALSO). `${name}` / `$name` documented in
`hawkeye(8)`; `HAWKEYE_KNOWLEDGE_PATH` override documented in
`hawkeye.conf(5)`.

## 21. CORE: local llama.cpp, update skip, rescue layout (2026-08-31)

T009 / T012 / T024. Product binary only. No hawkeye-www. No vendored
GGUF or hawkeye-data sqlite. Tests do not remount live ZFS.

### A. Local LLM is inference, not a skeleton

`Local.Complete` execs a configured llama.cpp-style binary (`-m`, `-p`,
`--no-display-prompt`, `-ngl 99` GPU / `-ngl 0` CPU). Prompts are
redacted first. `headroom.Allow()` still gates the job.

Red (before the exec, against the `fmt.Sprintf("local %s skeleton")` path):

```
--- FAIL: TestLocal_FakeBinaryCapturesPromptAndReturnsCanned
    llm_test.go: text="local llama.cpp skeleton; GPU=false" want canned
--- FAIL: TestConsult_LocalCompletionAfterPlaybook
    consult_llm_test.go: local completion missing after playbook hits
```

Green: fake `llama-cli` captures argv and returns canned text. Secret
`password=fake-password-for-tests-only` is not in argv or response.
No model → `ErrNoModel` (TTY quiet; `--json` still notes `llm skipped`).
GPU required missing → `ErrGPURequired`. PATH look-up finds `llama-cli`.
Consult TTY prints the completion after the remount playbook.

`llm.Complete` 95.0%. `resolveBin` 100%. `cliArgs` 100%. `invoke` 100%.

### B. update skip so rc start is healthy

Red:

```
--- FAIL: TestRun_EmptySourceSkips
    update_test.go: unset source must skip with no error (rc start):
        update source and destination are required
--- FAIL: TestUpdate_NoSourceSkipsHealthy
    update_skip_test.go: rc start must not log missing src/dest
```

Green: empty source returns `("", nil)`. Dest defaults to
`/usr/local/share/hawkeye/knowledge.sqlite`. `HAWKEYE_UPDATE_SOURCE`
copies when set. RO + set source still refuses (unlock-rw). Live
`hawkeye update` with no env: exit 0.

`update.ResolveDest` 100%. `update.Run` 75.0% (copy/rename IO faults
need a live dest we will not write).

### C. Rescue layout

`make install-rescue DESTDIR=$STAGE` created `$STAGE/rescue/hawkeye`
(executable) and `$STAGE/boot/hawkeye` without a live `/boot`.
Port option `RESCUE` is in `ports/sysutils/hawkeye/Makefile`.
Path order: `HAWKEYE_KNOWLEDGE_PATH` exclusive; default
`/boot/hawkeye` then `/usr/local/share/hawkeye`.

`SearchPaths` 100%. `knowledgePaths` covered by exclusive + order tests.

Green: `CGO_ENABLED=0 go test ./internal/... ./cmd/hawkeye -count=1 -coverprofile=coverage.out` PASS.

```
ok  github.com/revytechinc/hawkeye/internal/llm        coverage: 95.3%
ok  github.com/revytechinc/hawkeye/internal/update     coverage: 77.8%
ok  github.com/revytechinc/hawkeye/internal/cli        coverage: 85.0%
ok  github.com/revytechinc/hawkeye/internal/knowledge  coverage: 87.2%
ok  github.com/revytechinc/hawkeye/internal/consult    coverage: 96.8%
ok  github.com/revytechinc/hawkeye/internal/redact     coverage: 100.0%
total: (statements) 88.6%
```

`--check-config` on `configs/config.example.json`: exit 0.
`hawkeye update` (no source): exit 0, no src/dest error.
`hawkeye --json doctor` (no kit): UNHEALTHY, knowledge missing, GPU
absent ok. Exit 1.

`CGO_ENABLED=0 go build -buildvcs=false ./cmd/hawkeye` succeeded.
`GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0 go build` succeeded.

`mandoc` not installed here. Equivalent mdoc lint: required macros
present on `hawkeye.8` (Dd Dt NAME SYNOPSIS DESCRIPTION COMMANDS OPTIONS
SIGNALS FILES SEE ALSO). `hawkeye.conf.5` has NAME DESCRIPTION KEYS
ENVIRONMENT SEE ALSO. Rescue paths `/rescue/hawkeye` and `/boot/hawkeye`
are in `hawkeye.8`. `HAWKEYE_LLM_MODEL`, `HAWKEYE_LLM_BIN`, and
`HAWKEYE_UPDATE_SOURCE` are in `hawkeye.conf(5)`.

### D. Live product jail facts (hawkeye.revytechinc.com)

Installed SHA eaf77537: `hawkeye_enable=YES hawkeye_mcp=YES hawkeye_update=YES`.
Empty src/dest made every start log `hawkeye update failed (continuing)`.
Skip-if-no-src is the rc-healthy path; dest defaults to
`/usr/local/share/hawkeye/knowledge.sqlite`.

`llm.local.backend=llama.cpp`, empty `model_path`, `prefer_gpu=true`.
No llama-cli/server, no GGUF. `misc/llama-cpp` exists in ports and is
not a RUN_DEPENDS. Completer runs only when bin+model are configured.
No model → `ErrNoModel` (quiet TTY). `/dev/nvidia0` present,
`gpu_vram_free_bytes` null, ~129GiB RAM: CPU job is not blocked.

`/rescue` is a dangling bastille symlink; `/boot` exists without
`/boot/hawkeye`. `CanStageRescue` skips the dangling symlink; DESTDIR
still stages. `make install-rescue` with a fixture dangling symlink
printed `skip … (not a real directory)` and created `/boot/hawkeye`.
DESTDIR stage still wrote `/rescue/hawkeye`. Tests do not remount ZFS
`/`. knowledge embeddings table (0 rows) is unused; no corpus vendored.

## 22. install-rescue skip RO /boot (2026-08-31)

T025. Product jail after pkg of SHA c5e6414b: `make install-rescue`
skipped dangling `/rescue` then died exit 71 on
`mkdir /boot/hawkeye` EROFS. Jail `/boot` is a bastille symlink to a
read-only release boot. Do not remount.

Red (before `StageBootKit` / Makefile skip):

```
# undefined: knowledge.StageBootKit
# undefined: knowledge.IsReadOnlyCreateError
# Makefile missing EROFS/EACCES/EPERM skip
--- FAIL: TestMakefileInstallRescue
    layout_test.go: Makefile must DESTDIR-stage, skip dangling /rescue,
    and skip RO /boot (EROFS missing)
```

Green: `CGO_ENABLED=0 go test ./internal/... ./cmd/hawkeye -count=1 -coverprofile=coverage.out` PASS.

```
IsReadOnlyCreateError  100.0%
BootKitSkipMessage     100.0%
StageBootKit           100.0%
CanStageBootKit        100.0%
CanStageRescue         100.0%
realDir                100.0%
knowledge package      89.4%
redact                 100.0%
total                  88.7%
```

Fake RO dest (chmod 0555 `/boot`) skips with
`install-rescue: skip … (read-only)` and exit 0. Injected EROFS, EACCES,
and EPERM skip; ENOSPC still fails. DESTDIR stage still creates both
`/rescue/hawkeye` and `/boot/hawkeye`. DESTDIR + RO dest is an error, not
a live skip. Tests do not remount ZFS or the jail `/boot`.

`--check-config` on `configs/config.example.json`: exit 0.
`hawkeye --json doctor` (no kit): UNHEALTHY, knowledge missing, GPU
absent ok. Exit 1.

`CGO_ENABLED=0 go build -buildvcs=false ./cmd/hawkeye` succeeded.
`GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0 go build` succeeded.

`mandoc` not installed here. Equivalent mdoc lint: required macros
present on `hawkeye.8` (Dd Dt NAME SYNOPSIS DESCRIPTION COMMANDS OPTIONS
SIGNALS FILES SEE ALSO). RO `/boot` skip is documented. No hawkeye-www.

## 23. sqlite-vec / vector search when RAM allows (2026-08-31)

T026. CORE hole: `embeddings` table has 0 rows in the current jail kit.
FTS must still work. When FLOAT32 rows exist and RAM allows, consult
ranks with sqlite-vec (`modernc.org/sqlite/vec` v0.1.9). No cloud embed
API. No GGUF vendored. Consult stays read-only. Runtime fill of existing
chunks uses `OpenRW` + a local embedder only.

Red (before PackF32 / Search vector merge / Local.Embed):

```
# github.com/revytechinc/hawkeye/internal/knowledge_test
undefined: knowledge.UnpackF32
undefined: knowledge.PackF32
undefined: knowledge.DistanceCosine
undefined: knowledge.InsertEmbedding
st.Vec undefined
st.QueryVec undefined
# github.com/revytechinc/hawkeye/internal/llm_test
l.Embed undefined
unknown field EmbedModelPath
# github.com/revytechinc/hawkeye/internal/config_test
c.LLM.Local.EmbedModelPath undefined
FAIL
```

Green: `CGO_ENABLED=0 go test ./internal/... ./cmd/hawkeye -count=1 -coverprofile=coverage.out` PASS.

```
PackF32            100.0%
UnpackF32          100.0%
DistanceCosine     100.0%
useVectors         100.0%
searchVectors      100.0%
vectorHits         100.0%
vectorHitsSQL      100.0%
lookupTarget       100.0%
mergeHits          100.0%
probeVectors       100.0%
vecAvailable       100.0%
FakeEmbedder       100.0%
Local.Embed        100.0%
Local.Model        100.0%
parseEmbedding     100.0%
attachSearch       100.0%
openKnowledge      100.0%
knowledge package  90.7%
llm package        98.0%
redact             100.0%
total              89.3%
```

Tiny FAKE vectors only (dim 3). Empty `embeddings` keeps FTS order
(`boot environment` still leads with the BE playbook). Query vec
`[0.99,0.01,0]` matching remount `[1,0,0]` promotes
`Remount ZFS root read-write` and still attaches stored commands.
Low RAM (`ram_min_free_bytes` above free) stays FTS-only, no error.
No embedder + rows present stays FTS-only. Secrets are redacted before
the embedder. TTY has no `vec_distance` / embeddings chrome.

`--check-config` on `configs/config.example.json`: exit 0.
`hawkeye --json doctor` (no kit): UNHEALTHY, knowledge missing, GPU
absent ok. Exit 1. Human + JSON.

`CGO_ENABLED=0 go build -buildvcs=false ./cmd/hawkeye` succeeded.
`GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0 go build` succeeded.

`mandoc` not installed here. Equivalent mdoc lint: required macros
present. `llm.local.embed_model_path` and `HAWKEYE_EMBED_MODEL` are in
`hawkeye.conf(5)`. Consult sqlite-vec rank is in `hawkeye(8)`.
No hawkeye-www. Tests do not remount ZFS `/`. No GGUF vendored.

## 24. install-rescue bmake set -e mkdir (2026-08-31)

T027. Product jail after PR 21 (`0a43fb4ed536`): stock
`make install-rescue` died **exit 1** (not 71) at
`_boot_err=$(mkdir …)`. bmake recipes run with `set -e`, so EROFS
mkdir aborted the recipe before skip. GNU make on CI hid this.
Do not remount.

Red (stock PR 21 recipe extracted and run under `set -e`):

```
--- FAIL: TestMakefileInstallRescue_SetEDoesNotAbortOnROMkdir
    layout_stage_test.go: Makefile mkdir capture must survive set -e
    on RO dest (bmake): exit status 1
--- FAIL: TestMakefileInstallRescue
    layout_test.go: bmake recipes run with set -e; mkdir status must
    be captured with || so RO skip runs
```

Green: `CGO_ENABLED=0 go test ./internal/... ./cmd/hawkeye -count=1 -coverprofile=coverage.out` PASS.

```
TestUnguardedMkdirCommandSubDiesUnderSetE     PASS
TestMakefileInstallRescue_SetEDoesNotAbortOnROMkdir PASS
TestMakefileInstallRescue_FakeRODestExit0     PASS
TestMakefileInstallRescue_ExistingFileFails   PASS
TestMakefileInstallRescue_DESTDIRCreatesBoth  PASS
TestMakefileInstallRescue_DESTDIRRODestFails  PASS
IsReadOnlyCreateError  100.0%
BootKitSkipMessage     100.0%
StageBootKit           100.0%
CanStageBootKit        100.0%
CanStageRescue         100.0%
realDir                100.0%
redact                 100.0%
```

Live mkdir uses `_boot_rc=0; _boot_err=$(mkdir …) || _boot_rc=$?`.
Unguarded `$(mkdir)` is asserted to die under `set -e` so GNU make
cannot hide the trap. `/rescue` has no command-sub mkdir; any future
`$(mkdir)` must use `||`. DESTDIR still creates both prefixes.
EEXIST (file in the way) and DESTDIR + RO dest still fail the target.
Tests do not remount ZFS or the jail `/boot`.

`--check-config` on `configs/config.example.json`: exit 0.
`hawkeye --json doctor` (no kit): UNHEALTHY, knowledge missing, GPU
absent ok. Exit 1.

`CGO_ENABLED=0 go build -buildvcs=false ./cmd/hawkeye` succeeded.
`GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0 go build` succeeded.

`mandoc` not installed here. Equivalent mdoc lint: required macros
present on `hawkeye.8` (Dd Dt NAME SYNOPSIS DESCRIPTION COMMANDS OPTIONS
SIGNALS FILES SEE ALSO). `hawkeye.conf.5` has NAME DESCRIPTION KEYS
ENVIRONMENT SEE ALSO. No hawkeye-www.

## 25. Operator-readable daemon pidfile 0644 (2026-08-31)

T028. Live jail/host: unprivileged `hawkeye doctor` was UNHEALTHY because
`/var/run/hawkeye.pid` was mode `0600` `root:wheel` (permission denied).
Root doctor was healthy (root can read 0600). PR 7 `start_postcmd` chmod
races `daemon(8) -p` (`pidfile_open(..., 0600)` when the path is missing,
or `-f` stays in the foreground so poststart never runs).

Red:

```
# github.com/revytechinc/hawkeye/internal/pidfile_test
internal/pidfile/pidfile_test.go:43:14: undefined: pidfile.OperatorReadable
# github.com/revytechinc/hawkeye/internal/doctor_test
internal/doctor/doctor_test.go:102:3: unknown field PidMode
--- FAIL: TestDoctor_Mode0600UnhealthyEvenIfReadable
    more_test.go:145: 0600 pidfile must be unhealthy so root doctor
    catches the operator regression
--- FAIL: TestRcd_DaemonPidfileLeftReadable0644
    rcd_test.go: rc.d must have hawkeye_pidfile_operator_readable
```

Green: `CGO_ENABLED=0 go test ./internal/... ./cmd/hawkeye -count=1 -coverprofile=coverage.out` PASS.

```
ok  github.com/revytechinc/hawkeye/internal/apply     coverage: 98.0%
ok  github.com/revytechinc/hawkeye/internal/doctor    coverage: 96.1%
ok  github.com/revytechinc/hawkeye/internal/pidfile   coverage: 86.4%
ok  github.com/revytechinc/hawkeye/internal/redact    coverage: 100.0%
OperatorReadable  100.0%
Write             100.0%
Remove            100.0%
total             89.4%
```

`TestResolveMode_DefaultIsDryRun` still default dry-run. Apply was not
changed. rc.d seeds 0644 (`umask 022 && : >`) before `daemon -p` and
chmods again in poststart. Doctor reports `pidfile is not world-readable
(mode 0600)` when root can read a 0600 file. 0644 + knowledge is healthy.
Unreadable (permission denied) is still `unreadable`, not `empty`.

`--check-config` (no file): exit 0, defaults.
`hawkeye --json doctor` (no kit): UNHEALTHY, knowledge missing, pidfile
not required, GPU absent ok. Exit 1.

`CGO_ENABLED=0 go build -buildvcs=false ./cmd/hawkeye` succeeded.
`GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0 go build` succeeded.

`mandoc` not installed here. Equivalent mdoc lint: required macros
present on `hawkeye.8` (Dd Dt NAME SYNOPSIS DESCRIPTION COMMANDS OPTIONS
SIGNALS FILES SEE ALSO; Xr daemon 8). `hawkeye.conf.5` has NAME
DESCRIPTION KEYS ENVIRONMENT SEE ALSO. No hawkeye-www.

## 26. Operator GGUF discover, embed CLI, dangling /rescue (2026-08-31)

T029–T031. Product jail: empty `llm.local.model_path` never fired
`Local.Complete` even after a GGUF was on disk. Kit `embeddings` stayed
empty (FillEmbeddings existed, no CLI). Thin Bastille `/rescue` is often
a dangling symlink; `make install-rescue` skipped it.

Red (before discover / embed / Makefile replace):

```
# github.com/revytechinc/hawkeye/internal/llm_test
undefined: llm.DiscoverModel
undefined: llm.ResolveModel
# github.com/revytechinc/hawkeye/internal/cli_test
unknown field Embedder in struct literal of type cli.Env
--- FAIL: TestRun_MissingOptionalGGUFIsNoteNotFail
    doctor must note optional local GGUF
--- FAIL: TestCanStageRescue_DanglingSymlinkAllowed
    dangling bastille /rescue must be replaceable
--- FAIL: TestMakefileInstallRescue_DanglingSymlinkWritesBinary
    install-rescue: skip …/rescue (not a real directory)
--- FAIL: TestMakefileInstallRescue_SymlinkToRealRescueInstallsInto
    skip dangling /rescue (did not install into the real image)
```

Green: `CGO_ENABLED=0 go test ./internal/... ./cmd/hawkeye -count=1 -coverprofile=coverage.out` PASS.

```
ok  github.com/revytechinc/hawkeye/internal/apply      coverage: 98.0%
ok  github.com/revytechinc/hawkeye/internal/cli        coverage: 85.2%
ok  github.com/revytechinc/hawkeye/internal/doctor     coverage: 96.3%
ok  github.com/revytechinc/hawkeye/internal/knowledge  coverage: 90.9%
ok  github.com/revytechinc/hawkeye/internal/llm        coverage: 97.2%
ok  github.com/revytechinc/hawkeye/internal/redact     coverage: 100.0%
DiscoverModel          100.0%
ResolveModel           100.0%
ResolveEmbedModel      100.0%
DefaultModelDirs       100.0%
CanStageRescue         100.0%
RescueSkipReadOnly     100.0%
resolveLocalModels     100.0%
cmdEmbed                77.4%
total                  89.4%
```

`TestConsult_AutoDiscoversGGUFWithoutJSONEdit`: dropped `*.gguf` under
`HAWKEYE_MODELS_DIR` plus `PATH`/`HAWKEYE_LLM_BIN` llama-cli prints
playbook prose and the canned local-complete paragraph. Missing GGUF
stays a quiet TTY skip (`TestConsult_NoModelQuietTTY`). `--json` may
still note `llm skipped`. MCP consult stays `llm.None`.

`hawkeye embed` default is dry-run (no `embeddings` rows, embedder not
invoked). `--yes` fills a writable kit via FakeEmbedder.
`--dry-run` wins over `--yes`. Read-only dest is refused. Secrets are
not printed. Consult `Open` stays read-only.

`make install-rescue`: dangling `/rescue` becomes a real directory with
executable `/rescue/hawkeye`. A real `/rescue` (or symlink to a writable
rescue image) is installed *into*; existing tools stay. EROFS/EACCES
skips with `install-rescue: skip … (read-only)`. DESTDIR still stages
both prefixes. bmake `|| _rescue_rc=$?` / `|| _boot_rc=$?`. No remount.

`--check-config` (no file): exit 0, defaults.
`--check-config` on `configs/config.example.json`: exit 0 (empty
`model_path` still valid).
`hawkeye --json doctor` (no kit): UNHEALTHY, knowledge missing,
`local_llm` ok `optional local GGUF missing (consult skips quietly)`,
GPU absent ok. Exit 1.
`TestResolveMode_DefaultIsDryRun` still default dry-run.

`CGO_ENABLED=0 go build -buildvcs=false ./cmd/hawkeye` succeeded.
`GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0 go build` succeeded.

`mandoc` not installed here. Equivalent mdoc lint: required macros
present on `hawkeye.8` (Dd Dt NAME SYNOPSIS DESCRIPTION COMMANDS OPTIONS
SIGNALS FILES SEE ALSO; embed command; install-rescue dangling replace).
`hawkeye.conf.5` has NAME DESCRIPTION KEYS ENVIRONMENT SEE ALSO;
`HAWKEYE_MODELS_DIR` and well-known `models/` discover. No hawkeye-www.
No GGUF vendored. Tests do not remount ZFS `/`.

## 27. Jail GPU null VRAM is CPU; /boot symlink stages kit (2026-08-31)

T032. Product jail hawkeye.revytechinc.com (FreeBSD 16-CURRENT,
hawkeye-0.1.0_9): `doctor` reports `gpu=true` and
`gpu_vram_free_bytes=null`. Live config `prefer_gpu=true`. Treating
`gpu_present` as `-ngl 99` makes llama-cli fail and consult skips.
`/rescue` → `/.bastille/rescue` is dangling (touch ENOENT).
`/boot` → `/.bastille/boot` is a live writable symlink.
Well-known model dir: `/usr/local/share/hawkeye/models/*.gguf`.

Red:

```
--- FAIL: TestLocal_JailLikeGPUNullVRAMUsesCPUNotSkip
    null VRAM is not usable GPU
--- FAIL: TestLocal_GPUInvokeFailFallsBackToCPU
    GPU fail must fall back to CPU, not skip: cuda error: no usable VRAM
--- FAIL: TestConsult_NullVRAMStillCompletesOnCPU
    CPU complete missing after null-VRAM GPU present
--- FAIL: TestCanStageBootKit_SymlinkToRealBootAllowed
    live /boot symlink to a real boot image must allow creating /boot/hawkeye
```

Green: `CGO_ENABLED=0 go test ./internal/... ./cmd/hawkeye -count=1` PASS.

```
gpuUsable              100.0%
resolvedDir            100.0%
CanStageBootKit        100.0%
Complete                95.8%
llm package            96.1%
redact                100.0%
total                  89.4%
```

Null VRAM + prefer_gpu + nvidia0 uses `-ngl 0` and returns completion.
GPU invoke failure retries `-ngl 0` (unless require_gpu). Consult TTY
still prints playbook + local-complete. `make install-rescue` on a
dangling `/rescue` plus `/boot` symlink creates `/rescue/hawkeye` and
`/boot/hawkeye` through the live boot link without replacing a real
rescue image.

`--check-config` defaults: exit 0.
`TestResolveMode_DefaultIsDryRun` unchanged.
No GGUF vendored. No remount. No hawkeye-www.

## 28. llama-cli one-shot; prefer llama-completion (2026-08-31)

T033. Jail proof (hawkeye-0.1.0_9 SHA 52960eed, llama-cpp-9426_1):
Complete fires with a dropped GGUF. `llama-cli` 9426 is conversation-only
and hangs on `>` without `--single-turn`. Jail workaround:
`llm.local.bin=/usr/local/bin/llama-completion`. Models at
`/usr/local/share/hawkeye/models/*.gguf`.

Red:

```
--- FAIL: TestLocal_LlamaCLIGetsSingleTurn
    llama-cli 9426 is conversation-only; must pass --single-turn
--- FAIL: TestLocal_LookPathPrefersLlamaCompletion
    PATH must prefer llama-completion over llama-cli: "from-cli"
--- FAIL: TestLocal_LlamaCLIWithoutSingleTurnIsHangFailure
    conversation-mode hang is a product failure
--- FAIL: TestLocal_StripsLlamaChatLeftovers
    TTY must not show llama chat leftovers: "...> EOF by user\nExiting..."
```

Green: `CGO_ENABLED=0 go test ./internal/... ./cmd/hawkeye -count=1` PASS.
Hang fixture (`cat` without `--single-turn`) is a test failure; with
`--single-turn` Complete returns immediately. `llama-completion` argv
has no `--single-turn`. Leftovers stripped. PATH prefers
`llama-completion`.

`--check-config` defaults: exit 0.
`TestResolveMode_DefaultIsDryRun` unchanged.
No GGUF vendored. No hang of the panic session. No hawkeye-www.

CI on `0a8834c` failed `TestLocal_JailLikeGPUNullVRAMUsesCPUNotSkip` because
`strings.Contains(capture, "99")` matched a digit sequence in the temp
path while argv was `-ngl 0`. Assertions now match the `-ngl` argument
exactly (`nglIs`).

## 29. Embed uses llama-embedding, not llama-completion (2026-08-31)

T034. Product jail hawkeye.revytechinc.com: hawkeye-data-0.1.0_4 kit
has 16 nomic playbook embeddings (dim=768, length(vector)=3072,
model=nomic-embed-text-v1.5.Q8_0.gguf). Consult still ranked FTS5
BM25. Cause: llama-cpp-9426_1. `Embed()` reused `resolveBin(l.Bin)`
(`/usr/local/bin/llama-completion`) plus `--embedding
--no-display-prompt`. Facts: `llama-cli --embedding` is invalid;
`llama-embedding` exists and works; it rejects `--no-display-prompt`
and does not want `--embedding`; default newline embd-separator
splits playbooks (dim tens of thousands). Working argv:
`--embd-output-format array --pooling mean --embd-separator '<#sep#>'`.

Red (before resolveEmbedBin / embedArgs change):

```
--- FAIL: TestLocal_EmbedLlamaEmbeddingRejectsChatFlags
    llama-embedding 9426 rejects --embedding and --no-display-prompt:
    local llm llama-embedding: exit status 1
--- FAIL: TestLocal_EmbedPrefersLlamaEmbeddingNotCompletion
    no embedding floats parsed
--- FAIL: TestLocal_EmbedSiblingLlamaEmbeddingWhenPATHEmpty
    no embedding floats parsed
--- FAIL: TestLocal_EmbedDoesNotReuseLlamaCLI
    llama-cli 9426 cannot embed; missing llama-embedding must skip: <nil>
--- FAIL: TestLocal_EmbedParsesSmallArray
    local llm llama-embedding: exit status 1
```

Green: `CGO_ENABLED=0 go test ./internal/... ./cmd/hawkeye -count=1 -coverprofile=coverage.out` PASS.

```
Embed                  100.0%
embedArgs              100.0%
parseEmbedding         100.0%
resolveEmbedBin        100.0%
isEmbedBin             100.0%
resolveNamed           100.0%
Complete               95.8%
llm package            97.0%
redact                 100.0%
total                  89.5%
```

`llama-completion` is not invoked for Embed when `llama-embedding` is
on PATH or beside the completer. `--no-display-prompt` and
`--embedding` are not passed. Missing `llama-embedding` is
`ErrNoBinary` (consult FTS). Small array and 768-float array parse.
Complete still uses `llm.local.bin`. GPU embed fail retries `-ngl 0`.
`TestResolveMode_DefaultIsDryRun` unchanged.

`--check-config` (no file): exit 0, defaults.
`--check-config` on `configs/config.example.json`: exit 0.
`hawkeye --json doctor` (no kit): UNHEALTHY, knowledge missing,
`local_llm` ok optional GGUF missing, GPU absent ok. Exit 1.

`CGO_ENABLED=0 go build -buildvcs=false ./cmd/hawkeye` succeeded.
`GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0 go build` succeeded.

`mandoc` not installed here. Equivalent mdoc lint: required macros
present on `hawkeye.8` (Dd Dt NAME SYNOPSIS DESCRIPTION COMMANDS OPTIONS
SIGNALS FILES SEE ALSO; llama-embedding query-time rank).
`hawkeye.conf.5` has NAME DESCRIPTION KEYS ENVIRONMENT SEE ALSO;
`llm.local.bin` is the completer only. No hawkeye-www. No GGUF vendored.
Consult stays read-only. LLM never execs as root.

## 30. sysctl(8) kern.securelevel host overlay (2026-09-01)

T010. Doctor now always reports a `securelevel` check. Known values
come from `unix.SysctlUint32` on FreeBSD, then `sysctl(8) -n`
(`/sbin/sysctl`, `/usr/sbin/sysctl`, `/rescue/sysctl`). Unknown is a
note, not a failure. MIB names are restricted to letters, digits, `.`,
and `_`. Verbose `sysctl` output without `-n` is rejected.

Red (before overlay + doctor check):

```
--- FAIL: TestSysctl8Int_UsesInjectedRunner
    undefined: probe.Sysctl8Int
--- FAIL: TestRun_SecurelevelKnownIsNoteNotFail
    missing securelevel check
--- FAIL: TestRun_SecurelevelUnknownIsNoteNotFail
    missing securelevel check
```

Green: `CGO_ENABLED=0 go test ./internal/... ./cmd/hawkeye -count=1 -coverprofile=coverage.out` PASS.

```
ParseSysctlN           90.0%
safeMIB                85.7%
Sysctl8Int             88.9%
liveSysctl8Int         100.0%
doctor.Run             95.9%
doctor.JSON            100.0%
doctor.Human           100.0%
redact                 100.0%
total                  89.4%
```

`--check-config` (no file): exit 0, defaults.
`--check-config` on `configs/config.example.json`: exit 0.
`hawkeye --json doctor` (Linux host, no kit): UNHEALTHY, knowledge
missing, `local_llm` ok optional GGUF missing, GPU absent ok,
`securelevel` ok `kern.securelevel unknown (sysctl(8) not available)`.
Exit 1. High or unknown securelevel must not fail doctor.

`CGO_ENABLED=0 go build -buildvcs=false ./cmd/hawkeye` succeeded.
`GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0 go build` succeeded.

`scripts/e2e-freebsd16.sh` exits 2 on non-FreeBSD (Linux CI skip).
Live FreeBSD 16 product-jail e2e is recorded when the script is run
on that host (no `--yes`).

`mandoc` not installed here. Equivalent mdoc lint: required macros
present on `hawkeye.8` (Dd Dt NAME SYNOPSIS DESCRIPTION COMMANDS OPTIONS
SIGNALS FILES SEE ALSO; doctor `kern.securelevel` via `sysctl(8)`).
`hawkeye.conf.5` has NAME DESCRIPTION KEYS ENVIRONMENT SEE ALSO.
No hawkeye-www. No GGUF vendored. LLM never execs as root.

## 31. Streamable HTTP POST SSE (2026-09-01)

T011: MCP Streamable HTTP POST was JSON-only (GET already SSE). Spec
clients send `Accept: application/json, text/event-stream`. POST now
returns `event: message` with the JSON-RPC object when Accept includes
`text/event-stream`. JSON-only Accept stays JSON. Auth is still required
before any SSE frame. Apply dry-run default unchanged.

Red (`WantsSSE` missing; POST ignored Accept):

```
internal/mcp/sse_test.go:88:10: undefined: mcp.WantsSSE
--- FAIL: TestServeHTTP_POST_SSEWhenAcceptEventStream
    POST with Accept event-stream must be SSE, got "application/json"
```

Green: `go test ./internal/mcp ./internal/cli ./cmd/hawkeye -count=1` PASS.

`WantsSSE` 100%. Apply `ResolveMode` still 100%. Tests use FAKE token
fixtures only.

`--check-config` on `configs/config.example.json`: exit 0.
`hawkeye apply` without `--yes`: `"dry_run": true`.
`CGO_ENABLED=0 go build -buildvcs=false ./cmd/hawkeye` succeeded.

`mandoc` not installed here. Equivalent mdoc lint: `hawkeye(8)` documents
POST SSE when Accept includes `text/event-stream`.

## 32. FreeBSD 16.0-CURRENT e2e (2026-09-01)

Live guest: QEMU TCG, official
`FreeBSD-16.0-CURRENT-amd64-BASIC-CLOUDINIT-ufs` snapshot (2026-08-31),
hostname `hawkeye-e2e`. Nested KVM on this host hits
`kvm_spurious_fault`; TCG boots. `freebsd-version` = `16.0-CURRENT`.
`sysctl -n kern.securelevel` = `-1`.

First doctor read used `unix.SysctlUint32` and reported
`kern.securelevel=4294967295`. Red: `TestSignedSysctl32_SecurelevelMinusOne`.
Fix: prefer `sysctl(8) -n`, interpret 32-bit kernel values as int32.

`scripts/e2e-freebsd16.sh` with
`HAWKEYE=/root/e2e/hawkeye` and
`HAWKEYE_KNOWLEDGE_PATH=/root/e2e/knowledge.sqlite`
(CreatePlaybookTestDB remount playbook). No `--yes`.

```
e2e: FreeBSD 16.0-CURRENT host=hawkeye-e2e
e2e: --check-config ok
"detail": "kern.securelevel=-1 (sysctl(8))"
e2e: doctor reports securelevel
e2e: consult --json ok
e2e: plan --json ok
e2e: apply --dry-run ok
CT=text/event-stream
event: message
data: {"jsonrpc":"2.0","id":1,"result":{...}}
UNAUTH_CODE=401 CT=application/json
MCP_SSE_OK
e2e: MCP POST SSE ok
e2e: PASS
```

Plan steps are stored remount commands (`export PATH=/rescue:...`,
`mount -p`, `zfs set readonly=off "$ROOTDS"`, `mount -u -o rw /`),
not `echo <query>`. Apply JSON has dry-run true. MCP POST with
`Accept: application/json, text/event-stream` is SSE `event: message`.
401 is JSON. FAKE bearer token only. Doctor healthy with the fixture
kit. GPU absent is ok. No hawkeye-www. LLM never execs as root.
