# Changelog - Espelho de São Miguel

Instruction for AI agents editing this file: prioritize human-friendly, user-facing functionality; de-emphasize CI/CD and infrastructure-only changes.

## Unreleased

### 🐛 Goroutine leak (unhealthy restarts)
- **Cause**: Docker 503s were `goroutines=RED` (~100k). Query/mirror/broadcast stayed GREEN. go-nostr `dispatchEvent` spawns a goroutine per mirrored EVENT on an unbuffered channel; a slow `BroadcastEvent` (idle crawler websockets) made that grow without bound. Idle clients that never AUTH also left 3 goroutines each.
- **Dropping mirror**: ingest queue of 4096; firehose runs only while a client has an open REQ. Excess events are dropped instead of blocking go-nostr. Drop logs are rate-limited to once per 10s. Listener count is tracked locally — khatru `GetListeningFilters()` races and panics (`index out of range`) when REQs come in concurrently. Ingest/broadcast panics are recovered so they cannot take down the process.
- **Slow consumers isolated**: mirrored events are no longer sent via khatru `BroadcastEvent` (sequential `WriteJSON` with no deadline — one stalled socket blocked everyone). Each websocket has its own bounded queue and writer goroutine. A slow client drops only its own events; others keep flowing. Writes slower than 200ms are logged (`ip`, `pubkey`, duration); writes stuck >3s disconnect **that** socket only.
- **Disconnect non-members**: whitelist AUTH timeout (8s) or non-whitelisted AUTH closes the websocket.
- **Connection cap**: reject new websockets above 256 concurrent.
- **Goroutine dump**: when count hits YELLOW/RED, log a full stack (rate-limited) so the next autoheal Discord message can show the leak site.

### 🧪 nostr-lib fanout (this branch)
- **Uses** `github.com/girino/nostr-lib@1c87b99` (`fix/getstats-deadlock-and-mirror-backpressure`): `fanout.Attach` + `mirror.StartMirroringHub` instead of in-tree dropping_mirror/client_fanout.

### 📚 Docs
- **Humans**: README mirroring/health/autoheal/testing; [DEPLOYMENT.md](DEPLOYMENT.md) wait-for-HTTP and empty-reply troubleshooting.
- **Agents**: [AGENTS.md](../AGENTS.md) (pitfalls, health vs live, no `BroadcastEvent` / `GetListeningFilters`). [CLAUDE.md](../CLAUDE.md) points there.

### 🧪 Broadcast checks
- **Reproducible nak tests**: `scripts/nak-broadcast-tests.sh` (see `doc/NAK_BROADCAST_TESTS.md`) checks local fan-out (3 readers / 2 writers) and outbound presence on mandatory + top-N relays.

### 🩺 Health diagnostics
- **Richer unhealthy-restart trail**: `/api/v1/health` now logs each Docker probe (status, subsystem colors, consecutive failures, duration). Heartbeats every 30s record goroutine/memory even if the health handler is stuck. If stats collection hangs, the process dumps goroutines and returns `503` with `reason=stats_timeout:<component>`.
- **Health probe no longer calls GetAllStats()**: Docker healthchecks read relay/mirror/broadcast/app stats directly so a stuck broadcast-manager lock cannot freeze the probe. `/api/v1/stats` still uses the collector, with a timeout and the same goroutine dump.
- **Liveness endpoint**: `GET /api/v1/live` returns `200` without touching stats (process-up check).
- **Autoheal restart details**: `POST_RESTART_SCRIPT` dumps the last Docker healthcheck output and recent health/error logs to `WEBHOOK_URL` after a restart.
- **Longer start period**: Compose healthcheck `start_period` is 600s so the initial broadcast discovery (which can take ~4–5 minutes with thousands of relays) is not marked unhealthy.

### 🪪 Access Control
- **Pubkey whitelist**: Optional `ALLOWED_NPUBS` (or `--allowed-npubs`) list of npubs or hex pubkeys. When set, clients must authenticate with NIP-42 as a listed pubkey before they can query or publish. Unlisted or unauthenticated clients are rejected with `auth-required:` / `restricted:`. NIP-11 advertises `auth_required` and `restricted_writes`. HTTP health/stats endpoints stay public.

