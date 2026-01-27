#!/bin/bash
set -e

# Build script for mycelicmemory
# Creates binaries for all supported platforms for GitHub releases

VERSION="${VERSION:-$(node -p "require('./package.json').version")}"
OUTPUT_DIR="${OUTPUT_DIR:-dist}"

echo "Building mycelicmemory v${VERSION}"
echo "================================"
echo ""

# Clean and create output directories
rm -rf "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"

# Build flags
LDFLAGS="-s -w -X main.Version=${VERSION}"
BUILD_TAGS="fts5"

# Build for each platform
build_platform() {
    local goos=$1
    local goarch=$2
    local output_name=$3

    echo "Building for ${goos}/${goarch}..."

    CGO_ENABLED=1 GOOS="$goos" GOARCH="$goarch" go build \
        -tags "$BUILD_TAGS" \
        -ldflags "$LDFLAGS" \
        -o "${OUTPUT_DIR}/${output_name}" \
        ./cmd/mycelicmemory

    if [ $? -eq 0 ]; then
        local size=$(ls -lh "${OUTPUT_DIR}/${output_name}" | awk '{print $5}')
        echo "  -> ${output_name} (${size})"
    else
        echo "  -> FAILED (cross-compilation may require additional setup)"
    fi
}

# macOS
build_platform darwin arm64 mycelicmemory-macos-arm64
build_platform darwin amd64 mycelicmemory-macos-x64

# Linux
build_platform linux amd64 mycelicmemory-linux-x64
build_platform linux arm64 mycelicmemory-linux-arm64

# Windows
build_platform windows amd64 mycelicmemory-windows-x64.exe

echo ""
echo "Build complete!"
echo ""
echo "Binaries created in ${OUTPUT_DIR}:"
ls -lh "$OUTPUT_DIR"
echo ""
echo "Upload these to GitHub releases as v${VERSION}"
