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

build: format
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

markdownLintImage=tmknom/markdownlint
containerMarkdownLint=$(PROJECT_NAME)-markdownlint
containerMarkdownLintFix=$(PROJECT_NAME)-markdownlint-fix

lint:
	@golangci-lint run -c ./.golangci.yml --out-format=tab --issues-exit-code=0
	@# @if $(DOCKER) ps -a --format '{{.Names}}' | grep -Eq "^${containerMarkdownLint}$$"; then $(DOCKER) start -a $(containerMarkdownLint); else $(DOCKER) run --name $(containerMarkdownLint) -i -v "$(CURDIR):/work" $(markdownLintImage); fi


FIND_ARGS := -name '*.go' -type f -not -path "./sample_txs*" -not -path "*.git*" -not -path "./build_report/*" -not -path "./scripts*" -not -name '*.pb.go'

format:
	@find . $(FIND_ARGS) | xargs gofmt -w -s
	@find . $(FIND_ARGS) | xargs goimports -w -local github.com/aljo242/shmeeload.xyz


.PHONY: lint format


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



