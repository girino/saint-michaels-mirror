# Release Notes - Espelho de São Miguel v1.3.0

**Release Date**: January 23, 2025  
**Version**: 1.3.0  
**Codename**: "The Sacred Mirror's Wisdom"

---

## 🌟 Overview

Version 1.3.0 represents a major milestone in the Espelho de São Miguel's evolution, introducing a comprehensive logging refactoring and advanced concurrency control. This release transforms the relay into a more maintainable, debuggable, and performant system while maintaining full backward compatibility.

## 🚀 Major Features

### 🔍 Centralized Logging System

**What's New:**
- Complete replacement of all 76+ log statements with a structured logging package
- Granular verbose control with module and method-level filtering
- Environment variable and command-line flag support
- Consistent log formatting with `[LEVEL] module.method: message` structure

**Benefits:**
- **Targeted Debugging**: Enable verbose logging for specific modules or methods
- **Production Ready**: Minimal performance impact when verbose logging is disabled
- **Docker Friendly**: Perfect integration with containerized deployments
- **CI/CD Compatible**: Environment variable control for automated testing

**Usage Examples:**
```bash
# Enable all verbose logging
VERBOSE=1 ./saint-michaels-mirror

# Enable specific module
VERBOSE=relaystore ./saint-michaels-mirror

# Enable specific method
VERBOSE=relaystore.QueryEvents ./saint-michaels-mirror

# Enable multiple modules/methods
VERBOSE=relaystore.QueryEvents,mirror,main ./saint-michaels-mirror

# Command-line override
./saint-michaels-mirror --verbose=relaystore.QueryEvents
```

### ⚡ Advanced Concurrency Control

**What's New:**
- Semaphore implementation limiting concurrent FetchMany operations to 20 simultaneous calls
- Real-time semaphore monitoring with capacity, available slots, and wait count statistics
- Prevention of upstream relay overload from excessive concurrent requests
- Enhanced query performance through better resource management

**Benefits:**
- **Upstream Protection**: Prevents "too many concurrent REQs" errors from upstream relays
- **Performance Monitoring**: Real-time visibility into semaphore contention
- **Resource Efficiency**: Better utilization of available connections
- **Scalability**: Configurable limits for different deployment scenarios

**Monitoring:**
- New "Concurrency Control" section on the `/stats` page
- Real-time updates every 10 seconds
- Visual indicators for semaphore health

## 🔧 Configuration Enhancements

### Enhanced Verbose Control

**Environment Variables:**
- `VERBOSE=1` or `VERBOSE=true`: Enable all verbose logging
- `VERBOSE=relaystore`: Enable verbose for relaystore module only
- `VERBOSE=relaystore.QueryEvents,mirror`: Enable specific methods and modules
- `VERBOSE=`: Disable all verbose logging (default)

**Command-Line Flags:**
- `--verbose=relaystore`: Override environment variable
- `--verbose=relaystore.QueryEvents`: Enable specific method
- `--verbose=1`: Enable all verbose logging

### Docker Integration

**Container Usage:**
```bash
# Enable all verbose logging
docker run -e VERBOSE=1 ghcr.io/girino/saint-michaels-mirror:latest

# Enable specific module
docker run -e VERBOSE=relaystore ghcr.io/girino/saint-michaels-mirror:latest

# Multiple modules
docker run -e VERBOSE=relaystore,mirror ghcr.io/girino/saint-michaels-mirror:latest
```

## 📊 Enhanced Monitoring

### New Statistics

**Concurrency Control Metrics:**
- **Semaphore Capacity**: Total concurrent operations allowed (20)
- **Semaphore Available**: Currently available slots
- **Semaphore Wait Count**: Queries waiting for semaphore slots

**Web Interface Updates:**
- New "Concurrency Control" card on `/stats` page
- Real-time updates with auto-refresh every 10 seconds
- Visual indicators for semaphore health and contention

### Log Format Examples

**Structured Logging:**
```
[DEBUG] relaystore.QueryEvents: attempting semaphore acquisition (wait count: 5)
[DEBUG] relaystore.QueryEvents: acquired semaphore for FetchMany (remaining slots: 19)
[INFO] mirror.StartMirroring: starting mirroring for relay wss://relay.example.com
[WARN] main: connection rate limited for IP 192.168.1.100
[ERROR] relaystore.SaveEvent: failed to publish event to relay wss://relay.example.com
```

## 🛠️ Technical Improvements

