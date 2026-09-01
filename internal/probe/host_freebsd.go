// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

//go:build freebsd

package probe

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"unsafe"

	"golang.org/x/sys/unix"
)

func cstring(b []byte) string {
	if i := bytes.IndexByte(b, 0); i >= 0 {
		b = b[:i]
	}
	return string(b)
}

func liveSysctlInt(name string) (int, bool) {
	v, err := unix.SysctlUint32(name)
	if err == nil {
		return int(v), true
	}
	// Jail/rescue: libc sysctl can fail; sysctl(8) in /sbin or /rescue is the overlay.
	return liveSysctl8Int(name)
}

func liveMountTable() (string, error) {
	n, err := unix.Getfsstat(nil, unix.MNT_NOWAIT)
	if err != nil || n <= 0 {
		return "", err
	}
	buf := make([]unix.Statfs_t, n)
	n, err = unix.Getfsstat(buf, unix.MNT_NOWAIT)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for i := 0; i < n; i++ {
		st := buf[i]
		on := cstring(st.Mntonname[:])
		from := cstring(st.Mntfromname[:])
		fstype := cstring(st.Fstypename[:])
		opts := "rw"
		if st.Flags&unix.MNT_RDONLY != 0 {
			opts = "ro"
		}
		fmt.Fprintf(&b, "%s %s %s %s 0 0\n", from, on, fstype, opts)
	}
	return b.String(), nil
}

func liveStatfsReadOnly(path string) bool {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return false
	}
	return st.Flags&unix.MNT_RDONLY != 0
}

// liveNetworkCarrier uses getifaddrs (net.Interfaces) plus SIOCGIFMEDIA.
// ifconfig -a is the fallback when ioctl is unavailable. Not Linux sysfs.
func liveNetworkCarrier() bool {
	ifaces, err := net.Interfaces()
	if err == nil {
		st := statusesFromGoIfaces(ifaces)
		refineIfmedia(st)
		if CarrierUp(st) {
			return true
		}
	}
	return ifconfigLiveCarrier()
}

func ifconfigLiveCarrier() bool {
	out, err := exec.Command("/sbin/ifconfig", "-a").Output()
	if err != nil {
		return false
	}
	return CarrierUp(ParseIfconfig(string(out)))
}

func refineIfmedia(ifaces []IfaceStatus) {
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, 0)
	if err != nil {
		return
	}
	defer unix.Close(fd)
	for i := range ifaces {
		if ifaces[i].Loopback || IsLoopbackName(ifaces[i].Name) {
			continue
		}
		status, ok := ioctlIfmediaStatus(fd, ifaces[i].Name)
		if !ok {
			continue
		}
		active, valid := IfmediaActive(status)
		if !valid {
			continue
		}
		ifaces[i].CarrierKnown = true
		ifaces[i].Carrier = active
	}
}

// ioctlIfmediaStatus issues SIOCGIFMEDIA. ifm_status sits at offset 24
// (IFNAMSIZ + two ints) on every FreeBSD word size.
func ioctlIfmediaStatus(fd int, name string) (int32, bool) {
	if name == "" || len(name) >= 16 {
		return 0, false
	}
	size := int((uint(unix.SIOCGIFMEDIA) >> 16) & 0x1fff)
	if size < 28 {
		size = 48
	}
	buf := make([]byte, size)
	copy(buf[:16], name)
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), uintptr(unix.SIOCGIFMEDIA), uintptr(unsafe.Pointer(&buf[0])))
	if errno != 0 {
		return 0, false
	}
	status := int32(nativeEndian.Uint32(buf[24:28]))
	return status, true
}

var nativeEndian = func() binary.ByteOrder {
	var x uint16 = 1
	if *(*byte)(unsafe.Pointer(&x)) == 1 {
		return binary.LittleEndian
	}
	return binary.BigEndian
}()
