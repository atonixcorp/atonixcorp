FROM golang:1.27.1 AS build

# Put project inside GOPATH
WORKDIR /go/src/atonixcorp/cloud/database

COPY . .

RUN go mod tidy

RUN go build -o /bin/atdb-api ./cmd/atdb-api
RUN go build -o /bin/atdb-controller ./cmd/atdb-controller
RUN go build -o /bin/atdb-health ./cmd/atdb-health

FROM debian:12
LABEL org.opencontainers.image.source="https://github.com/orcadevstack/atonixcorp"

WORKDIR /app
COPY --from=build /bin/atdb-api /usr/local/bin/atdb-api
COPY --from=build /bin/atdb-controller /usr/local/bin/atdb-controller
COPY --from=build /bin/atdb-health /usr/local/bin/atdb-health
COPY --from=build /go/src/atonixcorp/cloud/database/config /app/config
HEALTHCHECK --interval=10s --timeout=3s --start-period=20s --retries=5 CMD ["/usr/local/bin/atdb-health"]

ENTRYPOINT ["/usr/local/bin/atdb-api"]
