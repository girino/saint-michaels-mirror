# Changelog - Espelho de São Miguel

Instruction for AI agents editing this file: prioritize human-friendly, user-facing functionality; de-emphasize CI/CD and infrastructure-only changes.

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


