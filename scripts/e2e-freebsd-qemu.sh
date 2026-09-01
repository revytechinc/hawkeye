#!/bin/sh
# Host-side FreeBSD e2e driver (Linux/QEMU fallback only).
#
# CloudBSD / FreeBSD hosts should use vm-bhyve instead:
#   sh scripts/e2e-freebsd-vm-bhyve.sh
#
# Cross-builds hawkeye for FreeBSD/amd64, boots a FreeBSD cloud image under
# QEMU (TCG or KVM), runs scripts/e2e-freebsd.sh in the guest, copies the
# log out via a VV FAT share.
#
# Requires: qemu-system-x86_64, OVMF, genisoimage, a FreeBSD *.qcow2
# cloud image, and enough RAM (~3G guest).
#
# Usage:
#   FREEBSD_QCOW=/path/to/FreeBSD-14.3-RELEASE-amd64.qcow2 \
#     sh scripts/e2e-freebsd-qemu.sh
#
# Env:
#   FREEBSD_QCOW   path to writable qcow2 (copied from a pristine image)
#   E2E_WORKDIR    scratch dir (default /tmp/hawkeye-e2e)
#   E2E_ACCEL      tcg|kvm (default: kvm if /dev/kvm writable, else tcg)
#   E2E_MEM        guest RAM MiB (default 3072)
set -eu

ROOT="$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)"
WORKDIR="${E2E_WORKDIR:-/tmp/hawkeye-e2e}"
QCOW="${FREEBSD_QCOW:-$WORKDIR/disk.qcow2}"
MEM="${E2E_MEM:-3072}"
OVMF_CODE="${OVMF_CODE:-/usr/share/OVMF/OVMF_CODE_4M.fd}"
OVMF_VARS_SRC="${OVMF_VARS_SRC:-/usr/share/OVMF/OVMF_VARS_4M.fd}"
# Debian/Ubuntu OVMF package ships *_4M.fd; override if needed.

if [ ! -r "$QCOW" ]; then
	printf 'missing FREEBSD_QCOW=%s\n' "$QCOW" >&2
	exit 1
fi
if [ ! -r "$OVMF_CODE" ] || [ ! -r "$OVMF_VARS_SRC" ]; then
	printf 'missing OVMF firmware (%s / %s)\n' "$OVMF_CODE" "$OVMF_VARS_SRC" >&2
	exit 1
fi

mkdir -p "$WORKDIR/share" "$WORKDIR/out"
rm -f "$WORKDIR/out"/*

printf 'cross-building freebsd/amd64 hawkeye...\n'
CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 go build -o "$WORKDIR/share/hawkeye" "$ROOT/cmd/hawkeye"
cp "$ROOT/scripts/e2e-freebsd.sh" "$WORKDIR/share/e2e-freebsd.sh"
genisoimage -quiet -o "$WORKDIR/share.iso" -R -J -V HAWKEYE "$WORKDIR/share"
cp "$OVMF_VARS_SRC" "$WORKDIR/OVMF_VARS.fd"

ACCEL="${E2E_ACCEL:-}"
if [ -z "$ACCEL" ]; then
	if [ -w /dev/kvm ] && [ -r /dev/kvm ]; then
		ACCEL=kvm
	else
		ACCEL=tcg
	fi
fi
case "$ACCEL" in
kvm) QEMU_ACCEL="-accel kvm -cpu host" ;;
tcg) QEMU_ACCEL="-accel tcg,thread=multi -cpu max" ;;
*)
	printf 'unknown E2E_ACCEL=%s\n' "$ACCEL" >&2
	exit 1
	;;
esac

printf 'booting FreeBSD under QEMU accel=%s (serial console; login as root)...\n' "$ACCEL"
printf 'After login:\n'
printf '  mkdir -p /mnt/share /mnt/out\n'
printf '  mount -t cd9660 /dev/cd0 /mnt/share\n'
printf '  mount -t msdosfs /dev/vtbd0s1 /mnt/out\n'
printf '  cp /mnt/share/* /tmp/ && chmod +x /tmp/hawkeye\n'
printf '  E2E_LOG=/mnt/out/hawkeye-e2e.log sh /tmp/e2e-freebsd.sh /tmp/hawkeye\n'
printf '  sync; poweroff\n'
printf 'Host then reads %s/out/hawkeye-e2e.log\n' "$WORKDIR"

# shellcheck disable=SC2086
exec qemu-system-x86_64 \
	$QEMU_ACCEL \
	-m "$MEM" \
	-smp 4 \
	-drive if=pflash,format=raw,readonly=on,file="$OVMF_CODE" \
	-drive if=pflash,format=raw,file="$WORKDIR/OVMF_VARS.fd" \
	-drive file="$QCOW",format=qcow2 \
	-cdrom "$WORKDIR/share.iso" \
	-drive file="fat:rw:$WORKDIR/out",format=raw,if=virtio \
	-netdev user,id=net0 \
	-device virtio-net-pci,netdev=net0 \
	-nographic
