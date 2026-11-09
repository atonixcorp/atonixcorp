FROM golang:1.27.1 AS builder

WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/devtools-cli ./cmd/devtools-cli

FROM alpine:3.22
RUN apk add --no-cache ca-certificates && adduser -D -H app
COPY --from=builder /out/devtools-cli /usr/local/bin/devtools-cli
USER app
ENTRYPOINT ["/usr/local/bin/devtools-cli"]