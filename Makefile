PATH := $(HOME)/.local/bin:$(HOME)/go/bin:$(PATH)

.PHONY: build test lint swagger web up dev dev-ui down uat

build:
	cd apps/api && go build -o ../../bin/noxis ./cmd/noxis

web:  ## build the React console (apps/web, pnpm) into the embedded static dir
	cd apps/web && pnpm install --frozen-lockfile && pnpm build
	# keep placeholder.html: go:embed needs a file, and a bare `go build` stays valid
	rm -rf apps/api/cmd/noxis/server/handlers/static/assets apps/api/cmd/noxis/server/handlers/static/index.html
	cp -r bin/dist/* apps/api/cmd/noxis/server/handlers/static/

test:
	cd apps/api && go test ./...

.PHONY: uat
uat:
	cd apps/api && UAT_TAGS="$${UAT_TAGS:-~@slow}" go test ./test/uat -count=1

lint:  ## run golangci-lint (installs it on first use)
	command -v golangci-lint >/dev/null 2>&1 || go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0
	cd apps/api && golangci-lint run ./...

swagger:  ## regenerate openapi.yaml|json (OpenAPI 3) from handler annotations
	cd apps/api && go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/noxis/server/server.go --output ../../.swagger-tmp
	rm -f .swagger-tmp/docs.go
	# swag emits Swagger 2.0; convert to OpenAPI 3 for Postman
	npx --yes swagger2openapi -y .swagger-tmp/swagger.json -o openapi.yaml
	npx --yes swagger2openapi .swagger-tmp/swagger.json -o openapi.json
	rm -rf .swagger-tmp

up:  ## run the first-instance stack (noxis only) from deploy/; needs deploy/.env
	docker compose -f deploy/docker-compose.yml --env-file deploy/.env up -d --build

dev:  ## build + run the dev stack (noxis + a target postgres for playing)
	docker compose -f docker-compose.yml up --build

dev-ui:  ## dev stack + the hot-reloading console on :5173
	docker compose -f docker-compose.yml --profile dev up --build

down:  ## stop the dev stack and wipe its data (volumes incl. the registry)
	docker compose -f docker-compose.yml --profile dev down -v
