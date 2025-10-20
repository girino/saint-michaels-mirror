# Changelog - Espelho de São Miguel

All notable changes to this project will be documented in this file.

## [v1.0.0-rc4] - 2025-10-20
### Added
- Curated changelog format replacing auto-generated git log.

### Changed
- Release workflow now fetches full git history and tags reliably.
- Changelog generation moved from automatic to curated approach.
- Deployment guide updated to use release archives; added nak usage for testing.
- Release notes template improved; clarified verification endpoints and WebSocket URL.

## [v1.0.0-rc3] - 2025-10-19
### Added
- Complete archives now include static assets, templates, example.env, docker-compose, nginx example, and deployment guide.

### Changed
- Multi-arch Docker builds and binaries in releases (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64, windows/arm64).
- Release body now generated from external template with variable substitution.

## [Earlier]
- Major refactor from `khatru-relay` to `saint-michaels-mirror`.
- Health and stats endpoints moved to `/api/v1/health` and `/api/v1/stats`.
- Added human-readable `/health` and `/stats` pages with auto-refresh.
- Implemented template inheritance; extracted CSS/JS; UI/branding fixes.
- Relaystore health checks with consecutive failure thresholds; runtime metrics.
- Docker and deployment improvements; Tor config; nginx example.


