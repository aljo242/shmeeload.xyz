#!/usr/bin/make -f

GOBIN ?= $(GOPATH)/BINARY_NAME
VERSION := $(shell echo $(shell git describe --tags 2> /dev/null || echo "dev-$(shell git describe --always)") | sed 's/^v//')
COMMIT := $(shell git log -1 --format='%H')
PROJECT_NAME = $(shell git remote get-url origin | xargs basename -s .git)

BINARY_NAME = server
ARM = arm
MY_ARCH = $(shell go env GOARCH)

export GO111MODULE = on

###############################################################################
###                               Build                                     ###
###############################################################################

all: lint build test 

build:
	@cd ./web_res && tsc
	@go build -o ${BINARY_NAME}

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
ifneq ("$(wildcard serviceWorker.js)", "")
	@rm -f serviceWorker.js 
endif
ifneq ("$(wildcard serviceWorker.js.map)", "")
	@rm -f serviceWorker.js.map
endif
	@sudo rm -rf static/
	@go clean

.PHONY: build clean


###############################################################################
###                                Linting                                  ###
###############################################################################
golangci_lint_cmd=golangci-lint
golangci_version=v2.1.6

lint:
	@echo "--> Running linter"
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(golangci_version)
	@$(golangci_lint_cmd) run --timeout=10m


lint-fix:
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(golangci_version)
	@$(golangci_lint_cmd) run --timeout=10m --fix

.PHONY: lint lint-fix

###############################################################################
###                                Testing                                  ###
###############################################################################

test: test-unit

test-unit:
# cannot use "-race" flag on ARM systems
ifeq ($(MY_ARCH), $(ARM))
	@go test -v  -coverprofile=coverage.out
else 
	@go test -v -race -coverprofile=coverage.out
endif
	@go tool cover -html coverage.out -o coverage.html

.PHONY: test test-unit


###############################################################################
###                               Commands                                  ###
###############################################################################

run: build
	sudo ./${BINARY_NAME}

.PHONY: run



