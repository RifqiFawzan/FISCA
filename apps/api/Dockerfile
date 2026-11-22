# syntax=docker/dockerfile:1.7

# ---------------------------------------------------------------------------
# Build stage
# ---------------------------------------------------------------------------
FROM golang:1.27-alpine AS builder

WORKDIR /src
ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=0.1.0
RUN go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/api ./cmd/api \
 && go build -trimpath -ldflags "-s -w" -o /out/migrate ./cmd/migrate \
 && go build -trimpath -ldflags "-s -w" -o /out/seed ./cmd/seed

# ---------------------------------------------------------------------------
# Runtime stage
# ---------------------------------------------------------------------------
FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata \
 && addgroup -S -g 10001 fisca \
 && adduser -S -D -H -u 10001 -G fisca fisca

WORKDIR /app
COPY --from=builder /out/ /app/

USER fisca
EXPOSE 8080

# Default command is the API; the db-init service overrides the entrypoint with
# /app/migrate (and /app/seed).
ENTRYPOINT ["/app/api"]
