# ============================
# Stage 1 — Build ATCLI
# ============================
FROM golang:1.27.1 AS builder

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum first (for caching)
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the CLI binary
RUN go build -o atcli .

# ============================
# Stage 2 — Final Image
# ============================
FROM alpine:3.19
LABEL org.opencontainers.image.source="https://github.com/orcadevstack/atonixcorp"

# Install CA certificates (needed for HTTPS)
RUN apk add --no-cache ca-certificates

# Create app directory
WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /app/atcli /usr/local/bin/atcli

# Expose port if you run server mode
EXPOSE 8000

# Default command
ENTRYPOINT ["atcli"]
