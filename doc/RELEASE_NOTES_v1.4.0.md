# Release Notes - Espelho de São Miguel v1.4.0

**Release Date**: October 27, 2025  
**Version**: 1.4.0  
**Codename**: "The Divider of Worlds"

---

## 🌟 Overview

Version 1.4.0 represents a fundamental architectural shift in the Espelho de São Miguel, introducing a complete separation of query and publish responsibilities, along with a custom JSON library for ordered data structures and a global stats collection system. This release transforms the relay into a more modular, maintainable, and flexible system with improved statistics tracking and health monitoring, plus performance optimizations and enhanced monitoring capabilities.

## 🚀 Major Features

### ⚡ Performance & Scalability Enhancements

**Auto-Scaling Broadcast Workers:**
- Broadcast worker count now automatically defaults to 2× the number of CPU cores
- Optimizes performance based on available system resources
- No manual tuning required for most deployments
- Scale automatically with your hardware

**Mandatory Relay Support:**
- `BROADCAST_MANDATORY_RELAYS` configuration now properly registers mandatory relays with the broadcast manager
- Ensures critical relays always receive events regardless of their performance score
- Essential for ensuring events reach specific relays (e.g., your own relay or backup systems)
- Configurable per-deployment needs

### 📊 Enhanced Monitoring & Statistics

**Execution Time Tracking:**
- Comprehensive execution time statistics for broadcaststore operations (SaveEvent)
- Matches the monitoring pattern used in relaystore for consistency
- Tracks average and total execution times for performance analysis
- Excludes cached events from timing for accurate metrics

**Stats Page Improvements:**
- New "Avg Save Duration" metric in Performance section
- New "Total Save Time" metric for cumulative tracking
- Performance statistics logically grouped with averages first, then totals
- Better visibility into broadcast performance

**Updated Dependencies:**
- Upgraded to nostr-lib with execution time tracking support
- Better performance visibility across all operations

### 🎯 Separation of Query and Publish Stores

**What's New:**
- **RelayStore** is now query-only: No longer handles event publishing, only queries remote relays for events
- **BroadcastStore** handles all event publishing: Uses a sophisticated broadcast system that discovers and ranks relays
- Clear separation of concerns: Each store has a single, well-defined responsibility

**Benefits:**
- **Cleaner Architecture**: Each component has a single responsibility
- **Better Performance**: Specialized implementations for query vs. publish operations
- **Easier Maintenance**: Clear boundaries between components
- **Flexible Configuration**: Choose which stores to use based on your needs

**Technical Details:**
- `RelayStore`: Query-only, forwards events from remote relays to clients
- `BroadcastStore`: Publish-only, broadcasts events to the best available relays
- Both stores implement the `eventstore.Store` interface
- Both provide statistics via the global stats collector

### 📊 Custom JSON Library with Ordered Structures

**What's New:**
- Complete custom JSON library in `nostr-lib/json`
- Ordered `JsonObject` type that preserves field insertion order
- Type-safe `JsonValue` and `JsonList` types
- All stats providers now use `JsonEntity` instead of `interface{}`

**Benefits:**
- **Order Preservation**: Statistics are displayed in a consistent, meaningful order
- **Type Safety**: No more `interface{}` type assertions
- **Predictable Output**: JSON output is deterministic and ordered
- **Better Debugging**: Consistent field ordering makes stats easier to read

**Architecture:**
```go
type JsonEntity interface {
    MarshalJSON() ([]byte, error)
}

type JsonValue struct {
    // Supports: int, float, string, bool, null
}

type JsonObject struct {
    // Ordered map of string keys to JsonEntity values
    // Preserves insertion order
}

type JsonList struct {
    // Ordered list of JsonEntity values
}
```

### 📈 Global Stats Collection System

**What's New:**
- Singleton `StatsCollector` accessible via `stats.GetCollector()`
- All components register themselves as `StatsProvider`s
- Unified statistics endpoint with ordered JSON output
- Health endpoint uses stats collector directly

**Benefits:**
- **Unified View**: All statistics in one place
- **Automatic Registration**: Components self-register with the global collector
- **Consistent API**: Single API for accessing all statistics
- **Performance**: Efficient, thread-safe singleton pattern

