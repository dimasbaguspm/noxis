# syntax=docker/dockerfile:1
#
# One binary. The build stage compiles ./cmd/noxis with CGO off, so the result is a static
# executable; the runtime stage is distroless: no shell, no package manager, no libc to exploit,
# and a nonroot user that owns nothing but its own state file.

# The console: built here, never committed. `internal/web/dist` holds only a placeholder in the
# repository, so the image is the one place a production build exists.
FROM node:22-alpine AS console
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
# vite writes to ../internal/web/dist, which is /internal/web/dist from here.
RUN npm run build

FROM golang:1.26 AS build
WORKDIR /src

# Dependencies first: a source change does not re-download the module graph.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# The console is embedded by the Go build, so it has to be in place before it runs.
COPY --from=console /internal/web/dist ./internal/web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -o /out/noxis ./cmd/noxis
# /data belongs to the runtime user, so a named volume inherits an owner that can write to it.
RUN mkdir -p /out/data && chown 65532:65532 /out/data

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/noxis /noxis
COPY --from=build --chown=nonroot:nonroot /out/data /data

# The state file lives on a volume the operator mounts; the image itself is read-only.
VOLUME ["/data"]

USER nonroot:nonroot
EXPOSE 9000 3000

# The binary is its own healthcheck: no curl, no wget, nothing to add to the image.
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/noxis", "http", "healthcheck"]

ENTRYPOINT ["/noxis"]
CMD ["http", "serve", "--state", "/data/noxis.db"]
