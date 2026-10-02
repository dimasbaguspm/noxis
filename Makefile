PATH := $(HOME)/.local/bin:$(HOME)/go/bin:$(PATH)

.PHONY: proto build test

proto:
	protoc --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	       internal/core/control.proto

build:
	go build -o bin/noxis ./cmd/noxis

test:
	go test ./...