PATH := $(HOME)/.local/bin:$(HOME)/go/bin:$(PATH)

.PHONY: proto build test swagger db dev down

proto:
	protoc --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	       internal/services/grpc/control.proto

build:
	go build -o bin/noxis ./cmd/noxis

test:
	go test ./...

swagger:  ## regenerate doc/swagger.yaml|json from handler annotations (Postman artifact)
	go install github.com/swaggo/swag/cmd/swag@latest
	swag init -g cmd/noxis/master/routes.go --output ./doc
	rm -f doc/docs.go  # generated runtime is unused; spec files are the artifact

db:   ## create the dev sqlite store under docker/
	@mkdir -p docker
	@test -f docker/db.sqlite || touch docker/db.sqlite

dev: db  ## build + run the dev stack (docker compose)
	docker compose -f docker/docker-compose.yml up --build

down:  ## stop the dev stack
	docker compose -f docker/docker-compose.yml down