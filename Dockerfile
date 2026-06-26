# syntax=docker/dockerfile:1

# ---- Stage 1: compile TypeScript -> JavaScript ----
FROM node:22-alpine@sha256:16e22a550f3863206a3f701448c45f7912c6896a62de43add43bb9c86130c3e2 AS web
WORKDIR /web
COPY web_res/package.json web_res/package-lock.json ./
RUN npm install --no-audit --no-fund
COPY web_res/ ./
RUN npm run build

# ---- Stage 2: build the Go server (static, no CGO) ----
FROM golang:1.26-alpine@sha256:3ad57304ad93bbec8548a0437ad9e06a455660655d9af011d58b993f6f615648 AS build
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Bring in the JS compiled by the web stage so it ships inside web_res/.
COPY --from=web /web/dist ./web_res/dist
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server .

# ---- Stage 3: runtime ----
FROM alpine:3.20@sha256:d9e853e87e55526f6b2917df91a2115c36dd7c696a35be12163d44e6e2a4b6bc
RUN apk add --no-cache ca-certificates wget && adduser -D -u 10001 app
WORKDIR /app
COPY --from=build /out/server /app/server
COPY --from=build /src/web_res /app/web_res
# The server reads web_res into an in-memory asset map at startup; it never writes
# to disk, so this only needs to be readable by the run user (the container can
# run with a read-only root filesystem).
RUN chown -R app:app /app
USER app
EXPOSE 8080
ENTRYPOINT ["/app/server"]
CMD ["-c", "/app/config.json"]
