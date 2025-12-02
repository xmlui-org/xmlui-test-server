
# Minimal Makefile — all logic in ./scripts/*.sh
SHELL := /usr/bin/env bash
.DEFAULT_GOAL := help

# Convenience: if a script is missing or not executable, print guidance.
define run
	@if [[ ! -x "./scripts/$(1).sh" ]]; then \
		echo "scripts/$(1).sh not found or not executable. Create it and chmod +x."; \
		exit 1; \
	fi; \
	./scripts/$(1).sh $(2)
endef

.PHONY: help ensure-valid deps tidy fmt vet lint build run test cover clean doterr

help:    ## Show this help message
	$(call run,help)

ensure-valid: tidy test lint vet

deps:    ## Download & tidy modules
	$(call run,deps)

tidy:    ## go mod tidy
	$(call run,tidy)

fmt:     ## gofmt (+ goimports if present)
	$(call run,fmt)

vet:     ## go vet
	$(call run,vet)

lint:    ## golangci-lint (fallback to vet)
	$(call run,lint)

build:   ## Build ./cmd -> ./bin/xmlui-test-server
	$(call run,build)

run:     ## Build then run; pass args like: make run -- --help
	$(call run,run,$(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS)))

test:    ## Run all tests (unit + integration); can target dirs: make test xmluisvr
	$(call run,test,$(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS)))

cover:   ## Coverage report -> bin/coverage.html
	$(call run,cover)

clean:   ## Remove bin/
	$(call run,clean)

doterr:  ## Sync doterr
	@~/Projects/go-pkgs/go-doterr/sync.sh

# Allow passing args to targets like: make test xmluisvr or make run -- --help
%:
	@:

