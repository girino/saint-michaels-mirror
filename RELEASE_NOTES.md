# Espelho de São Miguel - Release Notes

## Version 1.0.0-rc2

**Release Date**: January 2025  
**License**: Girino's Anarchist License (GAL)  
**Repository**: https://github.com/girino/saint-michaels-mirror

---

## 🎉 What is Espelho de São Miguel?

Espelho de São Miguel is a Nostr relay aggregator built on the khatru framework. It acts as a mirror between worlds, forwarding events between multiple Nostr relays while providing a unified interface for Nostr applications.

The name comes from a spiritual metaphor: the sacred mirror that stands between worlds, where every message is received, reflected, and transmitted without distortion under the Archangel's vigilant gaze.

---

## 🚀 Major Features

### **Relay Aggregation**
- **Event Forwarding**: Automatically forwards published events to configured remote relays
- **Query Aggregation**: Queries multiple remote relays and merges results for clients
- **Smart Routing**: Intelligent handling of internal vs. external queries
- **NIP-45 Support**: Automatic detection and use of count endpoints when available

### **Modern Web Interface**
- **NIP-11 Compliance**: Standard relay information endpoint
- **Real-time Statistics**: Live monitoring dashboard with detailed metrics
- **Health Monitoring**: Visual health status indicators with failure tracking
- **Responsive Design**: Mobile-friendly interface with modern styling
- **Template System**: Clean, maintainable HTML structure with inheritance

### **Production-Ready Deployment**
- **Docker Support**: Complete Docker Compose setup for easy deployment
- **Multi-Platform Binaries**: Native executables for Linux, macOS, and Windows
- **Multi-Architecture**: Support for both AMD64 and ARM64 systems
- **Health Checks**: Comprehensive health monitoring and failure detection
- **Auto-Recovery**: Automatic restart capabilities with failure tracking

### **Developer Experience**
- **Comprehensive API**: RESTful endpoints for health checks and statistics
- **Detailed Logging**: Configurable verbose logging for debugging
- **Metrics Collection**: Atomic counters for all operations
- **Configuration Management**: Environment-based configuration system

---

## 🔧 Technical Improvements

### **Health Monitoring System**
- **Backend Health Checks**: Monitors relay connectivity and performance
- **Failure Tracking**: Tracks consecutive failures for publish and query operations
- **Health States**: GREEN (healthy), YELLOW (degraded), RED (critical) indicators
- **Automatic Recovery**: Resets failure counts on successful operations
- **Threshold-Based Alerts**: Marks relay as unhealthy after 10 consecutive failures

### **Performance Metrics**
- **Operation Timing**: Detailed timing statistics for all relay operations
- **Atomic Counters**: Thread-safe counters for all metrics
- **Memory Monitoring**: Runtime statistics including goroutines and memory usage
- **Query Performance**: Separate timing for query and count operations
- **Publish Performance**: Detailed forwarding statistics

### **Web Interface Enhancements**
- **API Versioning**: Organized API endpoints under `/api/v1/`
- **AJAX Loading**: Dynamic content loading with auto-refresh
- **Template Inheritance**: Reusable HTML templates to reduce duplication
- **External Assets**: Separated CSS and JavaScript for better organization
- **Responsive Layout**: Mobile-friendly design with proper scaling

### **CI/CD Pipeline**
- **Automated Testing**: Comprehensive test suite for all platforms
- **Multi-Architecture Builds**: Docker images for multiple platforms
- **Release Automation**: Automated release creation with changelogs
- **Security Scanning**: Regular vulnerability assessments with Trivy
- **Resource Optimization**: Efficient workflows to minimize GitHub Actions usage

---

## 📦 Deployment Options

### **Docker Compose (Recommended)**
```bash
# Download and extract the complete archive
wget https://github.com/girino/saint-michaels-mirror/releases/download/v1.0.0-rc2/saint-michaels-mirror-v1.0.0-rc2-complete.tar.gz
tar -xzf saint-michaels-mirror-v1.0.0-rc2-complete.tar.gz
cd saint-michaels-mirror-v1.0.0-rc2

# Configure and deploy
cp .env.example .env
# Edit .env with your configuration
docker compose -f docker-compose.prod.yml up -d
```

### **Standalone Binary**
```bash
# Extract the archive
tar -xzf saint-michaels-mirror-v1.0.0-rc2-complete.tar.gz
cd saint-michaels-mirror-v1.0.0-rc2

# Configure
cp .env.example .env
# Edit .env with your settings

# Run the appropriate binary for your platform
chmod +x saint-michaels-mirror-linux-amd64
./saint-michaels-mirror-linux-amd64
```

### **Docker Run**
```bash
# Run with latest image
docker run -d \
  --name saint-michaels-mirror \
  -p 3337:3337 \
  -e RELAY_NAME="Your Relay Name" \
  ghcr.io/girino/saint-michaels-mirror:latest

# Or with specific version
docker run -d \
  --name saint-michaels-mirror \
  -p 3337:3337 \
  -e RELAY_NAME="Your Relay Name" \
  ghcr.io/girino/saint-michaels-mirror:v1.0.0-rc2
```