### 🐳 Deployment
- **Local Compose file**: `docker-compose.yml` is no longer tracked. Copy `docker-compose.prod.yml` and edit the copy for host-specific ports, binds, and image tags.
- **Autoheal webhook**: Autoheal posts restart notifications to `WEBHOOK_URL` from `.env` when set.

## v1.4.0 — 2025-10-27

### 🚀 Performance & Scalability
- **Auto-scaling broadcast workers**: Default broadcast worker count is now automatically set to 2× the number of CPU cores, optimizing performance based on system resources.
- **Mandatory relay support**: `BROADCAST_MANDATORY_RELAYS` configuration now properly registers mandatory relays with the broadcast manager for tracking and prioritization.

### 📊 Enhanced Monitoring
- **Execution time tracking**: Added comprehensive execution time statistics for broadcaststore operations (SaveEvent), matching the pattern used in relaystore.
- **Stats page updates**: Stats page now displays "Avg Save Duration" and "Total Save Time" metrics in the Performance section.
- **Ordered performance metrics**: Performance statistics are now logically grouped with averages first, then totals.

### 🔧 Technical Improvements
- **Updated dependencies**: Upgraded to nostr-lib with execution time tracking support.
- **Better performance visibility**: Monitor actual broadcast execution times, excluding cached events for accurate metrics.

### 🏗️ Major Architecture Refactoring
- **Separation of Query and Publish**: RelayStore is now query-only, BroadcastStore handles all event publishing.
- **Custom JSON Library**: Introduced ordered JSON structures with `JsonObject`, `JsonValue`, and `JsonList` that preserve field order.
- **Global Stats Collection**: Singleton stats collector with automatic component registration for unified statistics.
- **Package Reorganization**: Moved BroadcastStore to `nostr-lib/eventstore/broadcaststore` for better code organization.

### 📊 Statistics & Monitoring Improvements
- **Ordered Statistics**: All statistics now display in a consistent, meaningful order thanks to ordered JSON objects.
- **Type-Safe Stats**: Replaced `interface{}` with `JsonEntity` for better type safety and compile-time checking.
- **Enhanced Health Monitoring**: Added broadcast store health indicators and improved main health calculation.
- **Unified API**: Single `stats.GetCollector().GetAllStats()` call for all statistics from all components.

### ⚙️ Configuration System Overhaul
- **Command-Line Flags**: Every environment variable now has a corresponding command-line flag for better flexibility.
- **Removed PUBLISH_REMOTES**: No longer needed as broadcast system handles relay discovery automatically.
- **Renamed MAX_PUBLISH_RELAYS**: More descriptive name (was BROADCAST_TOP_N).
- **Simplified RelayStore**: Removed unused `publishUrls` and `relaySecKey` parameters.

### 🔧 Technical Improvements
- **Broadcast System**: Intelligent relay discovery and ranking by success rate for optimal event delivery.
- **Cache Optimization**: Removed redundant local caches, uses broadcaster's cache directly.
- **Health States**: Three-tier color system (Green, Yellow, Red) with granular indicators for each subsystem.
- **Stats Duplication Fix**: Eliminated duplicate statistics in the `/stats` endpoint.

### 📝 API Changes
- **RelayStore**: Query-only with no-op `SaveEvent()`, `ReplaceEvent()`, `DeleteEvent()` methods.
- **StatsProvider**: Now returns `json.JsonEntity` instead of `interface{}`.
- **BroadcastStore**: New store for publishing events with intelligent relay selection.
- **Configuration Priority**: CLI flags override environment variables, which override defaults.

### 🐛 Bug Fixes
- **Fixed stats ordering**: Statistics now display in a consistent order.
- **Fixed broadcast cache**: Removed redundant local cache implementation.
- **Fixed stats duplication**: Eliminated duplicate stats in stats endpoint.
- **Fixed health endpoint**: Now correctly uses global stats collector.

## v1.3.0 — 2025-01-23

### 🚀 Major Logging Refactoring
- **Centralized Logging System**: Replaced all 76+ log statements with a new structured logging package providing granular verbose control and consistent formatting.
- **Environment-Based Configuration**: Verbose logging can now be controlled via `VERBOSE` environment variable or `--verbose` command-line flag with module-specific filtering.
- **Enhanced Debugging**: Support for granular verbose control (e.g., `VERBOSE=relaystore`, `VERBOSE=relaystore.QueryEvents,mirror`) for targeted debugging.

