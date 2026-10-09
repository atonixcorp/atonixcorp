FROM golang:1.27-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /server ./cmd/server

FROM alpine:3.21
LABEL org.opencontainers.image.source="https://github.com/orcadevstack/atonixcorp"

RUN addgroup -S app && adduser -S app -G app
USER app
WORKDIR /app
COPY --from=builder /server /app/server
EXPOSE 8080
CMD ["/app/server"]
