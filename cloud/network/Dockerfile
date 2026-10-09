
FROM golang:1.27.1 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o atnetwork ./main.go


FROM debian:13-slim
LABEL org.opencontainers.image.source="https://github.com/orcastack/atonixcorp"

RUN apt-get update && apt-get install -y \
    ca-certificates \
    iproute2 \
    iptables \
    dnsmasq \
    frr \
    openvswitch-common \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/atnetwork /usr/local/bin/atnetwork
COPY config /etc/atnetwork/config

EXPOSE 5555

ENTRYPOINT ["/usr/local/bin/atnetwork"]
