// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package knowledge_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/revytechinc/hawkeye/internal/knowledge"
)

func TestCanStageRescue_DanglingSymlinkAllowed(t *testing.T) {
	dir := t.TempDir()
	rescue := filepath.Join(dir, "rescue")
	if err := os.Symlink(filepath.Join(dir, "missing-target"), rescue); err != nil {
		t.Fatal(err)
	}
	if !knowledge.CanStageRescue("", rescue, os.Lstat) {
		t.Fatal("dangling bastille /rescue must be replaceable with a real /rescue")
	}
	if !knowledge.CanStageRescue("", rescue, nil) {
		t.Fatal("nil lstat uses os.Lstat")
	}
	if !knowledge.CanStageRescue("/var/tmp/stage", rescue, os.Lstat) {
		t.Fatal("DESTDIR/STAGEDIR must still stage /rescue")
	}
	if !strings.Contains(knowledge.RescueSkipReadOnly(rescue), "skip "+rescue+" (read-only)") {
		t.Fatal(knowledge.RescueSkipReadOnly(rescue))
	}
}

func TestCanStageRescue_SymlinkToRealDirAllowed(t *testing.T) {
	dir := t.TempDir()
	realRescue := filepath.Join(dir, "real-rescue")
	if err := os.Mkdir(realRescue, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(realRescue, "ls"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	rescue := filepath.Join(dir, "rescue")
	if err := os.Symlink(realRescue, rescue); err != nil {
		t.Fatal(err)
	}
	if !knowledge.CanStageRescue("", rescue, os.Lstat) {
		t.Fatal("symlink to a real rescue image must be installable (into, not replace)")
	}
}

func TestCanStageRescue_RealDirectoryAllowed(t *testing.T) {
	dir := t.TempDir()
	rescue := filepath.Join(dir, "rescue")
	if err := os.Mkdir(rescue, 0o755); err != nil {
		t.Fatal(err)
	}
	if !knowledge.CanStageRescue("", rescue, os.Lstat) {
		t.Fatal("real /rescue directory must be installable")
	}
}

func TestCanStageRescue_MissingSkipped(t *testing.T) {
	dir := t.TempDir()
	if knowledge.CanStageRescue("", filepath.Join(dir, "rescue"), os.Lstat) {
		t.Fatal("missing /rescue must be skipped on a live jail")
	}
	if knowledge.CanStageRescue("", "", os.Lstat) {
		t.Fatal("empty rescue dir")
	}
}

func TestCanStageBootKit_BootExistsNoHawkeye(t *testing.T) {
	dir := t.TempDir()
	boot := filepath.Join(dir, "boot")
	if err := os.Mkdir(boot, 0o755); err != nil {
		t.Fatal(err)
	}
	kit := filepath.Join(boot, "hawkeye")
	if _, err := os.Stat(kit); !os.IsNotExist(err) {
		t.Fatal("fixture must not already have /boot/hawkeye")
	}
	if !knowledge.CanStageBootKit("", kit, os.Lstat) {
		t.Fatal("/boot exists: may create /boot/hawkeye")
	}
	if !knowledge.CanStageBootKit("/var/tmp/stage", kit, os.Lstat) {
		t.Fatal("DESTDIR must stage /boot/hawkeye even without live /boot")
	}
}

func TestCanStageBootKit_ExistingKitAllowed(t *testing.T) {
	dir := t.TempDir()
	kit := filepath.Join(dir, "boot", "hawkeye")
	if err := os.MkdirAll(kit, 0o755); err != nil {
		t.Fatal(err)
	}
	if !knowledge.CanStageBootKit("", kit, os.Lstat) {
		t.Fatal("existing /boot/hawkeye must be allowed")
	}
	if !knowledge.CanStageBootKit("", kit, nil) {
		t.Fatal("nil lstat uses os.Lstat")
	}
}

func TestCanStageBootKit_MissingBootSkipped(t *testing.T) {
	dir := t.TempDir()
	kit := filepath.Join(dir, "no-boot", "hawkeye")
	if knowledge.CanStageBootKit("", kit, os.Lstat) {
		t.Fatal("missing /boot must not invent a boot filesystem")
	}
	if knowledge.CanStageBootKit("", "", os.Lstat) {
		t.Fatal("empty boot kit")
	}
}

func TestIsReadOnlyCreateError(t *testing.T) {
	for _, errno := range []syscall.Errno{syscall.EROFS, syscall.EACCES, syscall.EPERM} {
		err := &os.PathError{Op: "mkdir", Path: "/boot/hawkeye", Err: errno}
		if !knowledge.IsReadOnlyCreateError(err) {
			t.Fatalf("%v must be a skippable RO create error", errno)
		}
		if !knowledge.IsReadOnlyCreateError(errno) {
			t.Fatalf("bare %v must be skippable", errno)
		}
	}
	if knowledge.IsReadOnlyCreateError(nil) {
		t.Fatal("nil is not RO")
	}
	if knowledge.IsReadOnlyCreateError(&os.PathError{Op: "mkdir", Path: "/boot/hawkeye", Err: syscall.ENOSPC}) {
		t.Fatal("ENOSPC must fail the target, not skip")
	}
}

func TestStageBootKit_FakeRODestSkipped(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("chmod 0555 is still writable as root")
	}
	dir := t.TempDir()
	boot := filepath.Join(dir, "boot")
	if err := os.Mkdir(boot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(boot, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(boot, 0o755) })
	kit := filepath.Join(boot, "hawkeye")
	skipped, err := knowledge.StageBootKit("", kit, os.MkdirAll)
	if err != nil {
		t.Fatalf("RO /boot must skip with exit 0, not fail: %v", err)
	}
	if !skipped {
		t.Fatal("fake RO dest must skip /boot/hawkeye")
	}
	if _, err := os.Stat(kit); !os.IsNotExist(err) {
		t.Fatal("must not create /boot/hawkeye on RO dest")
	}
	if !strings.Contains(knowledge.BootKitSkipMessage(kit), "skip "+kit) {
		t.Fatal(knowledge.BootKitSkipMessage(kit))
	}
	if !strings.Contains(knowledge.BootKitSkipMessage(kit), "read-only") {
		t.Fatal("skip message must match skip /rescue style")
	}
}

func TestStageBootKit_ReadOnlyErrnosSkipped(t *testing.T) {
	for _, errno := range []syscall.Errno{syscall.EROFS, syscall.EACCES, syscall.EPERM} {
		mkdir := func(string, os.FileMode) error {
			return &os.PathError{Op: "mkdir", Path: "/boot/hawkeye", Err: errno}
		}
		skipped, err := knowledge.StageBootKit("", "/boot/hawkeye", mkdir)
		if err != nil {
			t.Fatalf("%v must skip, not error: %v", errno, err)
		}
		if !skipped {
			t.Fatalf("%v must skip live /boot/hawkeye", errno)
		}
	}
}

func TestStageBootKit_DESTDIRCreates(t *testing.T) {
	dir := t.TempDir()
	kit := filepath.Join(dir, "boot", "hawkeye")
	skipped, err := knowledge.StageBootKit("/var/tmp/stage", kit, os.MkdirAll)
	if err != nil {
		t.Fatal(err)
	}
	if skipped {
		t.Fatal("DESTDIR must create /boot/hawkeye, not skip")
	}
	fi, err := os.Stat(kit)
	if err != nil || !fi.IsDir() {
		t.Fatalf("DESTDIR must create the kit: %v", err)
	}
}

func TestStageBootKit_DESTDIRReadOnlyIsError(t *testing.T) {
	mkdir := func(string, os.FileMode) error {
		return &os.PathError{Op: "mkdir", Path: "/stage/boot/hawkeye", Err: syscall.EROFS}
	}
	skipped, err := knowledge.StageBootKit("/var/tmp/stage", "/stage/boot/hawkeye", mkdir)
	if err == nil {
		t.Fatal("DESTDIR must not swallow RO errors; staging must create both prefixes")
	}
	if skipped {
		t.Fatal("DESTDIR RO must not be reported as a live skip")
	}
}

func TestStageBootKit_OtherErrorFails(t *testing.T) {
	mkdir := func(string, os.FileMode) error {
		return &os.PathError{Op: "mkdir", Path: "/boot/hawkeye", Err: syscall.ENOSPC}
	}
	skipped, err := knowledge.StageBootKit("", "/boot/hawkeye", mkdir)
	if err == nil {
		t.Fatal("ENOSPC must fail")
	}
	if skipped {
		t.Fatal("ENOSPC is not a RO skip")
	}
}

func TestStageBootKit_EmptyLiveSkipped(t *testing.T) {
	skipped, err := knowledge.StageBootKit("", "", os.MkdirAll)
	if err != nil || !skipped {
		t.Fatalf("empty live path is skip: skipped=%v err=%v", skipped, err)
	}
}

func TestStageBootKit_NilMkdirCreates(t *testing.T) {
	dir := t.TempDir()
	kit := filepath.Join(dir, "boot", "hawkeye")
	skipped, err := knowledge.StageBootKit("", kit, nil)
	if err != nil || skipped {
		t.Fatalf("writable dest with default mkdir: skipped=%v err=%v", skipped, err)
	}
	if fi, err := os.Stat(kit); err != nil || !fi.IsDir() {
		t.Fatalf("must create kit: %v", err)
	}
}

func TestStageBootKit_DESTDIRFakeRODestIsError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("chmod 0555 is still writable as root")
	}
	dir := t.TempDir()
	boot := filepath.Join(dir, "boot")
	if err := os.Mkdir(boot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(boot, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(boot, 0o755) })
	kit := filepath.Join(boot, "hawkeye")
	skipped, err := knowledge.StageBootKit("/var/tmp/stage", kit, os.MkdirAll)
	if err == nil {
		t.Fatal("DESTDIR must fail when the stage dest is RO, not skip")
	}
	if skipped {
		t.Fatal("DESTDIR RO is not a live skip")
	}
}

