#!/usr/bin/make -f

GOBIN ?= $(GOPATH)/BINARY_NAME
VERSION := $(shell echo $(shell git describe --tags 2> /dev/null || echo "dev-$(shell git describe --always)") | sed 's/^v//')
COMMIT := $(shell git log -1 --format='%H')
PROJECT_NAME = $(shell git remote get-url origin | xargs basename -s .git)

BINARY_NAME = server
ARM = arm
MY_ARCH = $(shell go env GOARCH)

# Our Go packages, excluding anything under web_res (npm deps can ship stray
# .go files, e.g. the "flatted" package, which would otherwise be built/linted).
PACKAGES = $(shell go list ./... | grep -v /web_res/)

export GO111MODULE = on

###############################################################################
###                               Build                                     ###
###############################################################################

all: lint build test 

build: webp
	@cd ./web_res && npm ci --no-audit --no-fund && npm run build
	@go build -o ${BINARY_NAME}

# Precompute WebP variants of the site images so the binary embeds them and
# serves them without encoding anything at startup.
webp:
	@echo "--> Generating WebP image variants"
	@go run ./cmd/genwebp ./site

.PHONY: webp

clean:
ifneq ("$(wildcard ./web_res/dist/)", "")
	@rm -rf ./web_res/dist/*
endif
ifneq ("$(wildcard ${BINARY_NAME})", "")
	@rm ${BINARY_NAME}
endif
ifneq ("$(wildcard coverage.html)", "")
	@rm coverage.html
endif
ifneq ("$(wildcard coverage.out)", "")
	@rm coverage.out
endif
	@find ./site -name '*.webp' -delete 2>/dev/null || true
	@go clean

.PHONY: build clean


###############################################################################
###                                Linting                                  ###
###############################################################################
golangci_lint_cmd=golangci-lint
golangci_version=v2.12.2
govulncheck_version=v1.5.0

lint:
	@echo "--> Running linter"
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(golangci_version)
	@$(golangci_lint_cmd) run --timeout=10m


lint-fix:
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(golangci_version)
	@$(golangci_lint_cmd) run --timeout=10m --fix

lint-web:
	@echo "--> Running TypeScript linter"
	@cd ./web_res && npm ci --no-audit --no-fund && npm run lint

vuln:
	@echo "--> Running govulncheck"
	@go run golang.org/x/vuln/cmd/govulncheck@$(govulncheck_version) $(PACKAGES)

.PHONY: lint lint-fix lint-web vuln

###############################################################################
###                                Testing                                  ###
###############################################################################

test: test-unit

test-unit:
# cannot use "-race" flag on ARM systems
ifeq ($(MY_ARCH), $(ARM))
	@go test -v -coverprofile=coverage.out $(PACKAGES)
else
	@go test -v -race -coverprofile=coverage.out $(PACKAGES)
endif
	@go tool cover -html coverage.out -o coverage.html

.PHONY: test test-unit


###############################################################################
###                               Commands                                  ###
###############################################################################

run: build
	./${BINARY_NAME}

.PHONY: run



