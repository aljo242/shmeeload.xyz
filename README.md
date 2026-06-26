# shmeeload.xyz

[![CI](https://github.com/aljo242/shmeeload.xyz/actions/workflows/go.yml/badge.svg)](https://github.com/aljo242/shmeeload.xyz/actions/workflows/go.yml) [![go report](https://goreportcard.com/badge/github.com/aljo242/shmeeload.xyz)](https://goreportcard.com/report/github.com/aljo242/shmeeload.xyz) [![license](https://img.shields.io/badge/license-MIT-blue.svg?style=flat)](./LICENSE)

My personal website. A single self-contained Go server (gorilla/mux +
gorilla/websocket) that embeds the whole site, serves the static assets, and runs the
shmeechat websocket hub. The frontend is TypeScript compiled to JavaScript at build time.

- The whole site is embedded with `//go:embed`; the binary is the only artifact
- Text assets minified (HTML/CSS/JS) and served with precomputed brotli/zstd/gzip;
  build-time WebP for images; content-hash ETags (`If-None-Match` → 304)
- Terminates TLS itself (self-signed cert), speaks HTTP/2, advertises HTTP/3 (QUIC)
- WebSockets for shmeechat
- Pages use origin-relative URLs, so the same build works at any host, port, or scheme

## Layout

- `main.go` — config load, cert setup, routing, HTTP/2 + HTTP/3 serving, graceful shutdown
- `staticsite.go` — the embedded static handler (compression + WebP + ETag negotiation)
- `tls.go` — self-signed certificate generation
- `embed.go` — `//go:embed site` and the embedded filesystem
- `cmd/genwebp/` — build-time tool that generates the WebP image variants
- `handlers/` — the `/donate` handler (the only remaining dynamic file handler)
- `client.go`, `hub.go` — the websocket chat hub
- `site/` — the embedded site tree (HTML pages, `static/`, `files/`)
- `web_res/` — TypeScript sources, compiled into `site/static/js/` at build time
- `deploy/` — Docker Compose stack for the homelab ([deploy/README.md](deploy/README.md))

## Develop

Requires Go and the TypeScript compiler (`tsc`).

```sh
make build    # genwebp + tsc + go build -> ./server
make test     # go test -race + coverage
make lint     # golangci-lint (installs the pinned version)
```

The server reads a JSON config (`-c <file>`, default `sample/sample_config.json`)
controlling host/IP/port, the TLS toggle, cert paths and hostnames, cache max-age, and
log level. At startup it indexes the embedded site, precomputes the compressed and WebP
variants' bookkeeping, and (when TLS is on) ensures a self-signed certificate exists.

## Run it (Docker)

```sh
cd deploy
docker compose up -d --build
```

This builds the image (TypeScript compile + WebP image generation + static Go binary) and
serves the site over HTTPS. The single binary embeds the whole site and terminates TLS
itself with a self-signed certificate, so there is no reverse proxy. See
[deploy/README.md](deploy/README.md) for the full homelab setup (Raspberry Pi + Pi-hole
DNS).

## CI

Build, test, and lint run on every push and pull request to `master`, on a self-hosted
x86 runner (no GitHub-hosted runners are used).