func readRootMakefile(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

func fakeROBootKit(t *testing.T) (kit string, cleanup func()) {
	t.Helper()
	dir := t.TempDir()
	boot := filepath.Join(dir, "boot")
	if err := os.Mkdir(boot, 0o755); err != nil {
		t.Fatal(err)
	}
	kit = filepath.Join(boot, "hawkeye")
	if err := os.Chmod(boot, 0o555); err != nil {
		t.Fatal(err)
	}
	return kit, func() { _ = os.Chmod(boot, 0o755) }
}

// liveBootMkdirShell extracts the Makefile live /boot/hawkeye mkdir + skip
// block and turns it into a /bin/sh script. Tests run it under set -e so
// GNU make cannot hide the bmake abort.
func liveBootMkdirShell(mk, bootHawkeye string) (string, error) {
	lines := strings.Split(mk, "\n")
	start := -1
	for i, line := range lines {
		if strings.Contains(line, "_boot_err=") && strings.Contains(line, "mkdir") {
			start = i
			if i > 0 && strings.Contains(lines[i-1], "_boot_rc=0") {
				start = i - 1
			}
			break
		}
	}
	if start < 0 {
		return "", os.ErrNotExist
	}
	var body []string
	depth := 0
	sawIf := false
	for _, line := range lines[start:] {
		trim := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(line), "\\"))
		trim = strings.ReplaceAll(trim, "$$", "$")
		trim = strings.ReplaceAll(trim, "$(BOOT_HAWKEYE)", bootHawkeye)
		trim = strings.ReplaceAll(trim, "$(KNOWLEDGE_SRC)", "")
		body = append(body, trim)
		if strings.HasPrefix(trim, "if ") {
			depth++
			sawIf = true
		}
		if strings.HasPrefix(trim, "fi") {
			depth--
			if sawIf && depth == 0 {
				break
			}
		}
	}
	if !sawIf {
		return "", os.ErrInvalid
	}
	return strings.Join(body, "\n"), nil
}

