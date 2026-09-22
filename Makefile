PLUGIN_UUID := com.valentderah.just-monitor-control
DIST        := dist
BUNDLE      := $(DIST)/$(PLUGIN_UUID).sdPlugin
STREAMDECK  := streamdeck

ifeq ($(OS),Windows_NT)
  BUILD_CMD  := powershell -NoProfile -ExecutionPolicy Bypass -File scripts/build.ps1
  MKDIR_DIST := powershell -NoProfile -ExecutionPolicy Bypass -Command "New-Item -ItemType Directory -Force -Path '$(DIST)' | Out-Null"
  CLEAN_CMD  := powershell -NoProfile -ExecutionPolicy Bypass -Command "Remove-Item -Recurse -Force '$(DIST)','bin' -ErrorAction SilentlyContinue"
else
  BUILD_CMD  := bash scripts/build.sh
  MKDIR_DIST := mkdir -p "$(DIST)"
  CLEAN_CMD  := rm -rf "$(DIST)" bin/
endif

.PHONY: all build package pack zip validate link test clean release help

all: package

help:
	@echo Targets:
	@echo   make build     - assemble plugin bundle into $(BUNDLE)/
	@echo   make package   - build + pack .streamDeckPlugin into $(DIST)/
	@echo   make validate  - validate $(BUNDLE) with Stream Deck CLI
	@echo   make link      - link $(BUNDLE) into Stream Deck (dev install)
	@echo   make test      - run tests
	@echo   make clean     - remove build artifacts
	@echo   make release   - clean build and package

build:
	$(BUILD_CMD)

package pack zip: build
	$(MKDIR_DIST)
	$(STREAMDECK) pack -f --no-update-check -o "$(DIST)" "$(BUNDLE)"

validate: build
	$(STREAMDECK) validate --no-update-check "$(BUNDLE)"

link: build
	$(STREAMDECK) link "$(BUNDLE)"

test:
	go test ./...

clean:
	$(CLEAN_CMD)

release: clean package