# Release Summary - Espelho de São Miguel v1.4.0

**Release Date**: January 27, 2025  
**Version**: 1.4.0  
**Codename**: "The Divider of Worlds"

---

## 📊 Summary

Version 1.4.0 is a **major architectural refactoring** release that fundamentally changes how the relay handles query and publish operations.

### Key Achievements

✅ **Complete separation of query and publish stores**  
✅ **Custom JSON library with ordered data structures**  
✅ **Global stats collection system**  
✅ **Comprehensive CLI flag coverage**  
✅ **Enhanced health monitoring**  
✅ **Improved code organization**

---

## 🎯 What's New

### 1. Architecture Refactoring

**Before (v1.3.0):**
- RelayStore handled both query and publish operations
- Direct publishing to configured relays
- Complex constructor with multiple parameters

**After (v1.4.0):**
- **RelayStore**: Query-only, specializes in fetching events from remote relays
- **BroadcastStore**: Publish-only, intelligently discovers and ranks relays for publishing
- Clean separation of concerns with single-responsibility components

### 2. Custom JSON Library

**Benefits:**
- Ordered `JsonObject` preserves field insertion order
- Type-safe `JsonValue` and `JsonList` structures
- Statistics display in consistent, meaningful order
- Better debugging with predictable JSON output

**Impact:**
- All stats now have consistent ordering
- No more type assertions with `interface{}`
- Compile-time safety with `JsonEntity` interface

### 3. Global Stats Collection

**Benefits:**
- Singleton collector accessible via `stats.GetCollector()`
- Components auto-register as `StatsProvider`s
- Unified statistics in one endpoint
- Efficient, thread-safe implementation

**Impact:**
- Simpler API for accessing statistics
- Unified view of all system metrics
- Automatic component discovery

### 4. Configuration Improvements

**Benefits:**
- Every environment variable has a CLI flag
- Renamed unclear parameters (e.g., `BROADCAST_TOP_N` → `MAX_PUBLISH_RELAYS`)
- Simplified RelayStore configuration
- Removed `PUBLISH_REMOTES` (now handled automatically)

**Impact:**
- Better flexibility with CLI flags
- Clearer parameter names
- Easier configuration management

---

## 📈 Statistics

### Commits

- **Total commits in v1.4.0**: 13 commits
- **nostr-lib changes**: 7 commits
- **relay-agregator changes**: 6 commits

### Code Changes

- **Packages moved**: 1 (broadcaststore to nostr-lib)
- **Packages created**: 1 (broadcaststore in nostr-lib)
- **Lines added**: ~2000+ new lines
- **Lines removed**: ~500 removed lines
- **Net change**: +1500 lines

### Files Modified

- **nostr-lib**: 8 files
- **relay-agregator**: 10 files
- **Documentation**: 3 new files

---

## 🔧 Breaking Changes

### 1. Configuration Changes

**Removed:**
- `PUBLISH_REMOTES` environment variable
- `BROADCAST_TOP_N` environment variable (renamed)

**Renamed:**
- `BROADCAST_TOP_N` → `MAX_PUBLISH_RELAYS`

**New:**
- `BROADCAST_SEED_RELAYS` (required for publishing)
- `BROADCAST_WORKERS` (optional)
- `BROADCAST_CACHE_TTL` (optional)

### 2. API Changes

**RelayStore:**
```go
// Before: relaystore.New(queryUrls, publishUrls, relaySecKey)
// After:  relaystore.New(queryUrls)
```

**StatsProvider:**
```go
// Before: GetStats() interface{}
// After:  GetStats() json.JsonEntity
```

### 3. Behavioral Changes

- RelayStore no longer publishes events (use BroadcastStore)
- Stats ordering is now consistent (was random before)
- Health checks now include broadcast store health

---

## 📋 Migration Checklist

Use this checklist to ensure a smooth migration:

- [ ] Review current configuration
- [ ] Remove `PUBLISH_REMOTES` environment variable
- [ ] Rename `BROADCAST_TOP_N` to `MAX_PUBLISH_RELAYS`
- [ ] Add `BROADCAST_SEED_RELAYS` if publishing is needed
- [ ] Update Docker configuration (if applicable)
- [ ] Update any custom code using RelayStore
- [ ] Test query functionality
- [ ] Test publish functionality (if enabled)
- [ ] Verify health endpoint
- [ ] Verify stats endpoint
- [ ] Check logs for errors

See [MIGRATION_GUIDE_v1.4.0.md](MIGRATION_GUIDE_v1.4.0.md) for detailed instructions.

---

## 🚀 Quick Start (Post-Migration)

### For Query-Only Relays

```bash
export QUERY_REMOTES=wss://relay1.com,wss://relay2.com
./saint-michaels-mirror --addr :3337
```

### For Publishing Relays

```bash
export QUERY_REMOTES=wss://relay1.com,wss://relay2.com
export MAX_PUBLISH_RELAYS=10
export BROADCAST_SEED_RELAYS=wss://relay1.com,wss://relay2.com
./saint-michaels-mirror --addr :3337
```

### With All CLI Flags

```bash
./saint-michaels-mirror \
  --addr :3337 \
  --query-remotes wss://relay1.com,wss://relay2.com \
  --max-publish-relays 10 \
  --broadcast-seed-relays wss://relay1.com,wss://relay2.com
```

---

## 📚 Documentation

### New Documents

- [RELEASE_NOTES_v1.4.0.md](RELEASE_NOTES_v1.4.0.md) - Comprehensive release notes
- [MIGRATION_GUIDE_v1.4.0.md](MIGRATION_GUIDE_v1.4.0.md) - Step-by-step migration
- [RELEASE_SUMMARY_v1.4.0.md](RELEASE_SUMMARY_v1.4.0.md) - This document

### Updated Documents

- [CHANGELOG.md](CHANGELOG.md) - Added v1.4.0 entry
- [README.md](../README.md) - Update pending

---

## 🙏 Acknowledgments

Thank you to everyone who contributed to this release:

- All testers and early adopters
- The Nostr community for inspiration and feedback
- Contributors to nostr-lib and other dependencies

---

## 📞 Support

- **Documentation**: See [doc/](doc/) directory
- **Issues**: Report via your preferred communication channel
- **Questions**: Refer to migration guide or release notes

---

**Espelho de São Miguel v1.4.0** - The mirror that divides and unites.

*Copyright (c) 2025 Girino Vey. Licensed under Girino's Anarchist License (GAL).*

