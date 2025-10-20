#!/usr/bin/env bash
set -euo pipefail

# Copyright (c) 2025 Girino Vey.
# 
# This software is licensed under Girino's Anarchist License (GAL).
# See LICENSE file for full license text.
# License available at: https://license.girino.org/
#
# Build and run script for Espelho de São Miguel.

# run.sh - build and run the Espelho de São Miguel for testing
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

echo "Determining version from git..."
if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    # Prefer the most recent annotated or lightweight tag reachable from HEAD
    if GIT_TAG=$(git describe --tags --abbrev=0 2>/dev/null); then
        VERSION="$GIT_TAG"
    else
        # Fall back to short commit with -dev suffix
        GIT_SHA=$(git rev-parse --short HEAD)
        VERSION="${GIT_SHA}-dev"
    fi
    # Mark dirty working trees
    if ! git diff --quiet --ignore-submodules HEAD -- 2>/dev/null; then
        VERSION="${VERSION}+dirty"
    fi
else
    VERSION="dev"
fi

echo "Building... (version: ${VERSION})"
rm -f bin/saint-michaels-mirror
GOFLAGS=${GOFLAGS:-}
go build -ldflags "-X main.Version=${VERSION}" -o bin/saint-michaels-mirror ./cmd/saint-michaels-mirror

echo "Starting Espelho de São Miguel (configuration comes from .env files)"
exec ./bin/saint-michaels-mirror