### ⚡ Concurrency Control & Performance
- **Semaphore Implementation**: Added semaphore to limit concurrent FetchMany operations to 20 simultaneous calls, preventing upstream relay overload.
- **Real-time Semaphore Monitoring**: Added semaphore statistics to the `/stats` page showing capacity, available slots, and wait counts.
- **Improved Query Performance**: Better resource management and contention tracking for query operations.

### 🔧 Configuration & Usability
- **Enhanced Verbose Control**: 
  - `VERBOSE=1` or `VERBOSE=true`: Enable all verbose logging
  - `VERBOSE=relaystore`: Enable verbose for relaystore module only
  - `VERBOSE=relaystore.QueryEvents,mirror`: Enable specific methods and modules
  - `VERBOSE=`: Disable all verbose logging (default)
- **Command-Line Override**: Verbose settings can be overridden with `--verbose` flag
- **Cleaner Configuration**: Removed deprecated `Verbose` fields from internal structs

### 📊 Enhanced Monitoring
- **Semaphore Statistics**: New "Concurrency Control" section on stats page showing:
  - Semaphore Capacity: Total concurrent operations allowed (20)
  - Semaphore Available: Currently available slots
  - Semaphore Wait Count: Queries waiting for semaphore slots
- **Better Performance Insights**: Monitor semaphore contention to identify bottlenecks
- **Real-time Updates**: Semaphore statistics auto-refresh every 10 seconds

### 🛠️ Technical Improvements
- **Centralized Error Handling**: Added `logging.Fatal()` function for consistent error handling and application exit
- **Cleaner Code**: Removed repetitive verbose conditionals throughout the codebase
- **Better Performance**: Verbose filtering handled efficiently by the logging package
- **Consistent API**: All logging now uses the same structured format with module.method prefixes

### 🔍 Developer Experience
- **Granular Debugging**: Debug specific modules or methods without enabling all verbose logging
- **Environment Integration**: Perfect for Docker containers, systemd services, and CI/CD pipelines
- **Better Troubleshooting**: Clear module.method prefixes make it easier to identify log sources
- **Flexible Configuration**: Support for both environment variables and command-line flags

## v1.2.1 — 2025-01-22

### 🐛 Bug Fixes
- **Fixed MirrorManager statistics display**: Corrected JavaScript to properly access mirror statistics from the correct data structure, fixing missing mirror statistics and health status on the web interface.

## v1.2.0 — 2025-01-22

### 🚀 Performance & Optimization
- **Internal query filtering**: Implemented intelligent caching mechanism for internal query requests to batch related operations and reduce upstream relay load.
- **Efficient blocked event lookup**: Optimized blocked event detection with O(1) map lookup instead of O(n) iteration for significantly improved performance.
- **Optimized cache locking**: Reduced lock contention with dedicated helper methods that minimize critical section duration during cache operations.
- **Smart internal request detection**: Enhanced detection of internal requests using since filter checks to prevent incorrect classification of legitimate client requests.

### 🔧 Architecture & Code Quality
- **Separated mirroring functionality**: Extracted mirroring logic into dedicated `mirror` package for better modularity and reusability.
- **Simplified constructor API**: Streamlined RelayStore constructor with intuitive parameter order (query relays mandatory, publish relays optional).
- **Removed default configurations**: Require explicit configuration for all components to prevent unexpected behavior.
- **Enhanced internal request handling**: Improved detection and handling of internal khatru requests to prevent unnecessary upstream forwarding.

### 🛡️ Reliability & Error Handling
- **Internal query blocking**: Implemented proper blocking mechanism for internal query requests with 3-second cache timeout.
- **Internal request filtering**: Added comprehensive filtering to prevent internal khatru operations from being forwarded to upstream relays.
- **Improved cache management**: Enhanced cache cleanup and management with better error handling and resource optimization.

### 📊 Technical Improvements
- **Better debugging**: Enhanced debug logging for internal requests and blocked events for improved troubleshooting.
- **Cleaner API**: Removed deprecated constructors and simplified the public API surface.
- **Optimized data structures**: Replaced complex cache entries with efficient map-based storage for blocked events.

## v1.1.0 — 2025-01-21

