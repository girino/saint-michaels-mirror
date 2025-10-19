# Multi-stage Dockerfile for saint-michaels-mirror
# Build stage
FROM golang:1.24.1-bullseye AS builder
WORKDIR /src

# Copy modules manifests first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the source
COPY . .

# Build the relay binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags "-X main.Version=$(cat cmd/saint-michaels-mirror/version.go | grep -oP '"\K[^"]+(?=\")')" -o /out/saint-michaels-mirror ./cmd/saint-michaels-mirror

# Final minimal image
FROM debian:bookworm-slim
# Install ca-certificates and curl for healthchecks and basic networking
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates curl \
 && rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN useradd -m -u 1000 relayuser
WORKDIR /home/relayuser

# Copy the binary
COPY --from=builder /out/saint-michaels-mirror ./saint-michaels-mirror

# Copy static files and templates
COPY --from=builder /src/cmd/saint-michaels-mirror/static ./cmd/saint-michaels-mirror/static
COPY --from=builder /src/cmd/saint-michaels-mirror/templates ./cmd/saint-michaels-mirror/templates

# Set proper ownership
RUN chown -R relayuser:relayuser ./saint-michaels-mirror ./cmd
USER relayuser

ENTRYPOINT ["./saint-michaels-mirror"]
