#!/usr/bin/env bash
set -euo pipefail

# run.sh - build and run the khatru relay for testing
# This script no longer accepts command-line flags or environment overrides
# directly. Instead it reads configuration from .env files. It will source
# the following files (if present) in this order, allowing overrides:
#  - .env
#  - .env.local
# Any variables defined there will be exported into the environment for the
# binary to consume.

BASEDIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

echo "Loading .env files (if present)"
# export variables from .env files
set -a
[ -f "${BASEDIR}/.env" ] && source "${BASEDIR}/.env"
[ -f "${BASEDIR}/.env.local" ] && source "${BASEDIR}/.env.local"
set +a

echo "Building..."
go build -o bin/khatru-relay ./cmd/khatru-relay

echo "Starting khatru relay (configuration comes from .env files)"
exec ./bin/khatru-relay
