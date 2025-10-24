# Verbose Logging Quick Reference - v1.3.0

## 🚀 Quick Start

### Enable All Verbose Logging
```bash
VERBOSE=1
VERBOSE=true
VERBOSE=all
```

### Enable Specific Modules
```bash
VERBOSE=relaystore
VERBOSE=mirror
VERBOSE=main
```

### Enable Specific Methods
```bash
VERBOSE=relaystore.QueryEvents
VERBOSE=mirror.StartMirroring
VERBOSE=relaystore.SaveEvent
```

### Enable Multiple Modules/Methods
```bash
VERBOSE=relaystore,mirror
VERBOSE=relaystore.QueryEvents,mirror.StartMirroring
VERBOSE=relaystore.QueryEvents,relaystore.SaveEvent,mirror
```

### Disable Verbose Logging
```bash
VERBOSE=
VERBOSE=0
VERBOSE=false
```

## 📋 Available Modules

| Module | Description | Key Methods |
|--------|-------------|-------------|
| `relaystore` | Core relay operations | `QueryEvents`, `SaveEvent`, `Init`, `ensureRelay` |
| `mirror` | Event mirroring | `StartMirroring`, `Init` |
| `main` | Application startup | `main` |

## 🔍 Available Methods

### relaystore Module
- `relaystore.QueryEvents` - Query event processing and semaphore management
- `relaystore.SaveEvent` - Event publishing and authentication
- `relaystore.Init` - Relay initialization and connection setup
- `relaystore.ensureRelay` - Relay connection management

### mirror Module
- `mirror.StartMirroring` - Mirroring process initialization
- `mirror.Init` - Mirror manager initialization

### main Module
- `main.main` - Application startup and configuration

## 🐳 Docker Usage

### Environment Variable
```bash
docker run -e VERBOSE=relaystore.QueryEvents ghcr.io/girino/saint-michaels-mirror:latest
```

### Docker Compose
```yaml
services:
  saint-michaels-mirror:
    environment:
      VERBOSE: "relaystore,mirror"
```

## 🖥️ Command Line Override

```bash
# Override environment variable
./saint-michaels-mirror --verbose=relaystore.QueryEvents

# Use environment variable
VERBOSE=mirror ./saint-michaels-mirror
```

## 📊 Log Format

### Structure
```
[LEVEL] module.method: message
```

### Examples
```
[DEBUG] relaystore.QueryEvents: attempting semaphore acquisition (wait count: 5)
[DEBUG] relaystore.QueryEvents: acquired semaphore for FetchMany (remaining slots: 19)
[INFO] mirror.StartMirroring: starting mirroring for relay wss://relay.example.com
[WARN] main: connection rate limited for IP 192.168.1.100
[ERROR] relaystore.SaveEvent: failed to publish event to relay wss://relay.example.com
```

## 🎯 Common Use Cases

### Production Debugging
```bash
# Debug query issues only
VERBOSE=relaystore.QueryEvents

# Debug mirroring issues only
VERBOSE=mirror.StartMirroring

# Debug authentication issues
VERBOSE=relaystore.SaveEvent
```

### Development Testing
```bash
# Full debugging
VERBOSE=relaystore,mirror,main

# Specific method debugging
VERBOSE=relaystore.QueryEvents,relaystore.SaveEvent

# Module-level debugging
VERBOSE=relaystore,mirror
```

### Performance Monitoring
```bash
# Monitor semaphore operations
VERBOSE=relaystore.QueryEvents

# Monitor mirroring performance
VERBOSE=mirror.StartMirroring
```

## 🔧 Troubleshooting

### Verbose Not Working
```bash
# Check environment variable
echo $VERBOSE

# Test with explicit setting
VERBOSE=relaystore ./saint-michaels-mirror --help

# Check Docker environment
docker exec container_name env | grep VERBOSE
```

### Too Much Logging
```bash
# Reduce to specific method
VERBOSE=relaystore.QueryEvents

# Disable completely
VERBOSE=
```

### Missing Logs
```bash
# Enable all verbose
VERBOSE=1

# Check specific module
VERBOSE=relaystore
```

## 📈 Performance Impact

- **VERBOSE=**: Minimal impact (production recommended)
- **VERBOSE=relaystore.QueryEvents**: Low impact (targeted debugging)
- **VERBOSE=relaystore**: Medium impact (module debugging)
- **VERBOSE=1**: Higher impact (full debugging)

## 🎨 Tips & Best Practices

1. **Start Specific**: Use method-level debugging first
2. **Monitor Performance**: Watch for high log volume
3. **Production Safe**: Use `VERBOSE=` in production
4. **Docker Friendly**: Perfect for containerized deployments
5. **CI/CD Ready**: Environment variable control for automation

---

**Need Help?** Check the full documentation in `README.md` or `MIGRATION_GUIDE_v1.3.0.md`
