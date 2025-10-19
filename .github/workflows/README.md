# GitHub Actions Workflows

This directory contains GitHub Actions workflows for automated CI/CD, testing, and deployment of Espelho de São Miguel.

## Workflows

### 1. CI (`ci.yml`)
- **Trigger**: Push to `main`, Pull Requests to `main`
- **Purpose**: Basic Go build and test validation
- **Actions**: 
  - Checkout code
  - Set up Go 1.24.1
  - Install dependencies
  - Format code with `gofmt`
  - Build application
  - Run tests

### 2. Test (`test.yml`)
- **Trigger**: Push to `main`, Pull Requests to `main`
- **Purpose**: Comprehensive testing including Docker (build only, no push)
- **Actions**:
  - Go build and test
  - Multi-architecture Docker image build test (AMD64, ARM64)
  - Docker container run test for both architectures
  - Health check validation

### 3. Docker (`docker.yml`)
- **Trigger**: Push to `main` branch only
- **Purpose**: Build and push latest Docker image for main branch
- **Actions**:
  - Build multi-architecture images (amd64, arm64)
  - Push `latest` tag to GitHub Container Registry (ghcr.io)
  - Cache layers for faster builds
  - Provides latest image for users between releases

### 4. Release (`release.yml`)
- **Trigger**: Push tags matching `v*` pattern
- **Purpose**: Create GitHub releases with Docker images and binary executables
- **Actions**:
  - Build and push release Docker images (multi-architecture) with version tags
  - Build binary executables for multiple platforms (Linux, macOS, Windows)
  - Generate SHA256 checksums for all binaries
  - Generate changelog from git commits
  - Create GitHub release with changelog and binary downloads
  - Upload individual binaries and complete archives to release assets
  - Tag images with semantic versioning (v1.0.0, v1.0, v1, latest)

### 5. Security (`security.yml`)
- **Trigger**: Push to `main`, Pull Requests to `main`, Weekly schedule
- **Purpose**: Security scanning and vulnerability detection
- **Actions**:
  - File system security scan with Trivy
  - Docker image security scan with Trivy
  - Upload results to GitHub Security tab

## Docker Images

### Registry
Images are published to: `ghcr.io/girino/saint-michaels-mirror`

### Tags
- `latest` - Latest stable release
- `v1.0.0` - Specific version tags
- `v1.0` - Major.minor version tags
- `v1` - Major version tags
- `main` - Latest main branch build
- `pr-123` - Pull request builds

## Binary Executables

### Supported Platforms
- **Linux**: AMD64, ARM64
- **macOS**: AMD64, ARM64 (Apple Silicon)
- **Windows**: AMD64, ARM64

### Release Assets
Each release includes:
- **Complete Archives**: Ready-to-use packages with binaries, static files, templates, and configuration
  - `saint-michaels-mirror-vX.X.X-complete.tar.gz` (Linux/macOS)
  - `saint-michaels-mirror-vX.X.X-complete.zip` (Windows)
- **Individual Binaries**: Platform-specific executable files
- **SHA256 Checksums**: For verification of all files
- **README**: Comprehensive setup instructions included in archives

### Usage

#### Docker (Latest Release)
```bash
docker run -d \
  --name saint-michaels-mirror \
  -p 3337:3337 \
  -e RELAY_NAME="Your Relay Name" \
  ghcr.io/girino/saint-michaels-mirror:latest
```

#### Docker (Specific Version)
```bash
docker run -d \
  --name saint-michaels-mirror \
  -p 3337:3337 \
  -e RELAY_NAME="Your Relay Name" \
  ghcr.io/girino/saint-michaels-mirror:v1.0.0
```

#### Binary (Linux)
```bash
# Download the appropriate binary for your platform
wget https://github.com/girino/saint-michaels-mirror/releases/download/v1.0.0/saint-michaels-mirror-linux-amd64
chmod +x saint-michaels-mirror-linux-amd64

# Run with environment variables
RELAY_NAME="Your Relay Name" ./saint-michaels-mirror-linux-amd64 --addr=:3337
```

#### Binary (macOS)
```bash
# Download for Apple Silicon
wget https://github.com/girino/saint-michaels-mirror/releases/download/v1.0.0/saint-michaels-mirror-darwin-arm64
chmod +x saint-michaels-mirror-darwin-arm64
RELAY_NAME="Your Relay Name" ./saint-michaels-mirror-darwin-arm64 --addr=:3337
```

#### Binary (Windows)
```cmd
# Download the Windows binary
curl -LO https://github.com/girino/saint-michaels-mirror/releases/download/v1.0.0/saint-michaels-mirror-windows-amd64.exe

# Run with environment variables
set RELAY_NAME=Your Relay Name
saint-michaels-mirror-windows-amd64.exe --addr=:3337
```

## Environment Variables

The following environment variables can be used to configure the relay:

- `RELAY_NAME` - Display name of the relay
- `RELAY_DESCRIPTION` - Description of the relay
- `RELAY_CONTACT` - Contact information (npub, email, etc.)
- `RELAY_SERVICE_URL` - Public URL of the relay
- `RELAY_ICON` - Path to relay icon
- `RELAY_BANNER` - Path to relay banner
- `ADDR` - Address to listen on (default: :3337)
- `PUBLISH_REMOTES` - Comma-separated list of publish relays
- `QUERY_REMOTES` - Comma-separated list of query relays
- `VERBOSE` - Enable verbose logging (1 to enable)

## Secrets

The following secrets are required for the workflows:

- `GITHUB_TOKEN` - Automatically provided by GitHub Actions
- `DOCKER_USERNAME` - Docker Hub username (if using Docker Hub)
- `DOCKER_PASSWORD` - Docker Hub password (if using Docker Hub)

## Permissions

The workflows require the following permissions:

- `contents: read` - Read repository contents
- `packages: write` - Write to GitHub Container Registry
- `security-events: write` - Write security scan results

These permissions are configured in the workflow files and will be automatically granted when the workflows are enabled.
