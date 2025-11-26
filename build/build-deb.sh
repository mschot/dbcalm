#!/bin/bash
set -e

# DBCalm Debian Package Build Script
# This script builds a .deb package using nFPM

echo "=== DBCalm Debian Package Build ==="

# Determine version from git tags or default
VERSION=${VERSION:-$(git describe --tags --abbrev=0 2>/dev/null || echo "0.1.0")}
echo "Building version: $VERSION"

# Ensure we're in the app root directory (parent of build/)
cd "$(dirname "$0")/.."

# Step 1: Clean previous .deb builds
echo "Cleaning previous .deb builds..."
rm -f build/dist/*.deb 2>/dev/null || true
mkdir -p build/dist/

# Step 2: Build all binaries
echo "Building Go binaries..."
make build

# Ensure binaries are in bin/ directory
mkdir -p bin/
cp app/bin/dbcalm ./bin/dbcalm
cp cmd/dbcalm-cmd ./bin/dbcalm-cmd
cp cmd/dbcalm-db-cmd ./bin/dbcalm-db-cmd

# Verify binaries exist
if [ ! -f "./bin/dbcalm" ] || [ ! -f "./bin/dbcalm-cmd" ] || [ ! -f "./bin/dbcalm-db-cmd" ]; then
    echo "ERROR: Missing required binaries"
    ls -la ./bin/
    exit 1
fi

# Step 3: Build package with nFPM
echo "Building .deb package with nFPM..."
export VERSION
nfpm package --packager deb --target build/dist/ --config build/nfpm.yaml

# Show result
echo ""
echo "=== Build Complete ==="
ls -lh build/dist/*.deb
echo ""
echo "Package info:"
dpkg-deb -I build/dist/*.deb

# Test installation (optional)
if [ "$TEST_INSTALL" = "1" ]; then
    echo ""
    echo "=== Testing Installation ==="
    sudo dpkg -i build/dist/*.deb || (sudo apt-get install -f -y && sudo dpkg -i build/dist/*.deb)
    echo "Package installed successfully!"
fi
