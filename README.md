# shmeeload.xyz

[![CI](https://github.com/aljo242/shmeeload.xyz/actions/workflows/go.yml/badge.svg)](https://github.com/aljo242/shmeeload.xyz/actions/workflows/go.yml) [![go report](https://goreportcard.com/badge/github.com/aljo242/shmeeload.xyz)](https://goreportcard.com/report/github.com/aljo242/shmeeload.xyz) [![license](https://img.shields.io/badge/license-MIT-blue.svg?style=flat)](./LICENSE)

My personal website. A Go server (gorilla/mux + gorilla/websocket) that renders the
pages, serves the static assets, and runs the shmeechat websocket hub. The frontend is
TypeScript compiled to JavaScript at build time.

- TypeScript to JavaScript build pipeline
- WebSockets for shmeechat
- Pages use origin-relative URLs, so the same build works at any host, port, or scheme
- Uses the [chef](https://github.com/aljo242/chef) library for config loading and the server wrapper

## Layout

- `main.go` — config load, template/asset setup, routing, graceful shutdown
- `handlers/` — HTTP handlers; all file serving funnels through `serveFile` / `servePage`
- `client.go`, `hub.go` — the websocket chat hub
- `web_res/` — HTML, CSS, and TypeScript sources (compiled into `web_res/dist/`)
- `deploy/` — Docker Compose stack for the homelab ([deploy/README.md](deploy/README.md))

## Develop

Requires Go and the TypeScript compiler (`tsc`).

```sh
make build    # tsc + go build -> ./server
make test     # go test -race + coverage
make lint     # golangci-lint (installs the pinned version)
```

The server reads a JSON config (`-c <file>`, default `sample/sample_config.json`)
controlling host/IP/port, the TLS toggle, cache max-age, and log level. At startup it
renders the HTML templates and copies `web_res/` into `./static/`, which it then serves.

## Run it (Docker)

```sh
cd deploy
docker compose up -d --build
```

This builds the image (TypeScript compile + static Go binary) and serves the site behind
Caddy over HTTPS. See [deploy/README.md](deploy/README.md) for the full homelab setup
(Raspberry Pi + Caddy internal CA + Pi-hole DNS).

## CI

Build, test, and lint run on every push and pull request to `master`, on a self-hosted
x86 runner (no GitHub-hosted runners are used).
