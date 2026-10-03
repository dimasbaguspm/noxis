PATH := $(HOME)/.local/bin:$(HOME)/go/bin:$(PATH)

.PHONY: build test lint swagger web dev down uat

build:
	go build -o bin/noxis ./cmd/noxis

web:  ## build the React console (web/, pnpm) into the embedded static dir
	cd web && pnpm install --frozen-lockfile && pnpm build
	rm -rf cmd/noxis/master/controller/static
	mkdir -p cmd/noxis/master/controller/static
	cp -r bin/dist/* cmd/noxis/master/controller/static/

test:
	go test ./...

.PHONY: uat
uat:
	UAT_TAGS="$${UAT_TAGS:-~@slow}" go test ./test/uat -count=1

lint:  ## run golangci-lint (installs it on first use)
	command -v golangci-lint >/dev/null 2>&1 || go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
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

dev:  ## build + run the dev stack (docker compose; registry lives in the noxis-data volume)
	docker compose -f docker/docker-compose.yml up --build

down:  ## stop the dev stack and wipe its data (compose volumes, slave containers/volumes/network, registry) — start fresh later with make dev
	docker compose -f docker/docker-compose.yml down -v
	@docker rm -f $$(docker ps -aq --filter label=noxis.engine) 2>/dev/null || true
	@docker volume rm -f $$(docker volume ls -q --filter name=noxis-zeus) 2>/dev/null || true
	@docker network rm noxis-zeus 2>/dev/null || true
	@rm -rf docker/db.sqlite
