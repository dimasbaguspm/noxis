# noxis as a container: one process (REST API + console + the engine modules).
#   docker run -d -p 9000:9000 -v noxis-data:/data \
#     -e NOXIS_LISTEN=:9000 -e NOXIS_DB=/data/noxis.db noxis run server
# Full image — the operator console (apps/web, React+TS) is built once and its
# dist copied into the embeddable static dir before the Go build.
# API only: Dockerfile.api skips the console and serves the placeholder.
FROM node:22-alpine AS web
WORKDIR /web
RUN npm install -g pnpm@10
COPY apps/web/package.json apps/web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY apps/web/ ./
RUN pnpm build

FROM golang:1.26-alpine AS build
WORKDIR /src/apps/api
COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download
COPY apps/api/ ./
# vite writes to <repo>/bin/dist, which is /bin/dist inside this stage
COPY --from=web /bin/dist /src/dist
RUN rm -rf cmd/noxis/server/handlers/static/assets cmd/noxis/server/handlers/static/index.html \
    && cp -r /src/dist/* cmd/noxis/server/handlers/static/ \
    && CGO_ENABLED=0 go build -o /out/noxis ./cmd/noxis

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /out/noxis /usr/local/bin/noxis
ENTRYPOINT ["noxis"]
