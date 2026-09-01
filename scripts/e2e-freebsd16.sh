#!/bin/sh
# Copyright (c) 2026 REVYTECH, Inc.
# SPDX-License-Identifier: BSD-3-Clause
#
# Product e2e on FreeBSD 14.5 / 15.1 / 16 jails. No --yes (dry-run only).
# Exit 2 if this is not FreeBSD so Linux CI can skip.

set -eu

HAWKEYE="${HAWKEYE:-hawkeye}"
QUERY="${QUERY:-ZFS root is read-only after boot}"

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
echo "e2e: PASS"
