# ============================
# Stage 1 — Build all analytics binaries
# ============================
FROM golang:1.27 AS builder

WORKDIR /app

# Copy go.mod and go.sum first
COPY go.mod ./
RUN go mod download

# Copy the entire analytics project
COPY . .

# Build API
RUN CGO_ENABLED=0 go build -o bin/analytics-api ./cmd/api

# Build collectors
RUN go build -o bin/collector-system ./cmd/collector
RUN go build -o bin/collector-nodes ./cmd/collector-nodes
RUN go build -o bin/collector-devices ./cmd/collector-devices
RUN go build -o bin/collector-jobs ./cmd/collector-jobs

# Build ingestor (stub)
RUN go build -o bin/ingestor ./cmd/ingestor

# ============================
# Stage 2 — Final runtime image
# ============================
FROM alpine:3.19
LABEL org.opencontainers.image.source="https://github.com/orcadevstack/atonixcorp"

RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy all binaries from builder
COPY --from=builder /app/bin/* /usr/local/bin/

# Default command (API)
ENTRYPOINT ["analytics-api"]
