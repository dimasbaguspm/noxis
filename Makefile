.DEFAULT_GOAL := help

GO ?= go
BIN := bin

ifneq (,$(wildcard .env))
include .env
export
endif

.PHONY: help build run test lint fmt tidy uat uat-slow clean

help: ## list the targets
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-8s\033[0m %s\n", $$1, $$2}'

build: ## compile bin/noxisd (server) and bin/noxis (cli)
	$(GO) build -o $(BIN)/noxisd ./cmd/http
	$(GO) build -o $(BIN)/noxis ./cmd/cli

run: ## run the server on HTTP_ADDR
	$(GO) run ./cmd/http

test: ## unit + architecture tests
	$(GO) test ./...

lint: ## golangci-lint
	golangci-lint run ./...

fmt: ## gofmt the tree
	$(GO) fmt ./...

tidy: ## settle go.mod
	$(GO) mod tidy

uat: ## acceptance suite (godog over a real noxisd process)
	$(GO) test ./test/uat/... -count=1 -timeout 10m

uat-slow: ## acceptance suite including @slow: boots MySQL 8.4 + MariaDB 11.4 (needs docker)
	UAT_SLOW=1 $(GO) test ./test/uat/... -count=1 -timeout 15m

build-uat:
	$(GO) test ./test/uat/... -count=1 -timeout 10m -tags uat_verbose

clean: ## drop build output
	rm -rf $(BIN)
