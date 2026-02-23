FROM golang:1.26-bookworm AS builder

WORKDIR /go/src/app
COPY . .
RUN make build

FROM debian:bullseye-slim
COPY --from=builder /go/src/app/dist/penny /
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ENTRYPOINT ["/penny"]
