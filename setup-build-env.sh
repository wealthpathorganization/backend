#!/bin/bash
# Setup Go build environment for CI/CD agents
# This script configures Go cache directories to avoid permission issues

set -e

# Create cache directories
mkdir -p /tmp/go-cache /tmp/go-mod /tmp/go-build

# Export environment variables
export GOCACHE=/tmp/go-cache
export GOMODCACHE=/tmp/go-mod
export GOTMPDIR=/tmp/go-build

# Print configuration
echo "Go build environment configured:"
echo "  GOCACHE=${GOCACHE}"
echo "  GOMODCACHE=${GOMODCACHE}"
echo "  GOTMPDIR=${GOTMPDIR}"

# Clean old cache if needed (optional)
if [ "$CLEAN_CACHE" = "true" ]; then
    echo "Cleaning cache directories..."
    rm -rf /tmp/go-cache/* /tmp/go-mod/* /tmp/go-build/*
fi

