# Espelho de São Miguel

[![License: GAL](https://img.shields.io/badge/License-GAL-blue.svg)](https://license.girino.org/)
[![Docker Image](https://img.shields.io/badge/Docker-ghcr.io%2Fgirino%2Fsaint--michaels--mirror-blue)](https://github.com/girino/saint-michaels-mirror/pkgs/container/saint-michaels-mirror)
[![Go Version](https://img.shields.io/badge/Go-1.24.1-blue.svg)](https://golang.org/)

> The Espelho de São Miguel is the sacred mirror that stands between worlds, where every message is received, reflected, and transmitted without distortion under the Archangel’s vigilant gaze. It unites the power of Exu, opener of paths, with the harmony of Ibeji, the divine twins, ensuring that all light crossing its surface returns as truth.

**Espelho de São Miguel** is a Nostr relay aggregator built on the [khatru](https://github.com/fiatjaf/khatru) framework. It acts as a mirror between worlds, forwarding events between multiple Nostr relays while providing a unified interface for Nostr applications.

 

## 🌟 Features

- **🔄 Event Aggregation**: Forwards published events to multiple remote relays
- **🔍 Query Unification**: Queries multiple relays and merges results for clients
- **🔐 Authentication Passthrough**: Automatically authenticates with upstream relays using configured relay key
- **📡 Event Mirroring**: Continuously mirrors events from query relays to provide comprehensive event coverage
- **⚠️ Structured Error Handling**: Passes through machine-readable error prefixes from upstream relays
- **📊 Real-time Monitoring**: Live statistics and health monitoring dashboard
- **🐳 Docker Ready**: Complete Docker Compose setup for easy deployment
- **🌐 Web Interface**: Modern, responsive web interface with NIP-11 compliance
- **⚡ High Performance**: Concurrent processing with atomic metrics collection
- **🔒 Security Focused**: Comprehensive health checks and failure tracking
- **📱 Multi-Platform**: Binaries for Linux, macOS, and Windows (AMD64/ARM64)

## 🚀 Quick Start

### Option 1: Docker Compose (Recommended)

```bash
# Clone the repository
git clone https://github.com/girino/saint-michaels-mirror.git
cd saint-michaels-mirror

# Configure your relay
cp example.env .env
# Edit .env with your settings

# Deploy with Docker Compose
docker compose -f docker-compose.prod.yml up -d
```

### Option 2: Docker Run

```bash
# Run with latest image
docker run -d \
  --name saint-michaels-mirror \
  -p 3337:3337 \
  -e RELAY_NAME="Your Relay Name" \
  -e PUBLISH_REMOTES="wss://relay1.example.com,wss://relay2.example.com" \
  -e QUERY_REMOTES="wss://relay1.example.com,wss://relay2.example.com" \
  ghcr.io/girino/saint-michaels-mirror:latest
```

### Option 3: Standalone Binary

```bash
# Download the latest release
wget https://github.com/girino/saint-michaels-mirror/releases/latest/download/saint-michaels-mirror-linux-amd64
chmod +x saint-michaels-mirror-linux-amd64

# Configure and run
export RELAY_NAME="Your Relay Name"
export PUBLISH_REMOTES="wss://relay1.example.com,wss://relay2.example.com"
export QUERY_REMOTES="wss://relay1.example.com,wss://relay2.example.com"
./saint-michaels-mirror-linux-amd64 --addr=:3337
```

## ⚙️ Configuration

Create a `.env` file with your configuration:

```bash
# Required: Remote relays to use
QUERY_REMOTES=wss://relay1.example.com,wss://relay2.example.com
PUBLISH_REMOTES=wss://relay1.example.com,wss://relay2.example.com

# Required: Relay identity
RELAY_NAME="Espelho de São Miguel"
RELAY_DESCRIPTION="The sacred mirror that stands between worlds..."

# Optional: Contact and branding
RELAY_CONTACT=npub1your-contact-npub-here
RELAY_SERVICE_URL=https://your-relay.com
RELAY_ICON=static/icon.png
RELAY_BANNER=static/banner.png

# Optional: Server settings
ADDR=:3337
VERBOSE=0

# Optional: Authentication (for upstream relays)
RELAY_SECKEY=nsec1your-relay-secret-key-here

# Optional: Docker settings
PROD_IMAGE=ghcr.io/girino/saint-michaels-mirror:latest
COMPOSE_RELAY_PORT=3337
```

### Configuration Variables

| Variable | Required | Description | Default |
|----------|----------|-------------|---------|
| `QUERY_REMOTES` | ✅ | Comma-separated list of relays to query | - |
| `PUBLISH_REMOTES` | ✅ | Comma-separated list of relays to forward to | - |
| `RELAY_NAME` | ✅ | Display name of your relay | "Espelho de São Miguel" |
| `RELAY_DESCRIPTION` | ✅ | Description of your relay | Mythic description |
| `RELAY_CONTACT` | ❌ | Contact npub or email | - |
| `RELAY_SERVICE_URL` | ❌ | Public URL of your relay | - |
| `RELAY_ICON` | ❌ | Path to relay icon | - |
| `RELAY_BANNER` | ❌ | Path to relay banner | - |
| `RELAY_SECKEY` | ❌ | Relay secret key (hex or nsec) for authentication | - |
| `ADDR` | ❌ | Address to listen on | `:3337` |
| `VERBOSE` | ❌ | Enable verbose logging (1/0) | `0` |
| `PROD_IMAGE` | ❌ | Docker image for compose | `latest` |

## 🔐 Authentication & Mirroring Features

### Authentication Passthrough
The relay automatically authenticates with upstream relays when required using the configured `RELAY_SECKEY`. This enables seamless operation with relays that require authentication for publishing events.

**Supported Key Formats:**
- **Raw Hex**: `a1b2c3d4e5f6...` (64-character hex string)
- **nsec Bech32**: `nsec1abc123...` (bech32 encoded secret key)

The relay automatically detects and decodes nsec keys to hex format for authentication, ensuring compatibility with both formats.

### Event Mirroring
The relay continuously mirrors events from query relays using a "since now" filter, providing comprehensive event coverage. Mirrored events are injected into the local relay via `khatru.BroadcastEvent()` and counted in statistics.

**Mirroring Benefits:**
- **Complete Coverage**: Ensures all events from query relays are available locally
- **Real-time Updates**: Events are mirrored immediately as they arrive
- **Deduplication**: Automatic deduplication prevents duplicate events
- **Statistics Tracking**: Mirroring activity is tracked in the stats endpoint

### Structured Error Handling
When all publish attempts fail, the relay returns machine-readable error prefixes from upstream relays (NIP-01 standard), including: `duplicate`, `pow`, `blocked`, `rate-limited`, `invalid`, `restricted`, `mute`, `error`, and `auth-required`.

**Error Format**: `prefix: message (relay-url)` - includes the source relay URL for context.

## 🌐 Web Interface

Once running, visit your relay in a web browser:

- **Main Page** (`/`): Relay information and NIP-11 metadata
- **Statistics** (`/stats`): Real-time performance metrics and counters
- **Health** (`/health`): Health status and failure tracking
- **API** (`/api/v1/stats`, `/api/v1/health`): JSON endpoints for monitoring

### Features

- **Real-time Updates**: Statistics and health pages auto-refresh every 10 seconds
- **Responsive Design**: Works on desktop and mobile devices
- **NIP-11 Compliance**: Standard relay information endpoint
- **Health Monitoring**: Visual indicators for relay health status
- **Performance Metrics**: Detailed timing and counter statistics

## 📊 Monitoring and Health

### Health States

The relay monitors its own health and reports status:

- **🟢 GREEN**: No failures, all operations successful
- **🟡 YELLOW**: Some failures detected, but below threshold
- **🔴 RED**: Critical failures, relay marked as unhealthy

### Metrics Available

- **Operation Counters**: Attempts, successes, failures for all operations
- **Timing Statistics**: Average, minimum, maximum operation times
- **System Resources**: Memory usage, goroutine counts, GC statistics
- **Remote Connectivity**: Status of connected remote relays
- **Failure Tracking**: Consecutive failure counts with automatic recovery
- **Mirroring Statistics**: Mirrored events, mirror attempts, successes, and failures
- **Relay Health**: Live/dead relay counts and mirroring health state

## 🏗️ Architecture

### Conceptual Mapping

The relay implements a spiritual metaphor through technical architecture:

- **Mirror**: Published events are accepted and forwarded (mirrored) to remote relays
- **Archangel**: Reliable, accountable message transmission and metadata signing
- **Ibeji Twins**: Mirroring and duplication through copying and distribution
- **Exu**: Opening of paths and active communication with other relays

### Technical Implementation

- **Event Forwarding**: Concurrent forwarding to multiple remote relays
- **Query Aggregation**: Merging responses from multiple query sources
- **Health Monitoring**: Failure tracking with configurable thresholds
- **Metrics Collection**: Atomic counters for all operations
- **Smart Routing**: Internal vs. external query differentiation

## 🛠️ Development

### Building from Source

```bash
# Clone the repository
git clone https://github.com/girino/saint-michaels-mirror.git
cd saint-michaels-mirror

# Build the binary
go build -o bin/saint-michaels-mirror ./cmd/saint-michaels-mirror

# Run with development settings
./bin/saint-michaels-mirror --addr=:3337
```

### Testing

```bash
# Run tests
go test -v ./...

# Run with verbose logging
VERBOSE=1 ./bin/saint-michaels-mirror
```

### Docker Development

```bash
# Build Docker image
docker build -t saint-michaels-mirror:dev .

# Run with development settings
docker run -d \
  --name saint-michaels-mirror-dev \
  -p 3337:3337 \
  -e VERBOSE=1 \
  saint-michaels-mirror:dev
```

## 📦 Releases

### Download Options

- **Complete Archives**: Ready-to-use packages with all assets
  - `saint-michaels-mirror-vX.X.X-complete.tar.gz` (Linux/macOS)
  - `saint-michaels-mirror-vX.X.X-complete.zip` (Windows)
- **Individual Binaries**: Platform-specific executable files
- **Docker Images**: Multi-architecture images on GitHub Container Registry

### Supported Platforms

- **Linux**: AMD64, ARM64
- **macOS**: AMD64, ARM64 (Apple Silicon)
- **Windows**: AMD64, ARM64
- **Docker**: Multi-architecture support

## 🔒 Security and Privacy

- **Event Integrity**: Events are forwarded without modification
- **Internal Query Filtering**: Prevents leakage of internal bookkeeping queries
- **Secure Key Management**: Proper handling of relay signing keys
- **Privacy Respect**: No modification or storage of client data
- **Security Scanning**: Automated vulnerability detection

## 🤝 Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

### Development Guidelines

- Follow Go project conventions
- Add tests for new functionality
- Update documentation as needed
- Ensure all tests pass
- Follow the existing code style

## 📄 License

This project is licensed under **Girino's Anarchist License (GAL)**.

- **License Text**: [LICENSE](LICENSE)
- **License URL**: https://license.girino.org/

> Note: The GAL is a nonstandard, humorous license with unusual conditions; treat its terms accordingly.

## 📞 Support

- **Documentation**: [GitHub Repository](https://github.com/girino/saint-michaels-mirror)
- **Issues**: [GitHub Issues](https://github.com/girino/saint-michaels-mirror/issues)
- **Releases**: [GitHub Releases](https://github.com/girino/saint-michaels-mirror/releases)

## 🙏 Acknowledgments

- Built on the excellent [khatru](https://github.com/fiatjaf/khatru) framework
- Inspired by the Nostr protocol and community
- Thanks to all contributors and testers

---

**Espelho de São Miguel** - The mirror that returns light as truth.

*Copyright (c) 2025 Girino Vey. Licensed under Girino's Anarchist License (GAL).*