# Changelog - Espelho de São Miguel

Human-friendly, user-focused changes. CI/CD and infra-only changes are omitted.

## [v1.0.0-rc4] - 2025-10-20
### Improvements
- Deployment guide: clearer steps using release archives; added `nak` example for quick testing.
- Verification docs: corrected NIP-11 `Accept` header and added WebSocket URL guidance.

Note: No functional changes to the relay behavior in this release candidate.

## [v1.0.0-rc3] - 2025-10-19
### Added
- Release archives now ship ready-to-run assets: binaries, static files, templates, `example.env`, `docker-compose.prod.yml`, `nginx.conf.example`, and `DEPLOYMENT.md`.

### Improvements
- Easier out-of-the-box setup for Docker Compose and standalone usage.

## [Earlier]
### Major features and changes
- Renamed service from `khatru-relay` to `saint-michaels-mirror` with updated paths and assets.
- Added `/api/v1/health` and `/api/v1/stats` endpoints with backend health integration.
- New human-readable pages at `/health` and `/stats` with auto-refresh and shared layout.
- Implemented template inheritance; extracted and unified CSS/JS; branding polished.
- Relaystore health tracking with GREEN/YELLOW/RED status and failure thresholds; runtime metrics (goroutines, memory).
- Accurate timing metrics for publish, query, and count operations.
- Dockerfile and deployment flow improved; Tor/NGINX examples for production.


