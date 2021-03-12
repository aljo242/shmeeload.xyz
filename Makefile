BINARY_NAME = server

all: build test

build:
	cd ./web_res && tsc
	go build -o ${BINARY_NAME}

test:
	go test -v

clean: 
	rm -rf ./web_res/dist/*
	rm ${BINARY_NAME}

run: build
	sudo ./${BINARY_NAME}


