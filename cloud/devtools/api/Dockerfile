FROM golang:1.27.1 AS builder

WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/devtools-api ./cmd/devtools-api

FROM alpine:3.22
RUN apk add --no-cache ca-certificates && adduser -D -H app
COPY --from=builder /out/devtools-api /usr/local/bin/devtools-api
USER app
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/devtools-api"]
