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

.PHONY: all build test cover install install-rescue clean man

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

# DESTDIR/STAGEDIR always stages prefixes (package/chroot). Live install
# writes /rescue only when it is a real directory — not a dangling bastille
# symlink. /boot/hawkeye is created when /boot exists and is writable.
# Missing /boot is skip. A present /boot that is read-only (bastille
# symlink to a release boot, EROFS/EACCES/EPERM) is skip — same style as
# skip /rescue. Do not remount. Knowledge artifacts come from hawkeye-data
# (dual prefix); set KNOWLEDGE_SRC to copy a sqlite file.
install-rescue: build
	@if [ -n "$(DESTDIR)" ] || { [ -d "$(RESCUE_DIR)" ] && [ ! -L "$(RESCUE_DIR)" ]; }; then \
		install -d $(DESTDIR)$(RESCUE_DIR); \
		install -m 0755 $(BIN) $(DESTDIR)$(RESCUE_DIR)/$(BIN); \
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
		_boot_err=$$(mkdir -m 0755 $(BOOT_HAWKEYE) 2>&1); \
		_boot_rc=$$?; \
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

clean:
	rm -f $(BIN) coverage.out coverage.html
