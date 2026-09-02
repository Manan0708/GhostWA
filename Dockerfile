# Multi-stage Dockerfile for GhostWA

# Stage 1: Build static CGO-free binary
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install git and certs
RUN apk add --no-cache git ca-certificates

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy project source code
COPY . .

# Build static Linux binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/ghostwa ./cmd/ghostwa

# Stage 2: Minimal runtime container
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /root

# Copy ghostwa executable
COPY --from=builder /app/bin/ghostwa /usr/local/bin/ghostwa

# Expose IPC port
EXPOSE 42069

# Persistent volume for messages.db & session.db
VOLUME ["/root/.local/share/wacli"]

# Default entrypoint
CMD ["ghostwa", "daemon-run"]
