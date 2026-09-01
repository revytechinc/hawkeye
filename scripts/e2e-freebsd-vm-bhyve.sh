#!/bin/sh
# Host-side Hawkeye FreeBSD e2e via vm-bhyve (CloudBSD preferred).
#
# Run ON a FreeBSD host as root (not inside Linux/QEMU). Requires:
#   pkg install vm-bhyve bhyve-firmware uefi-edk2-bhyve go
#   sysrc vm_enable=YES vm_dir="zfs:zroot/vm" (or your pool)
#   vm init && vm switch create public && vm switch add public <iface>
#
# One-time guest (example — adjust pool/switch to your host):
#   vm img https://download.freebsd.org/ftp/snapshots/VM-IMAGES/16.0-CURRENT/amd64/Latest/FreeBSD-16.0-CURRENT-amd64-BASIC-CLOUDINIT-ufs.qcow2.xz
#   E2E_SSH_PUB=/root/.ssh/id_ed25519.pub sh scripts/e2e-freebsd-vm-bhyve.sh --provision
#
# Routine run (guest already exists, SSH key authorized):
#   sh scripts/e2e-freebsd-vm-bhyve.sh
#
# Env:
#   E2E_VM_NAME     guest name (default hawkeye-e2e)
#   E2E_VM_SWITCH   vm switch (default public)
#   E2E_WORKDIR     scratch (default /tmp/hawkeye-e2e)
#   E2E_SSH_KEY     private key for guest SSH (default $E2E_WORKDIR/id_ed25519)
#   E2E_SSH_PUB     public key for cloud-init provision (default ${E2E_SSH_KEY}.pub)
#   E2E_SSH_USER    guest login (default root)
#   E2E_GUEST_IP    skip discovery when set
#   E2E_EVIDENCE    host dir for log (default docs/evidence/freebsd-e2e-latest)
#   E2E_DESTROY     1 to vm destroy guest after run (provision mode default)
#   E2E_SKIP_BUILD  1 to reuse $E2E_WORKDIR/share/hawkeye
set -eu

ROOT="$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)"
WORKDIR="${E2E_WORKDIR:-/tmp/hawkeye-e2e}"
VM="${E2E_VM_NAME:-hawkeye-e2e}"
SWITCH="${E2E_VM_SWITCH:-public}"
SSH_USER="${E2E_SSH_USER:-root}"
SSH_KEY="${E2E_SSH_KEY:-$WORKDIR/id_ed25519}"
SSH_PUB="${E2E_SSH_PUB:-${SSH_KEY}.pub}"
EVIDENCE="${E2E_EVIDENCE:-$ROOT/docs/evidence/freebsd-e2e-latest}"
PROVISION=0
DESTROY="${E2E_DESTROY:-0}"

log() { printf '%s\n' "$*"; }
die() { printf 'e2e-vm-bhyve: %s\n' "$*" >&2; exit 1; }

case "$(uname -s)" in
FreeBSD) ;;
*) die "must run on FreeBSD (got $(uname -s)); use scripts/e2e-freebsd-qemu.sh on Linux" ;;
esac

while [ $# -gt 0 ]; do
	case "$1" in
	--provision) PROVISION=1; DESTROY="${E2E_DESTROY:-1}" ;;
	-h|--help)
		sed -n '1,30p' "$0"
		exit 0
		;;
	*) die "unknown arg: $1 (try --help)" ;;
	esac
	shift
done

if ! command -v vm >/dev/null 2>&1; then
	die "vm-bhyve not installed (pkg install vm-bhyve bhyve-firmware uefi-edk2-bhyve)"
fi

if ! kldstat -m vmm >/dev/null 2>&1; then
	kldload vmm 2>/dev/null || die "vmm kernel module not loaded (kldload vmm as root)"
fi
if ! sysctl -n hw.vmm.vm.max >/dev/null 2>&1; then
	die "hw.vmm not available after loading vmm"
fi

mkdir -p "$WORKDIR/share" "$EVIDENCE"

if [ ! -f "$SSH_KEY" ]; then
	ssh-keygen -t ed25519 -N '' -f "$SSH_KEY" -C hawkeye-e2e
fi

if [ "${E2E_SKIP_BUILD:-0}" != 1 ]; then
	log "building hawkeye (native freebsd/amd64)..."
	CGO_ENABLED=0 go build -o "$WORKDIR/share/hawkeye" "$ROOT/cmd/hawkeye"
else
	[ -x "$WORKDIR/share/hawkeye" ] || die "E2E_SKIP_BUILD=1 but $WORKDIR/share/hawkeye missing"
fi
cp "$ROOT/scripts/e2e-freebsd.sh" "$WORKDIR/share/e2e-freebsd.sh"
genisoimage -quiet -o "$WORKDIR/share.iso" -R -J -V HAWKEYE "$WORKDIR/share"

vm_conf_dir() {
	vm info "$VM" 2>/dev/null | awk -F: '/^path:/ {gsub(/^[ \t]+/, "", $2); print $2; exit}'
}

