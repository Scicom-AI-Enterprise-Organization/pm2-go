# syntax=docker/dockerfile:1
FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=docker
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.Version=${VERSION}" \
    -o /out/pm2-go ./cmd/pm2-go

# Runtime stage. Alpine gives us /bin/sh, /bin/sleep, /usr/bin/true|false and
# friends so the supervisor can spawn realistic child processes.
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tini bash
COPY --from=builder /out/pm2-go /usr/local/bin/pm2-go

ENV PM2_GO_HOME=/data \
    PM2_GO_API_ADDR=0.0.0.0:9615
RUN mkdir -p /data && chown -R nobody:nogroup /data
USER nobody
WORKDIR /data
EXPOSE 9615

ENTRYPOINT ["/sbin/tini", "--", "/usr/local/bin/pm2-go"]
CMD ["__daemon"]