func ensureDummyHawkeye(t *testing.T) {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	dummy := filepath.Join(root, "hawkeye")
	if _, err := os.Stat(dummy); err != nil {
		if err := os.WriteFile(dummy, []byte("x"), 0o755); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Remove(dummy) })
	}
}

func makeInstallRescue(t *testing.T, extra ...string) (string, error) {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	ver, err := exec.Command("make", "--version").CombinedOutput()
	args := []string{"-C", root, "install-rescue"}
	if err == nil && strings.Contains(string(ver), "GNU Make") {
		args = append([]string{"-o", "build"}, args...)
	}
	args = append(args, extra...)
	cmd := exec.Command("make", args...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestMakefileInstallRescue_FakeRODestExit0(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("chmod 0555 is still writable as root")
	}
	if _, err := exec.LookPath("make"); err != nil {
		t.Skip("make not installed")
	}
	ensureDummyHawkeye(t)
	dir := t.TempDir()
	boot := filepath.Join(dir, "boot")
	if err := os.Mkdir(boot, 0o755); err != nil {
		t.Fatal(err)
	}
	rescue := filepath.Join(dir, "rescue")
	if err := os.Symlink(filepath.Join(dir, "missing"), rescue); err != nil {
		t.Fatal(err)
	}
	kit := filepath.Join(boot, "hawkeye")
	if err := os.Chmod(boot, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(boot, 0o755) })

	out, err := makeInstallRescue(t,
		"BIN=hawkeye",
		"RESCUE_DIR="+rescue,
		"BOOT_HAWKEYE="+kit,
		"DESTDIR=",
	)
	if err != nil {
		t.Fatalf("install-rescue must exit 0 on RO /boot: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(rescue, "hawkeye")); err != nil {
		t.Fatalf("dangling /rescue must become /rescue/hawkeye: %v\n%s", err, out)
	}
	if strings.Contains(out, "skip "+rescue+" (not a real directory)") {
		t.Fatalf("dangling /rescue is no longer a skip: %s", out)
	}
	if !strings.Contains(out, "skip "+kit+" (read-only)") {
		t.Fatalf("must skip RO /boot/hawkeye: %s", out)
	}
	if _, err := os.Stat(kit); !os.IsNotExist(err) {
		t.Fatal("must not create /boot/hawkeye on RO dest")
	}
}

func TestUnguardedMkdirCommandSubDiesUnderSetE(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("chmod 0555 is still writable as root")
	}
	kit, cleanup := fakeROBootKit(t)
	defer cleanup()

	// PR 21 idiom. bmake recipes run with set -e, so this aborts before
	// skip. GNU make on CI hid the product exit 1.
	script := "set -e\n_boot_err=$(mkdir -m 0755 " + shellQuote(kit) + " 2>&1)\n_boot_rc=$?\necho skip-never-reached\n"
	out, err := exec.Command("sh", "-c", script).CombinedOutput()
	if err == nil {
		t.Fatal("unguarded $(mkdir) must die under set -e so tests catch the bmake trap")
	}
	if strings.Contains(string(out), "skip-never-reached") {
		t.Fatalf("set -e must abort before skip logic: %s", out)
	}
}

