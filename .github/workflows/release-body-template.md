## Espelho de São Miguel {{VERSION}}

A Nostr relay aggregator built on khatru that forwards events between multiple relays.

### Docker Images
- `{{REGISTRY}}/{{IMAGE_NAME}}:{{VERSION}}`
- `{{REGISTRY}}/{{IMAGE_NAME}}:latest`

### Binary Downloads

#### Complete Archives (Recommended)
- **Linux/macOS**: `saint-michaels-mirror-{{VERSION}}-complete.tar.gz`
- **Windows**: `saint-michaels-mirror-{{VERSION}}-complete.zip`

These archives include:
- **Binaries**: Platform-specific executable files for Linux, macOS, and Windows (AMD64/ARM64)
- **Web Interface**: Static files (CSS, JavaScript, images) and HTML templates
- **Configuration**: `example.env` template and `docker-compose.prod.yml`
- **Deployment**: `DEPLOYMENT.md` guide and `nginx.conf.example`
- **Documentation**: `CHANGELOG.md` with detailed changes and `README.md`
- **Verification**: `checksums.txt` for file integrity verification

### Changes
See the curated CHANGELOG.md in this release for human-friendly summaries of changes across versions.

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

#### Using Docker Compose (from Complete Archive)
```bash
# Download and extract the complete archive
wget https://github.com/girino/saint-michaels-mirror/releases/download/{{VERSION}}/saint-michaels-mirror-{{VERSION}}-complete.tar.gz
tar -xzf saint-michaels-mirror-{{VERSION}}-complete.tar.gz
cd saint-michaels-mirror-{{VERSION}}

# Copy and configure environment
cp .env.example .env
# Edit .env with your configuration

# (Optional) Use specific release version instead of latest
echo "PROD_IMAGE={{REGISTRY}}/{{IMAGE_NAME}}:{{VERSION}}" >> .env

# Deploy with Docker Compose
docker compose -f docker-compose.prod.yml up -d
```


### Option 2: Standalone Binary

#### Linux/macOS (Complete Archive - Recommended)
```bash
# Download the complete archive with all assets
wget https://github.com/girino/saint-michaels-mirror/releases/download/{{VERSION}}/saint-michaels-mirror-{{VERSION}}-complete.tar.gz
tar -xzf saint-michaels-mirror-{{VERSION}}-complete.tar.gz
cd saint-michaels-mirror-{{VERSION}}

# Set environment variables
export RELAY_NAME="Your Relay Name"
export RELAY_DESCRIPTION="A Nostr relay aggregator"
export PUBLISH_REMOTES="wss://relay1.example.com,wss://relay2.example.com"
export QUERY_REMOTES="wss://relay1.example.com,wss://relay2.example.com"

# Run the appropriate binary
chmod +x saint-michaels-mirror-linux-amd64  # or darwin-arm64 for macOS
./saint-michaels-mirror-linux-amd64 --addr=:3337
```

#### Windows (Complete Archive - Recommended)
```cmd
# Download the complete archive with all assets
curl -LO https://github.com/girino/saint-michaels-mirror/releases/download/{{VERSION}}/saint-michaels-mirror-{{VERSION}}-complete.zip
# Extract using your preferred zip tool

# Set environment variables
set RELAY_NAME=Your Relay Name
set RELAY_DESCRIPTION=A Nostr relay aggregator
set PUBLISH_REMOTES=wss://relay1.example.com,wss://relay2.example.com
set QUERY_REMOTES=wss://relay1.example.com,wss://relay2.example.com

# Run the Windows binary
saint-michaels-mirror-windows-amd64.exe --addr=:3337
```


## Additional Files

The complete archives also include:

- **`CHANGELOG.md`** - Curated changelog with human-friendly summaries across versions
- **`DEPLOYMENT.md`** - Comprehensive deployment guide for production setups
- **`nginx.conf.example`** - Example nginx configuration for reverse proxy setup
- **`README.md`** - Quick start guide and usage instructions
- **`checksums.txt`** - SHA256 checksums for verifying file integrity

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
curl -H "Accept: application/nostr+json" http://localhost:3337/

# Check health status (API endpoint)
curl http://localhost:3337/api/v1/health

# Check statistics (API endpoint)
curl http://localhost:3337/api/v1/stats
```

**WebSocket URL for Nostr clients:**
- `ws://localhost:3337` (or `wss://your-domain.com` in production)

You can also visit these URLs in your browser:
- **Main page**: http://localhost:3337/
- **Health page**: http://localhost:3337/health
- **Stats page**: http://localhost:3337/stats

## Support

- **Documentation**: [GitHub Repository](https://github.com/girino/saint-michaels-mirror)
- **Issues**: [GitHub Issues](https://github.com/girino/saint-michaels-mirror/issues)
- **License**: Girino's Anarchist License (GAL) - [License Details](https://license.girino.org/)
