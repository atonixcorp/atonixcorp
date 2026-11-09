FROM golang:1.27.1 AS builder

WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/devtools-portal ./cmd/devtools-portal

FROM alpine:3.22
RUN apk add --no-cache ca-certificates && adduser -D -H app
COPY --from=builder /out/devtools-portal /usr/local/bin/devtools-portal
USER app
EXPOSE 8081
ENTRYPOINT ["/usr/local/bin/devtools-portal"]