func TestMakefileInstallRescue_SetEDoesNotAbortOnROMkdir(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("chmod 0555 is still writable as root")
	}
	kit, cleanup := fakeROBootKit(t)
	defer cleanup()

	recipe, err := liveBootMkdirShell(readRootMakefile(t), kit)
	if err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command("sh", "-c", "set -e\n"+recipe).CombinedOutput()
	got := string(out)
	if err != nil {
		t.Fatalf("Makefile mkdir capture must survive set -e on RO dest (bmake): %v\n%s\nrecipe:\n%s", err, got, recipe)
	}
	if !strings.Contains(got, "skip "+kit+" (read-only)") {
		t.Fatalf("RO skip must run after guarded mkdir: %s\nrecipe:\n%s", got, recipe)
	}
	if _, err := os.Stat(kit); !os.IsNotExist(err) {
		t.Fatal("must not create /boot/hawkeye on RO dest")
	}
}

func TestMakefileInstallRescue_ExistingFileFails(t *testing.T) {
	if _, err := exec.LookPath("make"); err != nil {
		t.Skip("make not installed")
	}
	dir := t.TempDir()
	boot := filepath.Join(dir, "boot")
	if err := os.Mkdir(boot, 0o755); err != nil {
		t.Fatal(err)
	}
	kit := filepath.Join(boot, "hawkeye")
	if err := os.WriteFile(kit, []byte("not-a-dir"), 0o644); err != nil {
		t.Fatal(err)
	}
	rescue := filepath.Join(dir, "rescue")
	if err := os.Symlink(filepath.Join(dir, "missing"), rescue); err != nil {
		t.Fatal(err)
	}
	out, err := makeInstallRescue(t,
		"BIN=hawkeye",
		"RESCUE_DIR="+rescue,
		"BOOT_HAWKEYE="+kit,
		"DESTDIR=",
	)
	if err == nil {
		t.Fatalf("EEXIST/not-a-dir must fail the target, not skip as read-only\n%s", out)
	}
	if strings.Contains(out, "skip "+kit+" (read-only)") {
		t.Fatalf("real mkdir error is not a live RO skip: %s", out)
	}
}

