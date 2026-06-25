# syntax=docker/dockerfile:1

# ---- Stage 1: compile TypeScript -> JavaScript ----
FROM node:20-alpine AS web
WORKDIR /web
COPY web_res/package.json web_res/package-lock.json ./
RUN npm install --no-audit --no-fund
COPY web_res/ ./
# Invoke the real TypeScript compiler directly. The "tsc" npm package listed in
# package.json is a broken stub; node_modules/typescript/bin/tsc is the actual compiler.
RUN node_modules/typescript/bin/tsc

# ---- Stage 2: build the Go server (static, no CGO) ----
FROM golang:1.26-alpine AS build
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Bring in the JS compiled by the web stage so it ships inside web_res/.
COPY --from=web /web/dist ./web_res/dist
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server .

# ---- Stage 3: runtime ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates wget && adduser -D -u 10001 app
WORKDIR /app
COPY --from=build /out/server /app/server
COPY --from=build /src/web_res /app/web_res
# The server rebuilds ./static on startup, so /app must be writable by the run user.
RUN chown -R app:app /app
USER app
EXPOSE 8080
ENTRYPOINT ["/app/server"]
CMD ["-c", "/app/config.json"]
