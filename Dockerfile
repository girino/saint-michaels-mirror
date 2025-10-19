# Copyright (c) 2025 Girino Vey.
# 
# This software is licensed under Girino's Anarchist License (GAL).
# See LICENSE file for full license text.
# License available at: https://license.girino.org/
#
# Multi-stage Dockerfile for Espelho de São Miguel

# Multi-stage Dockerfile for Espelho de São Miguel
# Build stage
FROM golang:1.24.1-bullseye AS builder
WORKDIR /src

# Accept build arguments for target architecture
ARG GOOS=linux
ARG GOARCH

# Copy modules manifests first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the source
COPY . .

# Build the relay binary
# If GOARCH is not provided, detect it from the build environment
RUN ARCH=${GOARCH:-$(go env GOARCH)} && \
    CGO_ENABLED=0 GOOS=${GOOS} GOARCH=${ARCH} \
    go build -ldflags "-X main.Version=$(grep 'const Version =' cmd/saint-michaels-mirror/version.go | cut -d'"' -f2)" -o /out/saint-michaels-mirror ./cmd/saint-michaels-mirror

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
