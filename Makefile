.PHONEY: build build-proto test

all: build

build:
	go mod download
	go build -o filewatcher

build-proto:
	protoc --go_out=plugins=grpc:. *.proto


docker:
	docker build -t filewatcher .

test:
	go test -v