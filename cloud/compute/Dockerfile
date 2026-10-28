FROM golang:1.27.1-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /compute-api ./cmd/atcompute-api

FROM alpine:3.21

RUN addgroup -S app && adduser -S app -G app \
    && mkdir -p /var/lib/atonixcorp-compute \
    && chown -R app:app /var/lib/atonixcorp-compute
USER app
WORKDIR /app
COPY --from=builder /compute-api /app/compute-api

ENV COMPUTE_API_ADDR=:8580
ENV COMPUTE_STORE_PATH=/var/lib/atonixcorp-compute/instances.json
EXPOSE 8580
VOLUME ["/var/lib/atonixcorp-compute"]
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:8580/health || exit 1
CMD ["/app/compute-api"]
