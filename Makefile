.DEFAULT_GOAL := help

GO ?= go
BIN := bin

ifneq (,$(wildcard .env))
include .env
export
endif

.PHONY: help build run test lint fmt tidy mocks spec web uat uat-slow clean

help: ## list the targets
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-8s\033[0m %s\n", $$1, $$2}'

# What this build is: the release line and the seven characters of the commit it came from. The publish
# job passes the same two values to `go build -ldflags`, so a binary says which build it is wherever it runs.
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short=7 HEAD 2>/dev/null || echo unknown)
LDFLAGS := -s -w -X noxis/internal/constants.Version=$(VERSION) -X noxis/internal/constants.Commit=$(COMMIT)

build: ## compile the one binary: bin/noxis (http serve + console, cli)
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN)/noxis ./cmd/noxis

web: ## build the console into internal/web/dist (pnpm, pinned; `pnpm -C web install` once)
	pnpm -C web install --frozen-lockfile
	pnpm -C web run build

run: ## run the API on the constant listen address
	$(GO) run ./cmd/noxis http serve

test: ## unit + architecture tests
	$(GO) test ./...

cover: ## per-package coverage against the committed floors
	$(GO) test ./... -cover | $(GO) run ./test/coverage

lint: ## golangci-lint
	golangci-lint run ./...

fmt: ## gofmt the tree
	$(GO) fmt ./...

mocks: ## regenerate the gomock mocks for every port (committed; CI fails on drift)
	$(GO) tool mockgen -source=internal/ports/ports.go -destination=internal/mocks/ports.go -package=mocks

spec: ## regenerate the OpenAPI document from the handler annotations (committed; CI fails on drift)
	$(GO) tool swag init --v3.1 --parseInternal --parseDependency \
		--outputTypes json,yaml -g serve.go -d ./cmd/noxis/http -o ./cmd/noxis/http/docs
	$(GO) run ./cmd/spec

tidy: ## settle go.mod
	$(GO) mod tidy

uat: ## acceptance suite (godog over a real `noxis http serve` process)
	$(GO) test ./test/uat/... -count=1 -timeout 10m

uat-slow: ## acceptance suite including @slow: boots MySQL 8.4 + MariaDB 11.4 (needs docker)
	UAT_SLOW=1 $(GO) test ./test/uat/... -count=1 -timeout 15m

build-uat:
	$(GO) test ./test/uat/... -count=1 -timeout 10m -tags uat_verbose

clean: ## drop build output
	rm -rf $(BIN)
