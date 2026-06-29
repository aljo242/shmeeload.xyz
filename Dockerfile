# syntax=docker/dockerfile:1

# ---- Stage 1: compile TypeScript into site/static/js ----
FROM node:22-alpine@sha256:16e22a550f3863206a3f701448c45f7912c6896a62de43add43bb9c86130c3e2 AS web
WORKDIR /app
COPY web_res/package.json web_res/package-lock.json ./web_res/
RUN --mount=type=cache,target=/root/.npm cd web_res && npm ci --no-audit --no-fund
COPY web_res ./web_res
COPY site ./site
# tsc writes to ../site/static/js (see web_res/tsconfig.json).
RUN cd web_res && npm run build

# ---- Stage 2: build the Go binary (embeds the whole site) ----
FROM golang:1.26-alpine@sha256:3ad57304ad93bbec8548a0437ad9e06a455660655d9af011d58b993f6f615648 AS build
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
# Bring in the compiled JS so //go:embed site includes it.
COPY --from=web /app/site/static/js ./site/static/js
# Precompute WebP/AVIF image variants so they are embedded too (no startup
# encoding). The /imgcache mount persists generated variants across builds; the
# tool's mtime check then skips unchanged images.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/imgcache \
    set -e; \
    if [ -d /imgcache/img ]; then cp -a /imgcache/img/. ./site/static/img/ 2>/dev/null || true; fi; \
    go run ./cmd/genimg ./site; \
    mkdir -p /imgcache/img; \
    find ./site/static/img \( -name '*.webp' -o -name '*.avif' \) -exec cp -a {} /imgcache/img/ \;
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server .

# ---- Stage 3: runtime (the site is embedded in the binary) ----
FROM alpine:3.20@sha256:d9e853e87e55526f6b2917df91a2115c36dd7c696a35be12163d44e6e2a4b6bc
RUN apk add --no-cache ca-certificates wget \
	&& adduser -D -u 10001 app \
	&& mkdir -p /data && chown app:app /data
WORKDIR /app
COPY --from=build /out/server /app/server
USER app
# /data holds the persisted self-signed TLS cert (a named volume inherits this
# app-owned dir on first mount, so the non-root process can write it).
VOLUME ["/data"]
EXPOSE 443
ENTRYPOINT ["/app/server"]
CMD ["-c", "/app/config.json"]
