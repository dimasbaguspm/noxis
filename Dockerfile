# syntax=docker/dockerfile:1
#
# One binary. The build stage compiles ./cmd/noxis with CGO off, so the result is a static
# executable; the runtime stage is distroless: no shell, no package manager, no libc to exploit,
# and a nonroot user that owns nothing but its own state file.

FROM golang:1.26 AS build
WORKDIR /src

# Dependencies first: a source change does not re-download the module graph.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
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
