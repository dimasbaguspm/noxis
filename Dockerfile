# syntax=docker/dockerfile:1
#
# One binary. The build stage compiles ./cmd/noxis with CGO off, so the result is a static
# executable; the runtime stage is distroless: no shell, no package manager, no libc to exploit,
# and a nonroot user that owns nothing but its own state file.

# The console: built here, never committed. `internal/web/dist` holds only a placeholder in the
# repository, so the image is the one place a production build exists.
FROM --platform=$BUILDPLATFORM node:22-alpine AS console
RUN corepack enable
WORKDIR /web
# The lockfile names the graph that was tested, and .npmrc names how it is installed.
COPY web/package.json web/pnpm-lock.yaml web/.npmrc ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
# vite writes to ../internal/web/dist, which is /internal/web/dist from here.
RUN pnpm run build

FROM --platform=$BUILDPLATFORM golang:1.26 AS build
WORKDIR /src

# What this image is. `docker build --build-arg VERSION=v2.0.0 --build-arg COMMIT=$(git rev-parse --short=7 HEAD)`
# stamps it; a plain build says `dev` and claims no commit, which is the honest answer.
ARG VERSION=dev
ARG COMMIT=unknown


# Both architectures, from one build: the compile happens on the build platform and crosses over, so a
# multi-arch push costs a second compile rather than an emulated toolchain. `docker compose build` and
# `docker buildx build` set these; a plain `docker build` falls back to linux/amd64.
ARG TARGETOS=linux
ARG TARGETARCH=amd64
# Dependencies first: a source change does not re-download the module graph.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# The console is embedded by the Go build, so it has to be in place before it runs.
COPY --from=console /internal/web/dist ./internal/web/dist
# The repository's placeholder carries the marker that says "not built"; the real build must not.
RUN rm -f ./internal/web/dist/NOT_BUILT
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath \
      -ldflags "-s -w -X noxis/internal/constants.Version=${VERSION} -X noxis/internal/constants.Commit=${COMMIT}" \
      -o /out/noxis ./cmd/noxis
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