**Stats Providers:**
- `RelayStore`: Query statistics (requests, failures, timing)
- `BroadcastStore`: Publish statistics (attempts, successes, health)
- `BroadcastManager`: Active relay management statistics
- `Broadcaster`: Event broadcasting statistics
- `MirrorManager`: Event mirroring statistics
- `AppStats`: Application runtime statistics

### 🏥 Enhanced Health Monitoring

**What's New:**
- Health states now include broadcast store health
- Three-tier color system (Green, Yellow, Red) based on failure thresholds
- Main health state considers all components (query, publish, mirror, broadcast)
- More granular health indicators for each subsystem

**Health Logic:**
- **GREEN**: All systems operational, no failures
- **YELLOW**: Some failures detected but within acceptable threshold
- **RED**: Critical failures exceeding acceptable threshold

**HTTP Status Codes:**
- `200 OK`: Healthy or degraded
- `503 Service Unavailable`: Unhealthy

### ⚙️ Improved Configuration System

**What's New:**
- All environment variables now have corresponding command-line flags
- Consistent naming: `--flag-name` for CLI, `FLAG_NAME` for environment
- Updated configuration parameters:
  - `MAX_PUBLISH_RELAYS` (was `BROADCAST_TOP_N`): Maximum relays for publishing
  - Removed `PUBLISH_REMOTES`: Now handled by broadcast discovery
  - Simplified `RelayStore` configuration

**Configuration Priority:**
1. Command-line flags (highest priority)
2. Environment variables (defaults)
3. Built-in defaults (fallback)

**New Flags:**
- `--max-publish-relays`: Maximum number of top relays for publishing
- `--broadcast-workers`: Number of worker goroutines
- `--broadcast-cache-ttl`: Cache TTL for broadcast events
- `--broadcast-seed-relays`: Seed relays for broadcast discovery
- `--broadcast-mandatory-relays`: Mandatory relays for broadcasting

## 🔧 Technical Changes

### Architecture Refactoring

**Before:**
```
RelayStore (Query + Publish)
  ├── Query Remotes
  ├── Publish Remotes
  └── Direct publish to upstream relays
```

**After:**
```
RelayStore (Query Only)
  └── Query Remotes

BroadcastStore (Publish Only)
  ├── BroadcastSystem
  │   ├── Discover relays from seeds
  │   ├── Rank relays by success rate
  │   └── Broadcast to top N relays
  └── In-memory event cache
```

### Package Organization

**Moved to nostr-lib:**
- `eventstore/broadcaststore` → `nostr-lib/eventstore/broadcaststore`
- Follows same pattern as `relaystore` and other event stores

**Benefits:**
- Consistent package organization
- Reusable across projects
- Better code organization

### API Changes

**RelayStore.New()** - Simplified signature:
```go
// Before
func New(queryUrls []string, publishUrls []string, relaySecKey string) *RelayStore

// After
func New(queryUrls []string) *RelayStore
```

**Removed Methods:**
- `PublishEvent()` - No longer part of RelayStore
- `RelayStore` now query-only with no-op `SaveEvent()`, `ReplaceEvent()`, `DeleteEvent()`

### Statistics API Changes

**StatsProvider Interface:**
```go
// Before
type StatsProvider interface {
    GetStats() interface{}
}

// After
type StatsProvider interface {
    GetStats() json.JsonEntity
}
```

**Benefits:**
- Type safety
- Order preservation
- Consistent structure

## 📝 Configuration Changes

### Removed Environment Variables

- `PUBLISH_REMOTES` - No longer needed (broadcast system handles this)
- `BROADCAST_TOP_N` - Renamed to `MAX_PUBLISH_RELAYS`

### Updated Environment Variables

- `MAX_PUBLISH_RELAYS` (was `BROADCAST_TOP_N`): Maximum relays for publishing

### All New Command-Line Flags

Every environment variable now has a corresponding command-line flag:

**Basic Settings:**
- `--addr`: Address to listen on
- `--query-remotes`: Query relay URLs
- `--verbose`: Verbose logging control

**Relay Identity:**
- `--relay-name`: Relay display name
- `--relay-description`: Relay description
- `--relay-contact`: Contact information
- `--relay-seckey`: Secret key for authentication
- `--relay-pubkey`: Public key
- `--relay-icon`: Icon URL
- `--relay-banner`: Banner URL

