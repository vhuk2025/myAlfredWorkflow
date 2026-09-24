SHELL := /bin/bash

DIST := dist
ALFRED := Alfred 5

WORKFLOWS := $(notdir $(wildcard cmd/*))
ifneq ($(WORKFLOW),)
WORKFLOWS := $(WORKFLOW)
endif

.DEFAULT_GOAL := package
.PHONY: help test build package import clean

help:
	@echo "make test                 运行测试"
	@echo "make build                编译 workflow 二进制"
	@echo "make package              打包 dist/NAME.alfredworkflow"
	@echo "make import               打包并导入全部 workflow 到 Alfred"
	@echo "make import WORKFLOW=tt   只导入一个 workflow"
	@echo "make clean                删除 dist/"

test:
	go test ./...

build:
	@set -euo pipefail; \
	if [ -z "$(WORKFLOWS)" ]; then echo "no workflows under cmd/"; exit 1; fi; \
	mkdir -p "$(DIST)"; \
	for name in $(WORKFLOWS); do \
		test -f "cmd/$$name/info.plist" || { echo "missing cmd/$$name/info.plist"; exit 1; }; \
		test -d "cmd/$$name" || { echo "missing cmd/$$name"; exit 1; }; \
		echo "build $$name"; \
		rm -rf "$(DIST)/$$name"; \
		mkdir -p "$(DIST)/$$name"; \
		CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "-s -w" -o "$(DIST)/$$name/$$name.arm64" "./cmd/$$name"; \
		CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o "$(DIST)/$$name/$$name.amd64" "./cmd/$$name"; \
		lipo -create -output "$(DIST)/$$name/$$name" "$(DIST)/$$name/$$name.arm64" "$(DIST)/$$name/$$name.amd64"; \
		rm -f "$(DIST)/$$name/$$name.arm64" "$(DIST)/$$name/$$name.amd64"; \
		chmod 755 "$(DIST)/$$name/$$name"; \
		codesign --force --sign - "$(DIST)/$$name/$$name"; \
		cp "cmd/$$name/info.plist" "$(DIST)/$$name/info.plist"; \
		if [ -f "cmd/$$name/icon.png" ]; then cp "cmd/$$name/icon.png" "$(DIST)/$$name/icon.png"; fi; \
	done

package: build
	@set -euo pipefail; \
	for name in $(WORKFLOWS); do \
		rm -f "$(DIST)/$$name.alfredworkflow"; \
		(cd "$(DIST)/$$name" && files=(info.plist "$$name") && [ -f icon.png ] && files+=(icon.png); zip -qr "../$$name.alfredworkflow" "$${files[@]}"); \
		echo "packed $(DIST)/$$name.alfredworkflow"; \
	done

import: package
	@set -euo pipefail; \
	for name in $(WORKFLOWS); do \
		echo "import $(DIST)/$$name.alfredworkflow"; \
		open -a "$(ALFRED)" "$(DIST)/$$name.alfredworkflow"; \
	done

clean:
	rm -rf "$(DIST)"
