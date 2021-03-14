BINARY_NAME = server

all: build test

build:
	cd ./web_res && tsc
	go fmt
	go build -o ${BINARY_NAME}

analyze:
	golint
	go fmt

test:
	go test ./... -v

clean: 
	rm -rf ./web_res/dist/*
	rm ${BINARY_NAME}

run: build
	sudo ./${BINARY_NAME}


