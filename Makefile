PATH := $(HOME)/.local/bin:$(HOME)/go/bin:$(PATH)

.PHONY: proto build test db dev down

proto:
	protoc --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	       internal/repository/grpc/control.proto

build:
	go build -o bin/noxis ./cmd/noxis

test:
	go test ./...

db:   ## create the dev sqlite store under docker/
	@mkdir -p docker
	@test -f docker/db.sqlite || touch docker/db.sqlite

dev: db  ## build + run the dev stack (docker compose)
	docker compose -f docker/docker-compose.yml up --build

down:  ## stop the dev stack
	docker compose -f docker/docker-compose.yml down