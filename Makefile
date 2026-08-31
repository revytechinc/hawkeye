# Copyright (c) 2026 REVYTECH, Inc.
# SPDX-License-Identifier: BSD-3-Clause

PREFIX ?= /usr/local
DESTDIR ?=
GO ?= go
CGO_ENABLED ?= 0
BIN = hawkeye

.PHONY: all build test cover install clean man

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

man:
	@if command -v mandoc >/dev/null 2>&1; then \
		mandoc -T lint man/hawkeye.8 man/hawkeye.conf.5; \
	else \
		echo "mandoc not installed; equivalent lint skipped at runtime (see docs/TEST-EVIDENCE.md)"; \
	fi

clean:
	rm -f $(BIN) coverage.out coverage.html
