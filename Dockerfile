# ABOUTME: Multi-stage Dockerfile for the reachy-mqtt bridge daemon.
# ABOUTME: Builds a minimal scratch-based image with just the Go binary.

FROM golang:1.23-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o reachy-mqtt .

FROM debian:bookworm-slim

WORKDIR /app
COPY --from=builder /app/reachy-mqtt .
COPY .env.example .env

CMD ["./reachy-mqtt"]
