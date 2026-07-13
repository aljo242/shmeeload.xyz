# syntax=docker/dockerfile:1

# ---- Stage 1: compile TypeScript into site/static/js ----
FROM node:26-alpine@sha256:725aeba2364a9b16beae49e180d83bd597dbd0b15c47f1f28875c290bfd255b9 AS web
WORKDIR /app
COPY web_res/package.json web_res/package-lock.json ./web_res/
RUN --mount=type=cache,target=/root/.npm cd web_res && npm ci --no-audit --no-fund
COPY web_res ./web_res
COPY site ./site
# tsc writes to ../site/static/js (see web_res/tsconfig.json).
RUN cd web_res && npm run build

# ---- Stage 2: build the Go binary (embeds the whole site) ----
FROM golang:1.26-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2 AS build
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
FROM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b
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
