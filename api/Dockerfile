FROM golang:1.27-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /server ./cmd/server

FROM alpine:3.21
LABEL org.opencontainers.image.source="https://github.com/atonixcorp/atonixcorp"

RUN apk add --no-cache libvirt-client qemu-img
RUN addgroup -S app && adduser -S app -G app
RUN mkdir -p /var/lib/atonixcorp-storage && chown app:app /var/lib/atonixcorp-storage
USER app
WORKDIR /app
COPY --from=builder /server /app/server
EXPOSE 8080
CMD ["/app/server"]
