# ───────────────────────────────────────────────
# Stage 1 — Build the Go binaries
# ───────────────────────────────────────────────
FROM golang:1.27.1-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /build

# Copy go.mod first for caching
COPY go.mod ./
RUN go mod download

# Copy source code
COPY . .

# Build static binaries for Linux
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o api ./cmd/api

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o scheduler ./cmd/scheduler

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o node-agent ./cmd/node-agent

# ───────────────────────────────────────────────
# Stage 2 — Runtime (Distroless)
# ───────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /

# Copy binaries from builder
COPY --from=builder /build/api /api
COPY --from=builder /build/scheduler /scheduler
COPY --from=builder /build/node-agent /node-agent

# Expose ports for API, Scheduler, Node Agent
EXPOSE 8080
EXPOSE 7070
EXPOSE 9090

# Default entrypoint (API service)
ENTRYPOINT ["/api"]
