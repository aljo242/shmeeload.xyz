BINARY_NAME = server

all: analyze build test 

build:
	cd ./web_res && tsc
	go fmt
	go build -o ${BINARY_NAME}

analyze:
	golint
	go vet
	go fmt
	gosec ./...

test:
	go test -v -race -coverprofile=coverage.out
	go tool cover -html coverage.out -o coverage.html

clean: 
	rm -rf ./web_res/dist/*
	rm ${BINARY_NAME}
	sudo rm -rf static/

run: build
	sudo ./${BINARY_NAME}


