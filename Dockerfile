# syntax=docker/dockerfile:1.7

FROM node:24-bookworm AS web-builder
WORKDIR /src

RUN corepack enable

COPY web/package.json web/pnpm-lock.yaml ./web/
WORKDIR /src/web
RUN pnpm install --frozen-lockfile

COPY web/ ./
RUN pnpm build

FROM golang:1.23-bookworm AS go-builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=web-builder /src/internal/adminui/dist ./internal/adminui/dist

RUN go generate ./internal/bootstrap/mod

ARG VERSION=dev
ARG COMMIT=none
ARG BUILD=local

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags "-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.Build=${BUILD}" \
    -o /out/backend-go \
    ./cmd/server

FROM debian:bookworm-slim AS runtime
WORKDIR /app

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl tzdata \
    && rm -rf /var/lib/apt/lists/*

COPY --from=go-builder /out/backend-go /app/backend-go

RUN groupadd --gid 10001 backend-go \
    && useradd --uid 10001 --gid backend-go --home-dir /app --shell /usr/sbin/nologin --no-create-home backend-go \
    && mkdir -p /app/config /app/databases /app/storage \
    && chown -R backend-go:backend-go /app

USER backend-go

ENV GIN_MODE=release \
    CONFIG_DIR=/app \
    HTTP_ADDR=0.0.0.0:8080 \
    HTTP_ADMIN_ADDR=0.0.0.0:8081

EXPOSE 8080 8081

ENTRYPOINT ["/app/backend-go"]