func TestMakefileInstallRescue_DESTDIRRODestFails(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("chmod 0555 is still writable as root")
	}
	if _, err := exec.LookPath("make"); err != nil {
		t.Skip("make not installed")
	}
	dir := t.TempDir()
	stage := filepath.Join(dir, "stage")
	if err := os.Mkdir(stage, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(stage, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(stage, 0o755) })
	out, err := makeInstallRescue(t,
		"BIN=hawkeye",
		"DESTDIR="+stage,
	)
	if err == nil {
		t.Fatalf("DESTDIR must fail when the stage dest is RO, not skip\n%s", out)
	}
	if strings.Contains(out, "(read-only)") && strings.Contains(out, "skip ") {
		t.Fatalf("DESTDIR RO is not a live skip: %s", out)
	}
}

func TestMakefileInstallRescue_DanglingSymlinkWritesBinary(t *testing.T) {
	if _, err := exec.LookPath("make"); err != nil {
		t.Skip("make not installed")
	}
	ensureDummyHawkeye(t)
	dir := t.TempDir()
	rescue := filepath.Join(dir, "rescue")
	if err := os.Symlink(filepath.Join(dir, "missing"), rescue); err != nil {
		t.Fatal(err)
	}
	boot := filepath.Join(dir, "boot")
	if err := os.Mkdir(boot, 0o755); err != nil {
		t.Fatal(err)
	}
	kit := filepath.Join(boot, "hawkeye")
	out, err := makeInstallRescue(t,
		"BIN=hawkeye",
		"RESCUE_DIR="+rescue,
		"BOOT_HAWKEYE="+kit,
		"DESTDIR=",
	)
	if err != nil {
		t.Fatalf("dangling /rescue must install: %v\n%s", err, out)
	}
	bin := filepath.Join(rescue, "hawkeye")
	fi, err := os.Stat(bin)
	if err != nil {
		t.Fatalf("need /rescue/hawkeye after replacing dangling symlink: %v\n%s", err, out)
	}
	if fi.Mode()&0o111 == 0 {
		t.Fatalf("/rescue/hawkeye must be executable: %s", fi.Mode())
	}
	if st, err := os.Lstat(rescue); err != nil || st.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("dangling symlink must become a real directory: %v %v", err, st)
	}
}

