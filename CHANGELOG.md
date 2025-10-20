# Changelog - Espelho de São Miguel

Instruction for AI agents: prioritize human-friendly, user-facing functionality; de-emphasize CI/CD and infrastructure-only changes.

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


