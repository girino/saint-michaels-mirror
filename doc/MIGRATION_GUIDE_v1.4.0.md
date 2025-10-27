# Migration Guide - Espelho de São Miguel v1.4.0

This guide will help you migrate from v1.3.0 to v1.4.0 of Espelho de São Miguel.

---

## 📋 Overview of Changes

Version 1.4.0 introduces a **fundamental architectural change** that separates query and publish functionality:

- **RelayStore** is now **query-only** (no longer handles publishing)
- **BroadcastStore** handles all **event publishing** via intelligent relay discovery
- Configuration parameters have changed
- Some environment variables have been renamed or removed

---

## 🔄 Migration Steps

### Step 1: Review Your Current Configuration

Check your current `.env` or environment variables:

```bash
# Check your current configuration
cat .env
# or
env | grep -E 'QUERY_REMOTES|PUBLISH_REMOTES|BROADCAST'
```

### Step 2: Update Environment Variables

#### Required Changes

**If you want publishing functionality:**

1. **Remove** `PUBLISH_REMOTES` environment variable (no longer used)
2. **Rename** `BROADCAST_TOP_N` to `MAX_PUBLISH_RELAYS`
3. **Add** `BROADCAST_SEED_RELAYS` for broadcast discovery

**Example:**
```bash
# BEFORE (v1.3.0)
QUERY_REMOTES=wss://relay1.example.com,wss://relay2.example.com
PUBLISH_REMOTES=wss://relay1.example.com,wss://relay2.example.com
BROADCAST_TOP_N=10

# AFTER (v1.4.0)
QUERY_REMOTES=wss://relay1.example.com,wss://relay2.example.com
MAX_PUBLISH_RELAYS=10
BROADCAST_SEED_RELAYS=wss://relay1.example.com,wss://relay2.example.com
```

**If you only need query functionality:**

No changes needed! Your existing `QUERY_REMOTES` configuration will continue to work.

### Step 3: Update Docker Configuration (if applicable)

#### Docker Compose

Update your `docker-compose.yml`:

```yaml
# BEFORE (v1.3.0)
environment:
  - QUERY_REMOTES=wss://relay1.example.com,wss://relay2.example.com
  - PUBLISH_REMOTES=wss://relay1.example.com,wss://relay2.example.com
  - BROADCAST_TOP_N=10

# AFTER (v1.4.0)
environment:
  - QUERY_REMOTES=wss://relay1.example.com,wss://relay2.example.com
  - MAX_PUBLISH_RELAYS=10
  - BROADCAST_SEED_RELAYS=wss://relay1.example.com,wss://relay2.example.com
```

#### Docker Run

```bash
# BEFORE (v1.3.0)
docker run -d \
  -e QUERY_REMOTES=wss://relay1.example.com \
  -e PUBLISH_REMOTES=wss://relay1.example.com \
  -e BROADCAST_TOP_N=10 \
  ghcr.io/girino/saint-michaels-mirror:v1.4.0

# AFTER (v1.4.0)
docker run -d \
  -e QUERY_REMOTES=wss://relay1.example.com \
  -e MAX_PUBLISH_RELAYS=10 \
  -e BROADCAST_SEED_RELAYS=wss://relay1.example.com \
  ghcr.io/girino/saint-michaels-mirror:v1.4.0
```

### Step 4: Verify Your Configuration

Run with the `--help` flag to see all available options:

```bash
./saint-michaels-mirror --help
```

You'll see the new flags:
- `--max-publish-relays` (new, replaces removed PUBLISH_REMOTES configuration)
- `--broadcast-seed-relays` (new)
- `--broadcast-workers` (new)
- `--broadcast-cache-ttl` (new)

### Step 5: Test Your Configuration

Start with verbose logging to verify everything works:

```bash
VERBOSE=1 ./saint-michaels-mirror --addr :3337
```

Check the `/health` and `/stats` endpoints to ensure all systems are operational.

---

## 📝 Configuration Parameter Reference

### Changed Parameters

| Old Name (v1.3.0) | New Name (v1.4.0) | Notes |
|-------------------|-------------------|-------|
| `PUBLISH_REMOTES` | (removed) | No longer needed; broadcast system discovers relays automatically |
| - | `MAX_PUBLISH_RELAYS` | New parameter for maximum publish relays |

