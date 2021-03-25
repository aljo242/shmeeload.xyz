BINARY_NAME = server
ARM = arm
MY_ARCH = $(shell go env GOARCH)

.PHONY: all
all: analyze build test 

.PHONY: build
build:
	cd ./web_res && tsc
	go fmt
	go build -o ${BINARY_NAME}

.PHONY: analyze
analyze:
	golint
	go vet
	go fmt
	gosec ./...

.PHONY: test
test:
# cannot use "-race" flag on ARM systems
ifeq ($(MY_ARCH), $(ARM))
	go test -v  -coverprofile=coverage.out
else 
	go test -v -race -coverprofile=coverage.out
endif
	go tool cover -html coverage.out -o coverage.html

.PHONY: clean
clean: 
ifneq ("$(wildcard ./web_res/dist/)", "")
	rm -rf ./web_res/dist/*
endif
ifneq ("$(wildcard ${BINARY_NAME})", "")
	rm ${BINARY_NAME}
endif
ifneq ("$(wildcard coverage.html)", "")
	rm coverage.html
endif
ifneq ("$(wildcard coverage.out)", "")
	rm coverage.out
endif
ifneq ("$(wildcard serviceWorker.js)", "")
	rm -f serviceWorker.js 
endif
ifneq ("$(wildcard serviceWorker.js.map)", "")
	rm -f serviceWorker.js.map
endif
	sudo rm -rf static/
	go clean

.PHONY: run
run: build
	sudo ./${BINARY_NAME}


