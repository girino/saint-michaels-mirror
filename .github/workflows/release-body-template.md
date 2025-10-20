## Espelho de São Miguel {{VERSION}}

A Nostr relay aggregator built on khatru that forwards events between multiple relays.

### Docker Images
- `{{REGISTRY}}/{{IMAGE_NAME}}:{{VERSION}}`
- `{{REGISTRY}}/{{IMAGE_NAME}}:latest`

### Binary Downloads

#### Complete Archives (Recommended)
- **Linux/macOS**: `saint-michaels-mirror-{{VERSION}}-complete.tar.gz`
- **Windows**: `saint-michaels-mirror-{{VERSION}}-complete.zip`

These archives include binaries, static files, templates, configuration examples, Docker Compose setup, and comprehensive release notes.

#### Individual Binaries
Download individual binaries for your platform:
- **Linux AMD64**: `saint-michaels-mirror-linux-amd64`
- **Linux ARM64**: `saint-michaels-mirror-linux-arm64`
- **macOS AMD64**: `saint-michaels-mirror-darwin-amd64`
- **macOS ARM64 (Apple Silicon)**: `saint-michaels-mirror-darwin-arm64`
- **Windows AMD64**: `saint-michaels-mirror-windows-amd64.exe`
- **Windows ARM64**: `saint-michaels-mirror-windows-arm64.exe`

### Changes
{{CHANGELOG}}

## Quick Start

### Option 1: Docker (Recommended)

#### Basic Usage
```bash
docker run -d \
  --name saint-michaels-mirror \
  -p 3337:3337 \
  -e RELAY_NAME="Your Relay Name" \
  {{REGISTRY}}/{{IMAGE_NAME}}:{{VERSION}}
```

#### With Custom Configuration
```bash
# Create a .env file with your configuration
cat > .env << EOF
RELAY_NAME=My Nostr Relay
RELAY_DESCRIPTION=A Nostr relay aggregator
RELAY_CONTACT=your-npub-here
RELAY_SERVICE_URL=https://your-relay.com
PUBLISH_REMOTES=wss://relay1.example.com,wss://relay2.example.com
QUERY_REMOTES=wss://relay1.example.com,wss://relay2.example.com
EOF

# Run with custom configuration
docker run -d \
  --name saint-michaels-mirror \
  -p 3337:3337 \
  --env-file .env \
  {{REGISTRY}}/{{IMAGE_NAME}}:{{VERSION}}
```

#### Using Docker Compose
```bash
# Clone the repository and use the provided docker-compose.prod.yml
git clone https://github.com/girino/saint-michaels-mirror.git
cd saint-michaels-mirror
cp example.env .env
# Edit .env with your configuration
docker compose -f docker-compose.prod.yml up -d
```

#### Using Release Archives (Docker Compose)
```bash
# Download and extract the complete archive
wget https://github.com/girino/saint-michaels-mirror/releases/download/{{VERSION}}/saint-michaels-mirror-{{VERSION}}-complete.tar.gz
tar -xzf saint-michaels-mirror-{{VERSION}}-complete.tar.gz
cd saint-michaels-mirror-{{VERSION}}

# Configure and deploy
cp .env.example .env
# Edit .env with your configuration

# (Optional) Use specific release version instead of latest
echo "PROD_IMAGE={{REGISTRY}}/{{IMAGE_NAME}}:{{VERSION}}" >> .env

docker compose -f docker-compose.prod.yml up -d
```

### Option 2: Standalone Binary

#### Linux/macOS (Complete Archive - Recommended)
```bash
# Download the complete archive with all assets
wget https://github.com/girino/saint-michaels-mirror/releases/download/{{VERSION}}/saint-michaels-mirror-{{VERSION}}-complete.tar.gz
tar -xzf saint-michaels-mirror-{{VERSION}}-complete.tar.gz
cd saint-michaels-mirror-{{VERSION}}

# Copy and configure environment
cp .env.example .env
# Edit .env with your settings

# Run the appropriate binary
chmod +x saint-michaels-mirror-linux-amd64  # or darwin-arm64 for macOS
./saint-michaels-mirror-linux-amd64 --addr=:3337
```

#### Windows (Complete Archive - Recommended)
```cmd
# Download the complete archive with all assets
curl -LO https://github.com/girino/saint-michaels-mirror/releases/download/{{VERSION}}/saint-michaels-mirror-{{VERSION}}-complete.zip
# Extract using your preferred zip tool

# Copy and configure environment
copy .env.example .env
# Edit .env with your settings

# Run the Windows binary
saint-michaels-mirror-windows-amd64.exe --addr=:3337
```

#### Individual Binary Downloads (Alternative)
```bash
# Linux example - download individual binary
wget https://github.com/girino/saint-michaels-mirror/releases/download/{{VERSION}}/saint-michaels-mirror-linux-amd64
chmod +x saint-michaels-mirror-linux-amd64

# Set environment variables
export RELAY_NAME="Your Relay Name"
export PUBLISH_REMOTES="wss://relay1.example.com,wss://relay2.example.com"
export QUERY_REMOTES="wss://relay1.example.com,wss://relay2.example.com"

# Run the relay
./saint-michaels-mirror-linux-amd64 --addr=:3337
```

## Configuration

### Required Environment Variables
- `RELAY_NAME` - Display name of your relay
- `PUBLISH_REMOTES` - Comma-separated list of relays to forward published events to
- `QUERY_REMOTES` - Comma-separated list of relays to query events from

### Optional Environment Variables
- `RELAY_DESCRIPTION` - Description of your relay
- `RELAY_CONTACT` - Contact information (npub, email, etc.)
- `RELAY_SERVICE_URL` - Public URL of your relay
- `RELAY_ICON` - Path to relay icon (for Docker)
- `RELAY_BANNER` - Path to relay banner (for Docker)
- `ADDR` - Address to listen on (default: :3337)
- `VERBOSE` - Enable verbose logging (1 to enable)

## Verification

After starting the relay, you can verify it's working:

```bash
# Check NIP-11 relay information
curl http://localhost:3337/

# Check health status
curl http://localhost:3337/api/v1/health

# Check statistics
curl http://localhost:3337/api/v1/stats
```

## Support

- **Documentation**: [GitHub Repository](https://github.com/girino/saint-michaels-mirror)
- **Issues**: [GitHub Issues](https://github.com/girino/saint-michaels-mirror/issues)
- **License**: Girino's Anarchist License (GAL) - [License Details](https://license.girino.org/)