---

## ⚙️ Configuration

### **Required Variables**
- `RELAY_NAME`: Display name of your relay
- `PUBLISH_REMOTES`: Comma-separated list of relays to forward published events to
- `QUERY_REMOTES`: Comma-separated list of relays to query events from

### **Optional Variables**
- `RELAY_DESCRIPTION`: Description of your relay service
- `RELAY_CONTACT`: Contact information (npub, email, etc.)
- `RELAY_SERVICE_URL`: Public URL of your relay
- `RELAY_ICON`: Path to relay icon
- `RELAY_BANNER`: Path to relay banner
- `ADDR`: Address to listen on (default: :3337)
- `VERBOSE`: Enable verbose logging (1 to enable)
- `PROD_IMAGE`: Docker image to use (defaults to latest if not set)

---

## 🌐 Web Interface

### **Main Page (`/`)**
- Relay information and NIP-11 metadata
- Contact information and service details
- Links to statistics and health monitoring

### **Statistics Page (`/stats`)**
- Real-time performance metrics
- Operation counters and timing statistics
- Memory usage and goroutine counts
- Auto-refreshes every 10 seconds

### **Health Page (`/health`)**
- Backend health status indicators
- Failure counts and health states
- Service status and version information
- Auto-refreshes every 10 seconds

### **API Endpoints**
- `/api/v1/stats`: JSON statistics data
- `/api/v1/health`: JSON health status
- `/.well-known/nostr.json`: NIP-11 relay information

---

## 🔍 Monitoring and Maintenance

### **Health Checks**
The relay provides comprehensive health monitoring:
- **Publish Health**: Tracks forwarding success/failure rates
- **Query Health**: Monitors query operation performance
- **Overall Health**: Combined health indicator
- **Thresholds**: Configurable failure thresholds for alerts

### **Metrics Available**
- Operation counters (attempts, successes, failures)
- Timing statistics (average, min, max operation times)
- Memory usage (heap, stack, garbage collection)
- Goroutine counts and system resources
- Remote relay connectivity status

### **Logging**
- Configurable verbosity levels
- Structured logging for easy parsing
- Operation tracing and debugging information
- Error reporting with context

---

## 🛡️ Security Features

### **Privacy Protection**
- Internal query filtering to prevent data leakage
- Secure key management for relay signing
- No modification of forwarded events
- Respect for client privacy and data integrity

### **Security Scanning**
- Automated vulnerability detection
- Regular security assessments
- Docker image security scanning
- Dependency vulnerability monitoring

---

## 📈 Performance

### **Optimizations**
- Concurrent event forwarding
- Efficient query aggregation
- Connection pooling for remote relays
- Atomic operations for thread-safe metrics
- Optimized memory usage and garbage collection

### **Scalability**
- Multi-architecture support
- Horizontal scaling capabilities
- Efficient resource utilization
- Configurable connection limits

---

## 🐛 Bug Fixes and Improvements

### **Major Fixes**
- Fixed Docker multi-architecture builds
- Resolved health check endpoint issues
- Corrected timing measurements for async operations
- Fixed template inheritance and asset loading

### **Improvements**
- Enhanced error handling and recovery
- Improved logging and debugging capabilities
- Better resource management and cleanup
- Optimized CI/CD pipeline performance

---

## 🔄 Migration Guide

### **From Previous Versions**
This release includes significant improvements and may require configuration updates:

1. **Update Configuration**: Review and update your `.env` file with new options
2. **Docker Users**: Update to new image tags and compose files
3. **API Users**: Update API endpoints to use new `/api/v1/` paths
4. **Monitoring**: Update health check endpoints and metrics collection

### **Breaking Changes**
- API endpoints moved from `/stats` and `/health` to `/api/v1/stats` and `/api/v1/health`
- Docker image repository changed to `ghcr.io/girino/saint-michaels-mirror`
- Some internal metrics structure changes (backward compatible for basic usage)

---

## 🎯 What's Next

### **Planned Features**
- Enhanced monitoring and alerting
- Advanced configuration options
- Performance optimization tools
- Extended API capabilities

### **Community**
- Open source development
- Community contributions welcome
- Regular updates and improvements
- Active issue tracking and support

---

## 📞 Support and Resources

### **Documentation**
- **Repository**: https://github.com/girino/saint-michaels-mirror
- **Issues**: https://github.com/girino/saint-michaels-mirror/issues
- **License**: https://license.girino.org/

### **Getting Help**
- Check the troubleshooting section in the README
- Review the configuration examples
- Open an issue on GitHub for bugs or feature requests
- Join the community discussions

---

## 🙏 Acknowledgments

- Built on the excellent [khatru](https://github.com/fiatjaf/khatru) framework
- Inspired by the Nostr protocol and community
- Thanks to all contributors and testers

---

**Espelho de São Miguel** - The mirror that returns light as truth.

*Copyright (c) 2025 Girino Vey. Licensed under Girino's Anarchist License (GAL).*
