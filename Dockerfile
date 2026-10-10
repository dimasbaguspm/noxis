# noxis as a container: one process (REST API + console + the SQL module).
#   docker run -d -p 9000:9000 -v noxis-data:/data \
#     -e NOXIS_LISTEN=:9000 -e NOXIS_DB=/data/noxis.db noxis run server
# The operator console (React+TS) — built once, dist copied into the
# embeddable static dir before the Go build.
FROM node:22-alpine AS web
WORKDIR /web
RUN npm install -g pnpm@10
COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /bin/dist /src/bin/dist
RUN rm -rf cmd/noxis/server/handlers/static/assets cmd/noxis/server/handlers/static/index.html \
    && cp -r bin/dist/* cmd/noxis/server/handlers/static \
    && CGO_ENABLED=0 go build -o /out/noxis ./cmd/noxis

FROM alpine:3.20
RUN apk add --no-cache docker-cli ca-certificates
COPY --from=build /out/noxis /usr/local/bin/noxis
ENTRYPOINT ["noxis"]