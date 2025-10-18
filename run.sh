#!/usr/bin/env bash
set -euo pipefail

# run.sh - build and run the khatru relay for testing
# Usage: ./run.sh [args]
# Environment variables to override defaults:
#  PUBLISH_REMOTES - comma-separated publish remotes (default: ws://localhost:10547)
#  QUERY_REMOTES   - comma-separated query remotes (default: wss://wot.girino.org,wss://nostr.girino.org)
#  ADDR            - address to listen on (default: :8080)
#  DATA_DIR        - data dir (default: ./data)
#  VERBOSE         - if set to 1, enables --verbose

PUBLISH_REMOTES=${PUBLISH_REMOTES:-ws://localhost:10547}
QUERY_REMOTES=${QUERY_REMOTES:-wss://wot.girino.org,wss://nostr.girino.org}
ADDR=${ADDR:-:8080}
DATA_DIR=${DATA_DIR:-./data}
VERBOSE_FLAG=""
if [ "${VERBOSE:-0}" != "0" ]; then
  VERBOSE_FLAG="--verbose"
fi

echo "Building..."
go build -o bin/khatru-relay ./cmd/khatru-relay

echo "Starting khatru relay"
./bin/khatru-relay --addr "${ADDR}" --data "${DATA_DIR}" --remotes "${PUBLISH_REMOTES}" --query-remotes "${QUERY_REMOTES}" ${VERBOSE_FLAG}