### New Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `MAX_PUBLISH_RELAYS` | 10 | Maximum number of top relays for publishing |
| `BROADCAST_SEED_RELAYS` | (required if broadcasting) | Comma-separated seed relays for broadcast discovery |
| `BROADCAST_WORKERS` | 5 | Number of worker goroutines |
| `BROADCAST_CACHE_TTL` | `1h` | Cache TTL for broadcast events |
| `BROADCAST_MANDATORY_RELAYS` | (optional) | Mandatory relays for broadcasting |

---

## ⚠️ Breaking Changes

### 1. RelayStore is Now Query-Only

**Impact:** If you were directly calling `RelayStore.PublishEvent()`, you'll need to use `BroadcastStore.SaveEvent()` instead.

**Migration:**
```go
// BEFORE
relayStore := relaystore.New(queryUrls, publishUrls, relaySecKey)
err := relayStore.PublishEvent(ctx, event)

// AFTER
broadcastStore := broadcaststore.NewBroadcastStore(cfg, 10)
err := broadcastStore.SaveEvent(ctx, event)
```

### 2. Simplified RelayStore Constructor

**Impact:** The `RelayStore.New()` signature has changed.

**Migration:**
```go
// BEFORE
rs := relaystore.New(queryUrls, publishUrls, relaySecKey)

// AFTER
rs := relaystore.New(queryUrls)
```

### 3. StatsProvider Interface Change

**Impact:** If you have custom stats providers, they need to return `json.JsonEntity`.

**Migration:**
```go
// BEFORE
func (p *MyProvider) GetStats() interface{} {
    return map[string]interface{}{
        "field1": "value1",
        "field2": 123,
    }
}

// AFTER
import "github.com/girino/nostr-lib/json"

func (p *MyProvider) GetStats() json.JsonEntity {
    obj := json.NewJsonObject()
    obj.Set("field1", json.NewJsonValue("value1"))
    obj.Set("field2", json.NewJsonValue(123))
    return obj
}
```

---

## 🧪 Testing Your Migration

### 1. Test Query Functionality

Verify that your relay can query events from upstream relays:

```bash
# Test query via WebSocket
echo '["REQ", "test", {"authors": ["npub1..."], "limit": 10}]' | \
  websocat wss://your-relay.com
```

### 2. Test Publish Functionality (if enabled)

Verify that your relay can publish events:

```bash
# Publish a test event via WebSocket
# (Use a Nostr client or test tool)
```

### 3. Check Health Endpoint

```bash
curl http://localhost:3337/api/v1/health
```

Expected response:
```json
{
  "status": "healthy",
  "main_health_state": "GREEN",
  "query_health_state": "GREEN",
  "broadcast_health_state": "GREEN"
}
```

### 4. Check Stats Endpoint

```bash
curl http://localhost:3337/api/v1/stats
```

Verify that:
- Query statistics are present
- Broadcast statistics are present (if broadcasting is enabled)
- All fields are ordered consistently

---

## 🔍 Troubleshooting

### Issue: "No query remotes provided"

**Solution:** Make sure `QUERY_REMOTES` is set:
```bash
export QUERY_REMOTES=wss://relay1.example.com,wss://relay2.example.com
```

### Issue: "BroadcastStore not initializing"

**Solution:** Add `BROADCAST_SEED_RELAYS`:
```bash
export BROADCAST_SEED_RELAYS=wss://relay1.example.com,wss://relay2.example.com
```

### Issue: Statistics not displaying

**Solution:** Check that you're using the latest version with the new stats system:
```bash
./saint-michaels-mirror --version
# Should show: v1.4.0
```

### Issue: Health endpoint shows "unknown" status

**Solution:** Check the logs for errors:
```bash
VERBOSE=1 ./saint-michaels-mirror
```

---

## 💡 Tips for a Smooth Migration

1. **Backup your configuration** before migrating
2. **Test in a staging environment** first
3. **Review your logs** after deployment for any warnings
4. **Monitor the `/health` endpoint** to ensure all systems are operational
5. **Check `/stats` page** to verify statistics are being collected properly

---

## 📚 Additional Resources

- [README.md](../README.md) - Complete documentation
- [RELEASE_NOTES_v1.4.0.md](RELEASE_NOTES_v1.4.0.md) - Detailed release notes
- [CHANGELOG.md](CHANGELOG.md) - Complete changelog

---

## ❓ Need Help?

If you encounter issues during migration:

1. Check the troubleshooting section above
2. Review the verbose logs with `VERBOSE=1`
3. Check the `/health` endpoint for system status
4. Verify your configuration against the examples in this guide

---

**Espelho de São Miguel v1.4.0** - The mirror that divides and unites.

*Copyright (c) 2025 Girino Vey. Licensed under Girino's Anarchist License (GAL).*