### 🔐 Authentication & Security
- **Authentication passthrough**: Relay now automatically authenticates with upstream relays using the configured `RELAY_SECKEY` when required, supporting both raw hex and nsec bech32 formats.
- **Enhanced NIP support**: Added NIP-42 (Authentication) to supported NIPs list for seamless upstream relay authentication.
- **Key format support**: Enhanced `RELAY_SECKEY` handling to support both raw hex keys and nsec bech32 encoded keys for maximum compatibility.

### 📡 Event Mirroring & Aggregation
- **Continuous event mirroring**: Relay automatically mirrors events from query relays using a "since now" filter, injecting them into the local relay via `khatru.BroadcastEvent()` for comprehensive event coverage.
- **Smart mirroring logic**: Only requires mirroring when `QUERY_REMOTES` is configured; gracefully handles partial relay availability.
- **Mirroring health monitoring**: Tracks live/dead relay connections and mirroring success/failure rates with configurable health thresholds.

### ⚠️ Error Handling & Reliability
- **Structured error handling**: Machine-readable error prefixes from upstream relays (NIP-01) are now passed through to clients when all publish attempts fail, including relay URLs for context.
- **Robust health system**: Enhanced health tracking with separate indicators for publish, query, and mirroring operations.
- **Fail-fast behavior**: Relay exits with clear error messages when configured query relays are unavailable.

### 📊 Enhanced Monitoring & Statistics
- **Comprehensive statistics**: Added mirroring metrics (`mirrored_events`, `mirror_attempts`, `mirror_successes`, `mirror_failures`) and relay health counters (`live_relays`, `dead_relays`).
- **Improved health indicators**: Separate health states for publish, query, and mirroring operations with configurable failure thresholds.
- **Better web interface**: Enhanced health and stats pages with new mirroring and relay health information.

### 🛠️ Technical Improvements
- **Updated dependencies**: Migrated from deprecated `SubMany` to `SubscribeMany` for better compatibility with current go-nostr library versions.
- **Improved error parsing**: Simplified error prefix extraction with proper handling of nested error messages.
- **Enhanced logging**: Better verbose logging for authentication attempts and mirroring operations.

## v1.0.0-rc4 — 2025-10-20
- Preparations for launch and documentation improvements (no functional changes).
  - Deployment guide tightened (archive-first flow, `nak` example).
  - Verification docs: NIP-11 `Accept` header fixed; WebSocket URL guidance.

## v1.0.0-rc3 — 2025-10-19
- Release packaging refinements (no functional changes).
  - Complete archives include binaries, static assets, templates, `example.env`, `docker-compose.prod.yml`, `nginx.conf.example`, `DEPLOYMENT.md`.

## v1.0.0-rc2 — 2025-10-19
- Build and release pipeline readiness (no functional changes).
  - Multi-arch builds validated; workflows stabilized.

## v1.0.0-rc1 — 2025-10-18
- Feature-freeze snapshot before launch (no new functionality added here).

## Earlier (pre-rc1) — Key functionality delivered
- Relay aggregation core: forwards publish and query operations to configured remote relays.
- Health model with relaystore counters:
  - Tracks consecutive failures for publish and query; resets counters on success.
  - Health states: GREEN (no failures), YELLOW (some failures), RED (≥10 consecutive failures).
  - Overall “main” health reflects the worst of publish/query.
- Stats and runtime metrics:
  - Goroutine count, memory usage, and other runtime indicators.
  - Timing metrics for publish, query, and count operations.
  - Query timing measures full flow including `EnsureRelay()` and `FetchMany()` and the goroutine duration.
- API endpoints:
  - `/api/v1/health`: HTTP status reflects backend health; JSON health details.
  - `/api/v1/stats`: Aggregated runtime metrics, timings, and health states.
  - NIP-11 served at `/` when `Accept: application/nostr+json` is provided.
- Web UI:
  - Human-readable pages: `/` (main), `/health`, `/stats` with auto-refresh every 10s.
  - Shared base template; 2-column layout; consistent footer and branding.
  - Externalized CSS/JS; dynamic version and names (instance name from config, project data from NIP-11 `Software`).
- Operational improvements (user-impacting):
  - Docker Compose production file with app healthcheck.
  - Dockerfile copies static/templates correctly; multi-arch ready.
  - Example environment (`example.env`) including `COMPOSE_RELAY_PORT=3337`.
  - Nginx example configuration for production deployments.
  - Tor container setup updated; removed unnecessary capabilities.
- Cleanups:
  - Removed `slicestore`.
  - Ignored binary in `.gitignore`.


