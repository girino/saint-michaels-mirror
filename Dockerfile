# Multi-stage Dockerfile for khatru-relay
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
    go build -ldflags "-X main.Version=$(cat cmd/khatru-relay/version.go | grep -oP '"\K[^"]+(?=\")')" -o /out/khatru-relay ./cmd/khatru-relay

# Final minimal image
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN useradd -m -u 1000 relayuser
WORKDIR /home/relayuser
COPY --from=builder /out/khatru-relay ./khatru-relay
RUN chown relayuser:relayuser ./khatru-relay
USER relayuser

EXPOSE 8080

ENTRYPOINT ["./khatru-relay"]
