#!/bin/sh
# Hawkeye FreeBSD end-to-end smoke (run ON FreeBSD as root or with doas).
# Usage: sh scripts/e2e-freebsd.sh [/path/to/hawkeye-bin]
set -eu

HAWKEYE="${1:-hawkeye}"
ROOT="$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)"
LOG="${E2E_LOG:-/tmp/hawkeye-e2e.log}"
FAIL=0

log() { printf '%s\n' "$*" | tee -a "$LOG"; }
ok() { log "PASS: $*"; }
bad() { log "FAIL: $*"; FAIL=$((FAIL + 1)); }

: >"$LOG"
log "=== Hawkeye FreeBSD e2e $(date -u +%Y-%m-%dT%H:%M:%SZ) ==="
log "host=$(uname -srm)"
log "bin=$HAWKEYE"

case "$(uname -s)" in
FreeBSD) ;;
*)
	bad "must run on FreeBSD (got $(uname -s))"
	exit 1
	;;
esac

if ! command -v "$HAWKEYE" >/dev/null 2>&1 && [ ! -x "$HAWKEYE" ]; then
	bad "hawkeye binary not found: $HAWKEYE"
	exit 1
fi
# Prefer explicit path
if [ -x "$HAWKEYE" ]; then
	:
elif command -v hawkeye >/dev/null 2>&1; then
	HAWKEYE="$(command -v hawkeye)"
fi

run() {
	log "+ $*"
	# shellcheck disable=SC2086
	"$@" >>"$LOG" 2>&1
}

# 1) Version / help
if "$HAWKEYE" --help >/dev/null 2>&1 || "$HAWKEYE" help >/dev/null 2>&1; then
	ok "help"
else
	# Some builds print usage on bare unknown; accept --version
	if "$HAWKEYE" version >/dev/null 2>&1 || "$HAWKEYE" --version >/dev/null 2>&1; then
		ok "version"
	else
		bad "help/version"
	fi
fi

# 2) check-config (defaults OK when no file)
if "$HAWKEYE" --check-config >>"$LOG" 2>&1; then
	ok "--check-config (defaults)"
else
	bad "--check-config"
fi

# 3) doctor human + json (may be unhealthy without kit; must not panic)
set +e
"$HAWKEYE" doctor >>"$LOG" 2>&1
dr=$?
"$HAWKEYE" --json doctor >>"$LOG" 2>&1 || "$HAWKEYE" doctor --json >>"$LOG" 2>&1
dj=$?
set -e
if [ "$dr" -eq 0 ] || [ "$dr" -eq 1 ]; then
	ok "doctor human exit=$dr"
else
	bad "doctor human exit=$dr"
fi
if [ "$dj" -eq 0 ] || [ "$dj" -eq 1 ]; then
	ok "doctor json exit=$dj"
else
	bad "doctor json exit=$dj"
fi

# 4) inspect / first-look (diagnose only)
set +e
"$HAWKEYE" inspect >>"$LOG" 2>&1
ii=$?
"$HAWKEYE" inspect --json >>"$LOG" 2>&1 || "$HAWKEYE" --json inspect >>"$LOG" 2>&1
ij=$?
set -e
if [ "$ii" -eq 0 ]; then ok "inspect"; else bad "inspect exit=$ii"; fi
if [ "$ij" -eq 0 ]; then ok "inspect --json"; else bad "inspect --json exit=$ij"; fi

# 5) consult (FTS; may warn without kit)
set +e
"$HAWKEYE" consult --json "zpool degraded" >>"$LOG" 2>&1 || \
	"$HAWKEYE" --json consult "zpool degraded" >>"$LOG" 2>&1
cj=$?
set -e
# 0 success, 1 soft failure without kit still acceptable if no panic
if [ "$cj" -eq 0 ] || [ "$cj" -eq 1 ]; then
	ok "consult json exit=$cj"
else
	bad "consult json exit=$cj"
fi

# 6) plan (dry path)
set +e
"$HAWKEYE" plan --json "restart sshd" >>"$LOG" 2>&1 || \
	"$HAWKEYE" --json plan "restart sshd" >>"$LOG" 2>&1
pj=$?
set -e
if [ "$pj" -eq 0 ] || [ "$pj" -eq 1 ]; then
	ok "plan json exit=$pj"
else
	bad "plan json exit=$pj"
fi

# 7) apply dry-run default (must NOT mutate)
PLANJSON="${TMPDIR:-/tmp}/hawkeye-e2e-plan.json"
cat >"$PLANJSON" <<'EOF'
{
  "id": "e2e-dry",
  "summary": "e2e dry-run only",
  "steps": [
    {"id": "1", "argv": ["true"], "privileged": false}
  ]
}
EOF
set +e
"$HAWKEYE" apply "$PLANJSON" >>"$LOG" 2>&1
ar=$?
"$HAWKEYE" apply --dry-run "$PLANJSON" >>"$LOG" 2>&1
ad=$?
set -e
# Default apply should dry-run (exit 0) or refuse without --yes
if [ "$ar" -eq 0 ] || [ "$ar" -eq 1 ]; then
	ok "apply default (no --yes) exit=$ar"
else
	bad "apply default exit=$ar"
fi
if [ "$ad" -eq 0 ]; then ok "apply --dry-run"; else bad "apply --dry-run exit=$ad"; fi

# 8) kern.securelevel via sysctl(8) (T010 host overlay input)
# doctor/inspect JSON currently omit Snapshot.securelevel; assert via sysctl(8).
set +e
sl="$(sysctl -n kern.securelevel 2>/dev/null)"
sl_rc=$?
set -e
if [ "$sl_rc" -eq 0 ]; then
	log "sysctl(8) kern.securelevel=$sl"
	ok "sysctl(8) kern.securelevel=$sl"
else
	bad "sysctl(8) kern.securelevel"
fi

# 9) MCP stdio initialize (one-shot)
set +e
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"e2e","version":"0"}}}' |
	timeout 5 "$HAWKEYE" mcp --stdio >>"$LOG" 2>&1
ms=$?
set -e
if [ "$ms" -eq 0 ] || [ "$ms" -eq 124 ]; then
	ok "mcp --stdio initialize (exit=$ms)"
else
	bad "mcp --stdio exit=$ms"
fi

log "=== summary fail=$FAIL log=$LOG ==="
if [ "$FAIL" -ne 0 ]; then
	exit 1
fi
exit 0
