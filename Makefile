.PHONEY: build test

all: build

build:
	protoc --go_out=plugins=grpc:. *.proto && \
	go build -o filewatcher

test:
	go test -v