PATH := $(HOME)/.local/bin:$(HOME)/go/bin:$(PATH)

.PHONY: build test lint swagger db dev down

build:
	go build -o bin/noxis ./cmd/noxis

test:
	go test ./...

lint:  ## run golangci-lint (installs it on first use)
	command -v golangci-lint >/dev/null 2>&1 || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run ./...

swagger:  ## regenerate doc/swagger.yaml|json (OpenAPI 3) from handler annotations
	go install github.com/swaggo/swag/cmd/swag@latest
	swag init -g cmd/noxis/master/routes.go --output ./doc
	rm -f doc/docs.go
	# swag emits Swagger 2.0; convert to OpenAPI 3 for Postman
	npx --yes swagger2openapi -y doc/swagger.json -o doc/openapi3.yaml
	npx --yes swagger2openapi doc/swagger.json -o doc/openapi3.json
	mv doc/openapi3.yaml doc/swagger.yaml
	mv doc/openapi3.json doc/swagger.json

db:   ## create the dev sqlite store under docker/
	@mkdir -p docker
	@test -f docker/db.sqlite || touch docker/db.sqlite

dev: db  ## build + run the dev stack (docker compose)
	docker compose -f docker/docker-compose.yml up --build

down:  ## stop the dev stack
	docker compose -f docker/docker-compose.yml down