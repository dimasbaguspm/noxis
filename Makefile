PATH := $(HOME)/.local/bin:$(HOME)/go/bin:$(PATH)

.PHONY: build test lint swagger web dev dev-ui down uat mocks

build:
	cd apps/api && go build -o ../../bin/noxis ./cmd/noxis

web:  ## build the React console (apps/web, pnpm) into the embedded static dir
	cd apps/web && pnpm install --frozen-lockfile && pnpm build
	# keep placeholder.html: go:embed needs a file, and a bare `go build` stays valid
	rm -rf apps/api/cmd/noxis/server/static/assets apps/api/cmd/noxis/server/static/index.html
	cp -r bin/dist/* apps/api/cmd/noxis/server/static/

test:
	cd apps/api && go test ./...

.PHONY: uat
uat:
	cd apps/api && UAT_TAGS="$${UAT_TAGS:-~@slow}" go test ./test/uat -count=1

lint:  ## run golangci-lint (installs it on first use)
	command -v golangci-lint >/dev/null 2>&1 || go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0
	cd apps/api && golangci-lint run ./...

mocks:  ## regenerate the test doubles from internal/interfaces
	cd apps/api && go generate ./internal/mocks

swagger:  ## regenerate openapi.yaml|json (OpenAPI 3) from handler annotations
	cd apps/api && go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/noxis/server/server.go --output ../../.swagger-tmp
	rm -f .swagger-tmp/docs.go
	# swag emits Swagger 2.0; convert to OpenAPI 3 for Postman
	npx --yes swagger2openapi -y .swagger-tmp/swagger.json -o openapi.yaml
	npx --yes swagger2openapi .swagger-tmp/swagger.json -o openapi.json
	rm -rf .swagger-tmp

# `make dev <target>`: the goal after dev names the target, and it must not look like a makefile target of its own.
DEV_TARGETS := postgres mysql mariadb cockroach redis valkey dragonfly mongodb couchdb elasticsearch meilisearch splunk
DEV_SEED_all := postgres mysql mariadb cockroach redis valkey dragonfly mongo couchdb elasticsearch meilisearch
ifeq (dev,$(firstword $(MAKECMDGOALS)))
  DEV_TARGET := $(word 2,$(MAKECMDGOALS))
  ifneq (,$(DEV_TARGET))
    $(eval $(DEV_TARGET):;@:)
  endif
endif

dev:  ## start one target with its seed data: make dev postgres|mysql|mariadb|cockroach|redis|valkey|dragonfly|mongodb|couchdb|elasticsearch|meilisearch|splunk|all
	@case "$(DEV_TARGET)" in \
	  "") \
	    echo "usage: make dev <target>"; \
	    echo "  targets: $(DEV_TARGETS) all"; \
	    echo "  example: make dev postgres   # noxis + postgres, seeded, console on :9000"; \
	    exit 1 ;; \
	  all) \
	    SEED_DRIVERS="$(DEV_SEED_all)" docker compose -f docker-compose.yml --profile all --profile seed up --build ;; \
	  postgres|mysql|mariadb|cockroach|redis|valkey|dragonfly|couchdb|elasticsearch|meilisearch|splunk) \
	    SEED_DRIVERS="$(DEV_TARGET)" docker compose -f docker-compose.yml --profile $(DEV_TARGET) --profile seed up --build ;; \
	  mongodb) \
	    SEED_DRIVERS="mongo" docker compose -f docker-compose.yml --profile mongo --profile seed up --build ;; \
	  *) \
	    echo "unknown target: $(DEV_TARGET)"; \
	    echo "  targets: $(DEV_TARGETS) all"; \
	    exit 1 ;; \
	esac

dev-ui:  ## every target + the hot-reloading console on :5173
	SEED_DRIVERS="$(DEV_SEED_all)" docker compose -f docker-compose.yml --profile all --profile seed --profile dev up --build

down:  ## stop the dev stack and wipe its data (volumes incl. the registry)
	docker compose -f docker-compose.yml --profile all --profile seed --profile dev down -v