### Code Quality
- **Centralized Error Handling**: New `logging.Fatal()` function for consistent error handling
- **Cleaner Code**: Removed repetitive verbose conditionals throughout the codebase
- **Better Performance**: Verbose filtering handled efficiently by the logging package
- **Consistent API**: All logging uses the same structured format

### Architecture
- **Modular Design**: Logging package can be reused across projects
- **Environment Integration**: Perfect for systemd services and CI/CD pipelines
- **Backward Compatibility**: All existing functionality preserved
- **Future Proof**: Extensible design for additional logging features

## 🔍 Developer Experience

### Debugging Capabilities
- **Granular Control**: Debug specific modules or methods without enabling all verbose logging
- **Clear Identification**: Module.method prefixes make it easy to identify log sources
- **Flexible Configuration**: Support for both environment variables and command-line flags
- **Production Safe**: Minimal performance impact when verbose logging is disabled

### Development Workflow
```bash
# Development with specific module debugging
VERBOSE=relaystore.QueryEvents go run ./cmd/saint-michaels-mirror

# Testing with mirror debugging
VERBOSE=mirror go test -v ./...

# Production deployment
VERBOSE= ./saint-michaels-mirror
```

## 📦 Installation & Upgrade

### From Source
```bash
git clone https://github.com/girino/saint-michaels-mirror
cd saint-michaels-mirror
git checkout v1.3.0
go build -o bin/saint-michaels-mirror ./cmd/saint-michaels-mirror
```

### Docker
```bash
docker pull ghcr.io/girino/saint-michaels-mirror:v1.3.0
docker run -d --name saint-michaels-mirror \
  -p 3337:3337 \
  -e VERBOSE=relaystore \
  ghcr.io/girino/saint-michaels-mirror:v1.3.0
```

### Binary Downloads
- **Linux**: `saint-michaels-mirror-v1.3.0-linux-amd64.tar.gz`
- **macOS**: `saint-michaels-mirror-v1.3.0-darwin-amd64.tar.gz`
- **Windows**: `saint-michaels-mirror-v1.3.0-windows-amd64.zip`

## 🔄 Migration Guide

### From v1.2.x

**No Breaking Changes**: This release maintains full backward compatibility.

**New Features Available:**
1. **Enhanced Verbose Control**: Update your `VERBOSE` environment variable usage
2. **Semaphore Monitoring**: New statistics available on `/stats` page
3. **Improved Logging**: Better structured logs for debugging

**Recommended Actions:**
1. **Update Configuration**: Consider using the new granular verbose control
2. **Monitor Semaphore**: Check the new concurrency control statistics
3. **Test Verbose Logging**: Verify your debugging workflow with the new system

### Configuration Migration

**Old Configuration:**
```bash
VERBOSE=1  # All or nothing
```

**New Configuration:**
```bash
VERBOSE=relaystore.QueryEvents,mirror  # Granular control
```

## 🐛 Bug Fixes

- **Fixed Logging Inconsistencies**: All log statements now use consistent formatting
- **Improved Error Handling**: Centralized fatal error handling with `logging.Fatal()`
- **Enhanced Debugging**: Better verbose control for troubleshooting
- **Performance Optimization**: Reduced overhead from verbose conditionals

## 🔮 Future Roadmap

### Planned Features
- **Log Level Filtering**: Support for INFO, WARN, ERROR level filtering
- **Structured Logging**: JSON log format for better parsing
- **Log Rotation**: Automatic log file rotation and management
- **Metrics Export**: Prometheus metrics export for monitoring

### Performance Improvements
- **Dynamic Semaphore Sizing**: Configurable semaphore limits
- **Connection Pooling**: Enhanced connection management
- **Caching Improvements**: Better cache management strategies

## 📞 Support & Community

### Getting Help
- **Documentation**: Updated README.md with comprehensive verbose logging guide
- **Issues**: Report bugs and feature requests on GitHub
- **Discussions**: Community discussions for questions and ideas

### Contributing
- **Code Quality**: Follow the new logging standards
- **Testing**: Use granular verbose control for debugging
- **Documentation**: Update docs for new features

## 🙏 Acknowledgments

Special thanks to:
- **The Nostr Community**: For continued feedback and testing
- **Khatru Framework**: For providing the solid foundation
- **Contributors**: For helping improve the relay's capabilities

---

**Espelho de São Miguel v1.3.0** - The mirror that reflects wisdom through structured light.

*For technical support and questions, visit our [GitHub repository](https://github.com/girino/saint-michaels-mirror) or join the Nostr community discussions.*
