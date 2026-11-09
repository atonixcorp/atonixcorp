FROM golang:1.27.1 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /app/hpc ./cmd/hpc

FROM debian:13-slim

RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
    ca-certificates wget munge slurm-client && \
    rm -rf /var/lib/apt/lists/* && \
    useradd --system --create-home --shell /usr/sbin/nologin hpcuser

WORKDIR /opt/hpc
COPY --from=builder /app/hpc /opt/hpc/hpc
COPY config/hpc.yaml /opt/hpc/config/hpc.yaml
RUN chown -R hpcuser:hpcuser /opt/hpc

USER hpcuser
EXPOSE 8080 9090
ENTRYPOINT ["/opt/hpc/hpc"]
