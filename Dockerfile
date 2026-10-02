# noxis as a container: master or slave, same image, role by env.
# Slaves need the docker socket mounted so doorman can manage engines:
#   docker run -d -v /var/run/docker.sock:/var/run/docker.sock \
#     -e NOXIS_MASTER=... -e NOXIS_NODE=zeus noxis slave
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/noxis ./cmd/noxis

FROM alpine:3.20
RUN apk add --no-cache docker-cli ca-certificates
COPY --from=build /out/noxis /usr/local/bin/noxis
ENTRYPOINT ["noxis"]