func TestMakefileInstallRescue_IntactRescueKeepsTools(t *testing.T) {
	if _, err := exec.LookPath("make"); err != nil {
		t.Skip("make not installed")
	}
	ensureDummyHawkeye(t)
	dir := t.TempDir()
	rescue := filepath.Join(dir, "rescue")
	if err := os.Mkdir(rescue, 0o755); err != nil {
		t.Fatal(err)
	}
	ls := filepath.Join(rescue, "ls")
	if err := os.WriteFile(ls, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	boot := filepath.Join(dir, "boot")
	if err := os.Mkdir(boot, 0o755); err != nil {
		t.Fatal(err)
	}
	kit := filepath.Join(boot, "hawkeye")
	out, err := makeInstallRescue(t,
		"BIN=hawkeye",
		"RESCUE_DIR="+rescue,
		"BOOT_HAWKEYE="+kit,
		"DESTDIR=",
	)
	if err != nil {
		t.Fatalf("intact /rescue must install into: %v\n%s", err, out)
	}
	if _, err := os.Stat(ls); err != nil {
		t.Fatalf("must not empty a working /rescue: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(rescue, "hawkeye")); err != nil {
		t.Fatalf("must install hawkeye beside existing tools: %v\n%s", err, out)
	}
	ents, err := os.ReadDir(rescue)
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) < 2 {
		t.Fatalf("working /rescue must keep its tools: %v", ents)
	}
}

func TestMakefileInstallRescue_SymlinkToRealRescueInstallsInto(t *testing.T) {
	if _, err := exec.LookPath("make"); err != nil {
		t.Skip("make not installed")
	}
	ensureDummyHawkeye(t)
	dir := t.TempDir()
	realRescue := filepath.Join(dir, "real-rescue")
	if err := os.Mkdir(realRescue, 0o755); err != nil {
		t.Fatal(err)
	}
	ls := filepath.Join(realRescue, "ls")
	if err := os.WriteFile(ls, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	rescue := filepath.Join(dir, "rescue")
	if err := os.Symlink(realRescue, rescue); err != nil {
		t.Fatal(err)
	}
	boot := filepath.Join(dir, "boot")
	if err := os.Mkdir(boot, 0o755); err != nil {
		t.Fatal(err)
	}
	out, err := makeInstallRescue(t,
		"BIN=hawkeye",
		"RESCUE_DIR="+rescue,
		"BOOT_HAWKEYE="+filepath.Join(boot, "hawkeye"),
		"DESTDIR=",
	)
	if err != nil {
		t.Fatalf("symlink-to-real /rescue must install into: %v\n%s", err, out)
	}
	if st, err := os.Lstat(rescue); err != nil || st.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("must keep the symlink to the real rescue image: %v %v", err, st)
	}
	if _, err := os.Stat(ls); err != nil {
		t.Fatalf("must not replace or empty the real rescue image: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(realRescue, "hawkeye")); err != nil {
		t.Fatalf("must install hawkeye into the real image: %v\n%s", err, out)
	}
}

func TestMakefileInstallRescue_RORescueSkipped(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("chmod 0555 is still writable as root")
	}
	if _, err := exec.LookPath("make"); err != nil {
		t.Skip("make not installed")
	}
	ensureDummyHawkeye(t)
	dir := t.TempDir()
	rescue := filepath.Join(dir, "rescue")
	if err := os.Mkdir(rescue, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(rescue, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(rescue, 0o755) })
	boot := filepath.Join(dir, "boot")
	if err := os.Mkdir(boot, 0o755); err != nil {
		t.Fatal(err)
	}
	out, err := makeInstallRescue(t,
		"BIN=hawkeye",
		"RESCUE_DIR="+rescue,
		"BOOT_HAWKEYE="+filepath.Join(boot, "hawkeye"),
		"DESTDIR=",
	)
	if err != nil {
		t.Fatalf("EROFS/EACCES /rescue must skip exit 0: %v\n%s", err, out)
	}
	if !strings.Contains(out, "skip "+rescue+" (read-only)") {
		t.Fatalf("must skip RO /rescue with the existing skip message: %s", out)
	}
	if _, err := os.Stat(filepath.Join(rescue, "hawkeye")); !os.IsNotExist(err) {
		t.Fatal("must not write hawkeye onto RO /rescue")
	}
}

func TestMakefileInstallRescue_DESTDIRCreatesBoth(t *testing.T) {
	if _, err := exec.LookPath("make"); err != nil {
		t.Skip("make not installed")
	}
	dir := t.TempDir()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	dummy := filepath.Join(root, "hawkeye")
	if _, err := os.Stat(dummy); err != nil {
		if err := os.WriteFile(dummy, []byte("x"), 0o755); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Remove(dummy) })
	}
	stage := filepath.Join(dir, "stage")
	out, err := makeInstallRescue(t,
		"BIN=hawkeye",
		"DESTDIR="+stage,
	)
	if err != nil {
		t.Fatalf("DESTDIR install-rescue must exit 0: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(stage, "rescue", "hawkeye")); err != nil {
		t.Fatalf("DESTDIR must create /rescue/hawkeye: %v\n%s", err, out)
	}
	if fi, err := os.Stat(filepath.Join(stage, "boot", "hawkeye")); err != nil || !fi.IsDir() {
		t.Fatalf("DESTDIR must create /boot/hawkeye: %v\n%s", err, out)
	}
}
