# Migration Guide - v1.2.x to v1.3.0

This guide helps you migrate from Espelho de São Miguel v1.2.x to v1.3.0, which introduces major logging improvements and concurrency control features.

## 🔄 Overview

**No Breaking Changes**: v1.3.0 maintains full backward compatibility with v1.2.x configurations and APIs.

**Key Changes:**
- Centralized logging system with granular verbose control
- Semaphore-based concurrency control for query operations
- Enhanced monitoring with semaphore statistics
- Improved debugging capabilities

## 📋 Pre-Migration Checklist

### 1. Backup Current Configuration
```bash
# Backup your current configuration
cp .env .env.backup
cp docker-compose.prod.yml docker-compose.prod.yml.backup
```

### 2. Review Current Verbose Settings
```bash
# Check your current VERBOSE configuration
grep VERBOSE .env
```

### 3. Test Current Deployment
```bash
# Ensure current deployment is working
docker compose -f docker-compose.prod.yml ps
curl http://localhost:3337/api/v1/health
```

## 🚀 Migration Steps

### Step 1: Update Configuration

#### Environment Variables

**Old Configuration (v1.2.x):**
```bash
VERBOSE=1  # All or nothing
```

**New Configuration (v1.3.0):**
```bash
# Option 1: Keep same behavior (all verbose)
VERBOSE=1
VERBOSE=true
VERBOSE=all

# Option 2: Use granular control (recommended)
VERBOSE=relaystore,mirror,main

# Option 3: Disable verbose (production)
VERBOSE=
VERBOSE=0
VERBOSE=false
```

#### Docker Compose Updates

**Update your `docker-compose.prod.yml`:**
```yaml
services:
  saint-michaels-mirror:
    image: ghcr.io/girino/saint-michaels-mirror:v1.3.0
    environment:
      # ... existing variables ...
      VERBOSE: "relaystore,mirror"  # Granular control
      # VERBOSE: "1"  # All verbose (old behavior)
      # VERBOSE: ""   # No verbose (production)
```

### Step 2: Deploy New Version

#### Docker Compose Deployment
```bash
# Pull new image
docker compose -f docker-compose.prod.yml pull

# Deploy with new version
docker compose -f docker-compose.prod.yml up -d

# Verify deployment
docker compose -f docker-compose.prod.yml ps
```

#### Binary Deployment
```bash
# Download new binary
wget https://github.com/girino/saint-michaels-mirror/releases/download/v1.3.0/saint-michaels-mirror-linux-amd64
chmod +x saint-michaels-mirror-linux-amd64

# Stop old version
systemctl stop saint-michaels-mirror

# Replace binary
cp saint-michaels-mirror-linux-amd64 /usr/local/bin/saint-michaels-mirror

# Start new version
systemctl start saint-michaels-mirror
```

### Step 3: Verify New Features

#### Check Semaphore Statistics
```bash
# Visit the stats page
curl http://localhost:3337/stats

# Look for new "Concurrency Control" section
# Should show:
# - Semaphore Capacity: 20
# - Semaphore Available: [current available]
# - Semaphore Wait Count: [current wait count]
```

#### Test Verbose Logging
```bash
# Test granular verbose control
docker exec saint-michaels-mirror sh -c 'VERBOSE=relaystore.QueryEvents /app/saint-michaels-mirror --help'

# Check logs for structured format
docker logs saint-michaels-mirror | grep "\[DEBUG\]"
```

## 🔧 Configuration Examples

### Production Configuration
```bash
# Minimal verbose logging for production
VERBOSE=

# Or disable completely
VERBOSE=0
```

### Development Configuration
```bash
# Enable specific modules for debugging
VERBOSE=relaystore.QueryEvents,mirror.StartMirroring

# Or enable all verbose logging
VERBOSE=1
```

### Testing Configuration
```bash
# Enable all modules for comprehensive testing
VERBOSE=relaystore,mirror,main

# Or enable specific methods
VERBOSE=relaystore.QueryEvents,relaystore.SaveEvent,mirror.StartMirroring
```

## 📊 New Monitoring Features

### Semaphore Statistics

**Access via Web Interface:**
- Visit `http://your-relay:3337/stats`
- Look for "Concurrency Control" section
- Monitor semaphore capacity, available slots, and wait counts

