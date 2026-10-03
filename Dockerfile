# noxis as a container: master or slave, same image, role by env.
# Slaves need the docker socket mounted so doorman can manage engines:
#   docker run -d -v /var/run/docker.sock:/var/run/docker.sock \
#     -e NOXIS_MASTER=... -e NOXIS_NODE=zeus noxis slave
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
RUN rm -rf cmd/noxis/master/controller/static && cp -r bin/dist cmd/noxis/master/controller/static \
    && CGO_ENABLED=0 go build -o /out/noxis ./cmd/noxis

FROM alpine:3.20
RUN apk add --no-cache docker-cli ca-certificates
COPY --from=build /out/noxis /usr/local/bin/noxis
ENTRYPOINT ["noxis"]