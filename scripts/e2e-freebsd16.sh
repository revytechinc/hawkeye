#!/bin/sh
# Copyright (c) 2026 REVYTECH, Inc.
# SPDX-License-Identifier: BSD-3-Clause
#
# Product e2e on FreeBSD 14.5 / 15.1 / 16 jails. No --yes (dry-run only).
# Exit 2 if this is not FreeBSD so Linux CI can skip.

set -eu

HAWKEYE="${HAWKEYE:-hawkeye}"
QUERY="${QUERY:-ZFS root is read-only after boot}"
# Optional RO kit directory or knowledge.sqlite path (CreatePlaybookTestDB).
# Consult/plan need a harvest-schema playbook; product jail already has one.
if [ -n "${HAWKEYE_KNOWLEDGE_PATH:-}" ]; then
	export HAWKEYE_KNOWLEDGE_PATH
fi

if [ "$(uname -s)" != "FreeBSD" ]; then
	echo "e2e: not FreeBSD ($(uname -s) $(uname -r)); skip live jail" >&2
	exit 2
fi

echo "e2e: $(uname -s) $(uname -r) host=$(hostname)"
echo "e2e: binary=$HAWKEYE"

"$HAWKEYE" --check-config
echo "e2e: --check-config ok"

doc="$("$HAWKEYE" --json doctor || true)"
echo "$doc" | head -c 4000
echo
echo "$doc" | grep -q '"name": "securelevel"' || {
	echo "e2e: doctor JSON missing securelevel check" >&2
	exit 1
}
echo "$doc" | grep -q 'kern.securelevel unknown' && {
	echo "e2e: FreeBSD doctor must read kern.securelevel via sysctl(8)" >&2
	exit 1
}
echo "$doc" | grep -q 'kern.securelevel=' || {
	echo "e2e: doctor securelevel missing numeric value" >&2
	exit 1
}
# Documented values are -1..3. unix.SysctlUint32 of -1 is 4294967295.
echo "$doc" | grep -Eq 'kern.securelevel=-?[0-3] \(sysctl' || {
	echo "e2e: kern.securelevel must be a signed -1..3, not a uint32 wrap" >&2
	exit 1
}
echo "e2e: doctor reports securelevel"

cons="$("$HAWKEYE" consult --json "$QUERY")"
echo "$cons" | grep -q '"hits"' || {
	echo "e2e: consult --json missing hits" >&2
	exit 1
}
echo "e2e: consult --json ok"

plan="$("$HAWKEYE" plan --json "$QUERY")"
echo "$plan" | grep -q '"steps"' || {
	echo "e2e: plan --json missing steps" >&2
	exit 1
}
if echo "$plan" | grep -q '"echo"'; then
	if echo "$plan" | grep -q "$QUERY"; then
		echo "e2e: plan still echo-stub" >&2
		exit 1
	fi
fi
echo "e2e: plan --json ok"

tmp="$(mktemp -t hawkeye-e2e-plan.XXXXXX)"
trap 'rm -f "$tmp"' EXIT
printf '%s\n' "$plan" >"$tmp"
apply="$("$HAWKEYE" apply --dry-run "$tmp")"
echo "$apply" | grep -q '"dry_run": true\|"dry_run":true' || {
	echo "e2e: apply default must be dry-run" >&2
	exit 1
}
echo "e2e: apply --dry-run ok"

# T011: Streamable HTTP POST SSE. FAKE token only. Loopback. No --yes.
if command -v python3 >/dev/null 2>&1; then
	mcp_token="FAKESECRET_a3b4c5d6e7f8g9h0i1j2"
	export HAWKEYE_MCP_TOKEN="$mcp_token"
	"$HAWKEYE" mcp --http >/tmp/hawkeye-e2e-mcp.out 2>&1 &
	mcp_pid=$!
	trap 'rm -f "$tmp"; kill "$mcp_pid" 2>/dev/null || true' EXIT
	i=0
	while [ "$i" -lt 50 ]; do
		if python3 -c 'import socket; s=socket.socket(); s.settimeout(0.2); s.connect(("127.0.0.1",8741)); s.close()' 2>/dev/null; then
			break
		fi
		i=$((i + 1))
		sleep 0.1
	done
	sse="$(HAWKEYE_MCP_TOKEN="$mcp_token" python3 - <<'PY'
import json, os, urllib.error, urllib.request
token = os.environ["HAWKEYE_MCP_TOKEN"]
body = json.dumps({"jsonrpc":"2.0","id":1,"method":"initialize"}).encode()
req = urllib.request.Request(
    "http://127.0.0.1:8741/mcp",
    data=body,
    headers={
        "Authorization": "Bearer " + token,
        "Accept": "application/json, text/event-stream",
        "Content-Type": "application/json",
    },
    method="POST",
)
with urllib.request.urlopen(req, timeout=5) as r:
    ct = r.headers.get("Content-Type", "")
    data = r.read().decode()
print("CT=" + ct)
print(data)
PY
)" || {
		echo "e2e: MCP POST SSE request failed" >&2
		cat /tmp/hawkeye-e2e-mcp.out >&2 || true
		exit 1
	}
	echo "$sse" | grep -q 'text/event-stream' || {
		echo "e2e: MCP POST Accept event-stream must be SSE" >&2
		echo "$sse" >&2
		exit 1
	}
	echo "$sse" | grep -q 'event: message' || {
		echo "e2e: MCP SSE missing event: message" >&2
		echo "$sse" >&2
		exit 1
	}
	unauth="$(python3 - <<'PY'
import json, urllib.error, urllib.request
body = json.dumps({"jsonrpc":"2.0","id":2,"method":"initialize"}).encode()
req = urllib.request.Request(
    "http://127.0.0.1:8741/mcp",
    data=body,
    headers={"Accept": "text/event-stream", "Content-Type": "application/json"},
    method="POST",
)
try:
    urllib.request.urlopen(req, timeout=5)
    print("UNEXPECTED_OK")
except urllib.error.HTTPError as e:
    print("CODE=%d" % e.code)
    print("CT=" + e.headers.get("Content-Type", ""))
PY
)"
	echo "$unauth" | grep -q 'CODE=401' || {
		echo "e2e: MCP POST without token must be 401" >&2
		echo "$unauth" >&2
		exit 1
	}
	echo "$unauth" | grep -q 'text/event-stream' && {
		echo "e2e: 401 must not be SSE" >&2
		exit 1
	}
	kill "$mcp_pid" 2>/dev/null || true
	wait "$mcp_pid" 2>/dev/null || true
	echo "e2e: MCP POST SSE ok"
else
	echo "e2e: python3 missing; MCP SSE not exercised" >&2
	exit 1
fi

echo "e2e: PASS"
