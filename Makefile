PATH := $(HOME)/.local/bin:$(HOME)/go/bin:$(PATH)

.PHONY: build test lint swagger web dev down uat

build:
	go build -o bin/noxis ./cmd/noxis

web:  ## build the React console (web/, pnpm) into the embedded static dir
	cd web && pnpm install --frozen-lockfile && pnpm build
	# keep placeholder.html: go:embed needs a file, and a bare `go build` stays valid
	rm -rf cmd/noxis/server/handlers/static/assets cmd/noxis/server/handlers/static/index.html
	cp -r bin/dist/* cmd/noxis/server/handlers/static/

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
	swag init -g cmd/noxis/server/server.go --output ./doc
	rm -f doc/docs.go
	# swag emits Swagger 2.0; convert to OpenAPI 3 for Postman
	npx --yes swagger2openapi -y doc/swagger.json -o doc/openapi3.yaml
	npx --yes swagger2openapi doc/swagger.json -o doc/openapi3.json
	mv doc/openapi3.yaml doc/swagger.yaml
	mv doc/openapi3.json doc/swagger.json

dev:  ## build + run the dev stack (noxis + a target postgres for playing)
	docker compose -f docker/docker-compose.yml up --build

dev-ui:  ## dev stack + the hot-reloading console on :5173
	docker compose -f docker/docker-compose.yml --profile dev up --build

down:  ## stop the dev stack and wipe its data (volumes incl. the registry)
	docker compose -f docker/docker-compose.yml --profile dev down -v
