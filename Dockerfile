# syntax=docker/dockerfile:1.7
#
# Production image — single binary that serves both the API and the SPA.
# External Postgres + OIDC provider are required at runtime via env vars.

# ---------- Stage 1: build the Nuxt SPA ----------
FROM node:20-alpine AS frontend-build
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY frontend/ ./
# `nuxt generate` outputs the static SPA to .output/public/
RUN npm run generate

# ---------- Stage 2: build the Go binary with the SPA embedded ----------
FROM golang:1.25-alpine AS backend-build
WORKDIR /src
RUN apk add --no-cache git
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
# Drop the generated SPA into the embed location before `go build`.
COPY --from=frontend-build /app/.output/public/ ./internal/frontend/dist/
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /out/levitate \
    ./cmd/levitate

# ---------- Stage 3: runtime ----------
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S levitate && \
    adduser -S -u 1000 -G levitate levitate
COPY --from=backend-build /out/levitate /usr/local/bin/levitate
USER levitate
ENV LEVITATE_HTTP_ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/levitate"]
