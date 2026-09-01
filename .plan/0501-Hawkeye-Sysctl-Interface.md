# Hawkeye — Sysctl Interface

**Document ID:** HAWKEYE-0501
**Version:** 0.1.0
**Last Updated:** 2026-08-30
**Status:** ACTIVE

Hawkeye reads `kern.securelevel` via the host overlay: `unix.SysctlUint32`
on FreeBSD, then `sysctl(8) -n` (`/sbin/sysctl`, `/usr/sbin/sysctl`,
`/rescue/sysctl`). Doctor reports the value as a note; unknown is not a
failure. It does not register its own MIB.
