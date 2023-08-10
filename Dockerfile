# syntax=docker/dockerfile:1
FROM golang:1.21-alpine AS builder

ENV CGO_ENABLED=0
WORKDIR /build

RUN apk add ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go install ./cmd/elphi-calendar


FROM scratch

COPY --from=builder /etc/ssl/certs /etc/ssl/certs
COPY --from=builder /go/bin/elphi-calendar /elphi-calendar
EXPOSE 8080
ENTRYPOINT ["/elphi-calendar"]