**Access via API:**
```bash
curl http://localhost:3337/api/v1/stats | jq '.relay.semaphore_capacity'
curl http://localhost:3337/api/v1/stats | jq '.relay.semaphore_available'
curl http://localhost:3337/api/v1/stats | jq '.relay.semaphore_wait_count'
```

### Log Format Changes

**Old Format (v1.2.x):**
```
[relaystore][DEBUG] attempting semaphore acquisition
[mirror][INFO] starting mirroring for relay
```

**New Format (v1.3.0):**
```
[DEBUG] relaystore.QueryEvents: attempting semaphore acquisition (wait count: 5)
[INFO] mirror.StartMirroring: starting mirroring for relay wss://relay.example.com
```

## 🐛 Troubleshooting

### Common Issues

#### 1. Verbose Logging Not Working
**Problem**: Verbose logs not appearing after migration

**Solution**:
```bash
# Check environment variable
echo $VERBOSE

# Test with explicit setting
VERBOSE=relaystore ./saint-michaels-mirror --help

# Check Docker environment
docker exec saint-michaels-mirror env | grep VERBOSE
```

#### 2. Semaphore Statistics Missing
**Problem**: No semaphore statistics on stats page

**Solution**:
```bash
# Verify new version is running
docker exec saint-michaels-mirror /app/saint-michaels-mirror --version

# Check API response
curl http://localhost:3337/api/v1/stats | jq '.relay | keys'
```

#### 3. Performance Issues
**Problem**: Slower performance after migration

**Solution**:
```bash
# Check semaphore wait count
curl http://localhost:3337/api/v1/stats | jq '.relay.semaphore_wait_count'

# If wait count is high, consider:
# 1. Increasing semaphore capacity (future feature)
# 2. Optimizing query patterns
# 3. Adding more query relays
```

### Rollback Procedure

If you need to rollback to v1.2.x:

```bash
# Docker Compose
docker compose -f docker-compose.prod.yml down
docker compose -f docker-compose.prod.yml.prod.yml.backup up -d

# Binary
systemctl stop saint-michaels-mirror
cp /usr/local/bin/saint-michaels-mirror.backup /usr/local/bin/saint-michaels-mirror
systemctl start saint-michaels-mirror
```

## ✅ Post-Migration Verification

### 1. Health Check
```bash
curl http://localhost:3337/api/v1/health
# Should return 200 OK with health status
```

### 2. Statistics Check
```bash
curl http://localhost:3337/api/v1/stats
# Should include semaphore statistics
```

### 3. Verbose Logging Test
```bash
# Test granular verbose control
VERBOSE=relaystore.QueryEvents ./saint-michaels-mirror --help
# Should show structured debug logs
```

### 4. Performance Monitoring
```bash
# Monitor semaphore statistics
watch -n 5 'curl -s http://localhost:3337/api/v1/stats | jq ".relay.semaphore_available"'
```

## 🎯 Best Practices

### Production Deployment
1. **Start with minimal verbose logging**: `VERBOSE=`
2. **Monitor semaphore statistics**: Check for high wait counts
3. **Gradually enable verbose logging**: Only when needed for debugging
4. **Use structured logs**: Take advantage of module.method prefixes

### Development Workflow
1. **Use granular verbose control**: Enable specific modules/methods
2. **Monitor semaphore contention**: Watch for performance bottlenecks
3. **Test with different configurations**: Verify behavior with various VERBOSE settings
4. **Leverage new monitoring**: Use semaphore statistics for optimization

### Troubleshooting
1. **Start with specific modules**: `VERBOSE=relaystore` instead of `VERBOSE=1`
2. **Use method-level debugging**: `VERBOSE=relaystore.QueryEvents` for specific issues
3. **Monitor semaphore health**: Check wait counts and available slots
4. **Check log format**: Verify structured logging is working correctly

## 📞 Support

If you encounter issues during migration:

1. **Check the logs**: Use the new verbose logging to identify problems
2. **Review semaphore statistics**: Monitor concurrency control metrics
3. **Test with minimal configuration**: Start with `VERBOSE=` and gradually enable features
4. **Report issues**: Create GitHub issues with detailed logs and configuration

---

**Migration completed successfully!** 🎉

Your Espelho de São Miguel relay now has enhanced logging capabilities and better concurrency control. Enjoy the improved debugging experience and performance monitoring!
