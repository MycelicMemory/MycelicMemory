#!/bin/bash
set -e

# Build script for ultrathink
# Creates binaries for all supported platforms

VERSION="${VERSION:-1.2.0}"
OUTPUT_DIR="${OUTPUT_DIR:-dist}"
NPM_BIN_DIR="npm/bin"

echo "Building ultrathink v${VERSION}"
echo "================================"
echo ""

# Clean and create output directories
rm -rf "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"
mkdir -p "$NPM_BIN_DIR"

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
        ./cmd/ultrathink

    if [ $? -eq 0 ]; then
        # Copy to npm bin directory
        cp "${OUTPUT_DIR}/${output_name}" "${NPM_BIN_DIR}/${output_name}"

        # Get file size
        local size=$(ls -lh "${OUTPUT_DIR}/${output_name}" | awk '{print $5}')
        echo "  -> ${output_name} (${size})"
    else
        echo "  -> FAILED (cross-compilation may require additional setup)"
    fi
}

# macOS
build_platform darwin arm64 ultrathink-macos-arm64
build_platform darwin amd64 ultrathink-macos-x64

# Linux
build_platform linux amd64 ultrathink-linux-x64
build_platform linux arm64 ultrathink-linux-arm64

# Windows
build_platform windows amd64 ultrathink-windows-x64.exe
build_platform windows arm64 ultrathink-windows-arm64.exe

echo ""
echo "Build complete!"
echo ""
echo "Binaries created in ${OUTPUT_DIR}:"
ls -lh "$OUTPUT_DIR"

echo ""
echo "npm binaries copied to ${NPM_BIN_DIR}:"
ls -lh "$NPM_BIN_DIR"