attach_share_iso() {
	conf="$(vm_conf_dir)" || die "vm $VM not found (run with --provision once)"
	[ -n "$conf" ] || die "cannot resolve vm config dir for $VM"
	cfg="$conf/${VM}.conf"
	[ -f "$cfg" ] || die "missing $cfg"
	# Idempotent: replace prior HAWKEYE cdrom line block.
	grep -v 'HAWKEYE share.iso' "$cfg" >"$cfg.tmp" || true
	mv "$cfg.tmp" "$cfg"
	{
		echo "# HAWKEYE share.iso (e2e attach)"
		echo 'disk1_type="ahci-cd"'
		echo 'disk1_dev="custom"'
		echo "disk1_name=\"$WORKDIR/share.iso\""
	} >>"$cfg"
}

provision_guest() {
	[ -r "$SSH_PUB" ] || die "missing E2E_SSH_PUB=$SSH_PUB for --provision"
	vm info "$VM" >/dev/null 2>&1 && vm destroy -f "$VM"
	log "provisioning vm $VM (cloud-init + SSH key)..."
	# Requires a cloud-init-capable template and image in vm datastore (vm img ...).
	vm create -t cloud-init -C -k "$SSH_PUB" -s 8g -m 4096M -c 2 -w "$SWITCH" "$VM" \
		|| vm create -t freebsd-cloud -C -k "$SSH_PUB" -s 8g -m 4096M -c 2 -w "$SWITCH" "$VM" \
		|| die "vm create failed; ensure cloud image is in vm datastore (vm img ...)"
	attach_share_iso
}

if [ "$PROVISION" -eq 1 ]; then
	provision_guest
else
	vm info "$VM" >/dev/null 2>&1 || die "vm $VM missing; run: E2E_SSH_PUB=$SSH_PUB $0 --provision"
	attach_share_iso
fi

log "starting vm $VM..."
vm start "$VM" || vm restart "$VM"

guest_ip() {
	if [ -n "${E2E_GUEST_IP:-}" ]; then
		printf '%s\n' "$E2E_GUEST_IP"
		return 0
	fi
	# vm-bhyve prints lease/IP on some builds; fall back to parsing vm info.
	vm info "$VM" 2>/dev/null | awk -F: '/ip-address|ipv4/ {gsub(/^[ \t]+/, "", $2); print $2; exit}'
}

wait_ssh() {
	ip=""
	i=0
	while [ "$i" -lt 90 ]; do
		ip="$(guest_ip || true)"
		if [ -n "$ip" ] && ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
			-o ConnectTimeout=5 -o BatchMode=yes "$SSH_USER@$ip" 'uname -s' >/dev/null 2>&1; then
			printf '%s\n' "$ip"
			return 0
		fi
		i=$((i + 1))
		sleep 5
	done
	return 1
}

IP="$(wait_ssh)" || die "guest SSH not ready (set E2E_GUEST_IP if discovery fails)"
log "guest ssh ready at $IP"

SSH="ssh -i $SSH_KEY -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null $SSH_USER@$IP"
SCP="scp -i $SSH_KEY -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null"

log "copying hawkeye + e2e script into guest..."
$SCP "$WORKDIR/share/hawkeye" "$WORKDIR/share/e2e-freebsd.sh" "$SSH_USER@$IP:/tmp/"

log "running e2e inside guest..."
$SSH "$SSH_USER@$IP" 'chmod +x /tmp/hawkeye /tmp/e2e-freebsd.sh; E2E_LOG=/tmp/hawkeye-e2e.log sh /tmp/e2e-freebsd.sh /tmp/hawkeye; echo E2E_EXIT:$?'

$SCP "$SSH_USER@$IP:/tmp/hawkeye-e2e.log" "$EVIDENCE/hawkeye-e2e.log"
{
	printf 'guest=%s\n' "$($SSH "$SSH_USER@$IP" 'uname -srm; freebsd-version 2>/dev/null || true')"
	printf 'vm=%s switch=%s ip=%s\n' "$VM" "$SWITCH" "$IP"
	printf 'host=%s\n' "$(uname -srm)"
	printf 'binary=native go build @ %s\n' "$(cd "$ROOT" && git rev-parse --short HEAD 2>/dev/null || echo unknown)"
	printf 'runner=scripts/e2e-freebsd-vm-bhyve.sh\n'
} >"$EVIDENCE/host-builder.txt"

grep -E '^(PASS|FAIL|NOTE|===)' "$EVIDENCE/hawkeye-e2e.log" || true
if grep -q '=== summary fail=0' "$EVIDENCE/hawkeye-e2e.log"; then
	log "e2e PASS — evidence in $EVIDENCE"
	rc=0
else
	log "e2e FAIL — see $EVIDENCE/hawkeye-e2e.log"
	rc=1
fi

if [ "$DESTROY" = 1 ]; then
	log "destroying vm $VM..."
	vm stop "$VM" 2>/dev/null || true
	vm destroy -f "$VM"
fi

exit "$rc"