**Broadcast Settings:**
- `--max-publish-relays`: Maximum relays for publishing (default: 50)
- `--broadcast-workers`: Worker goroutines (default: 2 × CPU cores, auto-scaling)
- `--broadcast-cache-ttl`: Cache TTL (default: 1h)
- `--broadcast-seed-relays`: Seed relays for discovery
- `--broadcast-mandatory-relays`: Mandatory relays (always included in broadcasts)

## 🐛 Bug Fixes

- **Fixed stats ordering**: Statistics now display in a consistent order
- **Fixed broadcast cache**: Removed redundant local cache, uses broadcaster's cache
- **Fixed stats duplication**: Eliminated duplicate stats in `/stats` endpoint
- **Fixed health endpoint**: Now correctly uses global stats collector
- **Fixed mandatory relay registration**: Mandatory relays are now properly registered with the broadcast manager for tracking and prioritization
- **Fixed stats page layout**: Performance metrics are now logically grouped with averages first, totals after

## 🔄 Migration Guide

### For Upgraders

**If using only query functionality:**
- No changes needed
- Continue using `QUERY_REMOTES` environment variable
- Remove `PUBLISH_REMOTES` from your configuration

**If using publish functionality:**
1. Add `BROADCAST_SEED_RELAYS` environment variable
2. Remove `PUBLISH_REMOTES` environment variable
3. Update `BROADCAST_TOP_N` to `MAX_PUBLISH_RELAYS`
4. Configure `--broadcast-seed-relays` for broadcast discovery
5. (Optional) Set `BROADCAST_MANDATORY_RELAYS` for critical relays
6. (Optional) Configure `BROADCAST_WORKERS` (defaults to 2× CPU cores)

**Example Migration:**

**Before:**
```bash
QUERY_REMOTES=wss://relay1.com,wss://relay2.com
PUBLISH_REMOTES=wss://relay1.com,wss://relay2.com
BROADCAST_TOP_N=10
```

**After:**
```bash
QUERY_REMOTES=wss://relay1.com,wss://relay2.com
MAX_PUBLISH_RELAYS=10
BROADCAST_SEED_RELAYS=wss://relay.damus.io,wss://relay.nostr.band
BROADCAST_MANDATORY_RELAYS=wss://relay1.com
# BROADCAST_WORKERS defaults to 2 × CPU cores (auto-scaling)
```

### For Developers

**RelayStore:**
- Now query-only, no publish methods
- `SaveEvent()`, `ReplaceEvent()`, `DeleteEvent()` are no-ops
- Use `BroadcastStore` for publishing

**BroadcastStore:**
- New store for publishing events
- Uses broadcast system to discover and rank relays
- Provides its own statistics via global collector

**Statistics:**
- All stats providers return `JsonEntity` instead of `interface{}`
- Use `stats.GetCollector().GetAllStats()` to get all statistics
- Access stats via `allStats.Get("provider-name")`

## 📊 Performance Improvements

- **Better Concurrency**: Specialized stores for query vs. publish operations
- **Optimized Broadcasting**: Broadcast system ranks relays by success rate
- **Reduced Redundancy**: Removed duplicate cache implementations
- **Memory Efficiency**: Global stats collector reduces memory usage

## 🔍 Debugging Enhancements

- **Ordered Stats**: Statistics display in consistent order
- **Type Safety**: No more type assertions, compile-time safety
- **Better Health**: More granular health indicators for each subsystem
- **Clear Separation**: Easier to debug query vs. publish issues

## 📚 Documentation Updates

- **Updated README**: Reflects new architecture and configuration
- **Updated CHANGELOG**: Detailed list of all changes
- **Migration Guide**: Step-by-step migration instructions
- **API Documentation**: Updated for new interfaces

## 🙏 Acknowledgments

- Built on the excellent [khatru](https://github.com/fiatjaf/khatru) framework
- Thanks to all contributors and testers
- Inspired by the Nostr protocol and community

---

**Espelho de São Miguel v1.4.0** - The mirror that divides and unites.

*Copyright (c) 2025 Girino Vey. Licensed under Girino's Anarchist License (GAL).*

