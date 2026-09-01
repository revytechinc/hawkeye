# Copyright (c) 2026 REVYTECH, Inc.
# SPDX-License-Identifier: BSD-3-Clause

PREFIX ?= /usr/local
DESTDIR ?=
GO ?= go
CGO_ENABLED ?= 0
BIN = hawkeye

# Rescue / RO kit prefixes. DESTDIR-friendly: the build chroot need not
# have a live /boot or /rescue. Override RESCUE_DIR / BOOT_HAWKEYE if needed.
RESCUE_DIR ?= /rescue
BOOT_HAWKEYE ?= /boot/hawkeye
# Optional knowledge.sqlite from hawkeye-data. Do not vendor a corpus here.
KNOWLEDGE_SRC ?=

.PHONY: all build test cover install install-rescue clean man e2e-freebsd

all: build

build:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build -buildvcs=false -trimpath -ldflags "-s -w" -o $(BIN) ./cmd/hawkeye

test:
	$(GO) test ./internal/... ./cmd/hawkeye -count=1

cover:
	$(GO) test ./internal/... ./cmd/hawkeye -count=1 -coverprofile=coverage.out
	$(GO) tool cover -func=coverage.out

install: build
	install -d $(DESTDIR)$(PREFIX)/bin
	install -m 0755 $(BIN) $(DESTDIR)$(PREFIX)/bin/$(BIN)
	install -d $(DESTDIR)$(PREFIX)/etc/cloudbsd/hawkeye
	install -m 0644 configs/config.example.json $(DESTDIR)$(PREFIX)/etc/cloudbsd/hawkeye/config.json.sample
	install -d $(DESTDIR)$(PREFIX)/etc/rc.d
	install -m 0755 rc.d/hawkeye $(DESTDIR)$(PREFIX)/etc/rc.d/hawkeye
	install -d $(DESTDIR)$(PREFIX)/share/man/man8
	install -d $(DESTDIR)$(PREFIX)/share/man/man5
	install -m 0644 man/hawkeye.8 $(DESTDIR)$(PREFIX)/share/man/man8/hawkeye.8
	install -m 0644 man/hawkeye.conf.5 $(DESTDIR)$(PREFIX)/share/man/man5/hawkeye.conf.5

# DESTDIR/STAGEDIR always stages prefixes (package/chroot). Live /rescue:
# a real directory or a symlink to a writable rescue image is installed
# INTO (do not replace or empty FreeBSD tools). A dangling bastille
# symlink is unlinked and replaced with a real directory so
# /rescue/hawkeye is a runnable binary. EROFS/EACCES/EPERM skip with the
# same read-only message as /boot. Missing /rescue is skip. Do not
# remount. bmake recipes run with set -e, so live status is captured
# with ||; an unguarded $(mkdir) exits 1 before skip. DESTDIR still
# creates both prefixes and still fails on errors. Knowledge artifacts
# come from hawkeye-data (dual prefix); set KNOWLEDGE_SRC to copy a
# sqlite file.
install-rescue: build
	@if [ -n "$(DESTDIR)" ]; then \
		install -d $(DESTDIR)$(RESCUE_DIR); \
		install -m 0755 $(BIN) $(DESTDIR)$(RESCUE_DIR)/$(BIN); \
	elif [ -d "$(RESCUE_DIR)" ]; then \
		_rescue_rc=0; \
		_rescue_err=$$(install -m 0755 $(BIN) "$(RESCUE_DIR)/$(BIN)" 2>&1) || _rescue_rc=$$?; \
		if [ $$_rescue_rc -ne 0 ]; then \
			case "$$_rescue_err" in \
			*Read-only*|*read-only*|*Permission\ denied*|*Operation\ not\ permitted*|*EROFS*|*EACCES*|*EPERM*) \
				echo "install-rescue: skip $(RESCUE_DIR) (read-only)"; \
				;; \
			*) \
				echo "$$_rescue_err" >&2; \
				exit $$_rescue_rc; \
				;; \
			esac; \
		fi; \
	elif [ -L "$(RESCUE_DIR)" ]; then \
		_rescue_rc=0; \
		_rescue_err=$$({ rm -f "$(RESCUE_DIR)" && mkdir -m 0755 "$(RESCUE_DIR)" && install -m 0755 $(BIN) "$(RESCUE_DIR)/$(BIN)"; } 2>&1) || _rescue_rc=$$?; \
		if [ $$_rescue_rc -ne 0 ]; then \
			case "$$_rescue_err" in \
			*Read-only*|*read-only*|*Permission\ denied*|*Operation\ not\ permitted*|*EROFS*|*EACCES*|*EPERM*) \
				echo "install-rescue: skip $(RESCUE_DIR) (read-only)"; \
				;; \
			*) \
				echo "$$_rescue_err" >&2; \
				exit $$_rescue_rc; \
				;; \
			esac; \
		fi; \
	else \
		echo "install-rescue: skip $(RESCUE_DIR) (not a real directory)"; \
	fi
	@if [ -n "$(DESTDIR)" ]; then \
		install -d $(DESTDIR)$(BOOT_HAWKEYE); \
		if [ -n "$(KNOWLEDGE_SRC)" ] && [ -f "$(KNOWLEDGE_SRC)" ]; then \
			install -m 0644 "$(KNOWLEDGE_SRC)" $(DESTDIR)$(BOOT_HAWKEYE)/knowledge.sqlite; \
		fi; \
	elif [ -d "$(BOOT_HAWKEYE)" ]; then \
		if [ -n "$(KNOWLEDGE_SRC)" ] && [ -f "$(KNOWLEDGE_SRC)" ]; then \
			install -m 0644 "$(KNOWLEDGE_SRC)" $(BOOT_HAWKEYE)/knowledge.sqlite; \
		fi; \
	elif [ -d "`dirname $(BOOT_HAWKEYE)`" ]; then \
		_boot_rc=0; \
		_boot_err=$$(mkdir -m 0755 $(BOOT_HAWKEYE) 2>&1) || _boot_rc=$$?; \
		if [ $$_boot_rc -eq 0 ]; then \
			if [ -n "$(KNOWLEDGE_SRC)" ] && [ -f "$(KNOWLEDGE_SRC)" ]; then \
				install -m 0644 "$(KNOWLEDGE_SRC)" $(BOOT_HAWKEYE)/knowledge.sqlite; \
			fi; \
		else \
			case "$$_boot_err" in \
			*Read-only*|*read-only*|*Permission\ denied*|*Operation\ not\ permitted*|*EROFS*|*EACCES*|*EPERM*) \
				echo "install-rescue: skip $(BOOT_HAWKEYE) (read-only)"; \
				;; \
			*) \
				echo "$$_boot_err" >&2; \
				exit $$_boot_rc; \
				;; \
			esac; \
		fi; \
	else \
		echo "install-rescue: skip $(BOOT_HAWKEYE) (no /boot)"; \
	fi

man:
	@if command -v mandoc >/dev/null 2>&1; then \
		mandoc -T lint man/hawkeye.8 man/hawkeye.conf.5; \
	else \
		echo "mandoc not installed; equivalent lint skipped at runtime (see docs/TEST-EVIDENCE.md)"; \
	fi

# Product e2e on FreeBSD 14.5 / 15.1 / 16. Exits 2 on non-FreeBSD.
# Dry-run only (no --yes). Override HAWKEYE= and QUERY=.
e2e-freebsd:
	@sh scripts/e2e-freebsd16.sh

clean:
	rm -f $(BIN) coverage.out coverage.html
