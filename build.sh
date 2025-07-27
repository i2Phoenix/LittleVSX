#!/bin/bash

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

BUILD_DIR="build"
mkdir -p "$BUILD_DIR"

print_status "Starting littlevsx build..."

#PLATFORMS=("windows" "darwin" "linux")
#ARCHITECTURES=("amd64" "arm64")
PLATFORMS=("linux")
ARCHITECTURES=("amd64")

SUCCESS_COUNT=0
TOTAL_COUNT=0

for OS in "${PLATFORMS[@]}"; do
    for ARCH in "${ARCHITECTURES[@]}"; do
        TOTAL_COUNT=$((TOTAL_COUNT + 1))

        if [ "$OS" = "windows" ]; then
            EXT=".exe"
        else
            EXT=""
        fi

        OUTPUT_NAME="littlevsx-${OS}-${ARCH}${EXT}"
        OUTPUT_PATH="${BUILD_DIR}/${OUTPUT_NAME}"

        print_status "Building for ${OS}/${ARCH}..."

        export GOOS="$OS"
        export GOARCH="$ARCH"
        export CGO_ENABLED=1

        if go build -o "$OUTPUT_PATH" -ldflags="-s -w" ./main.go; then
            print_status "✓ Successfully built: $OUTPUT_NAME"
            SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
        else
            print_error "✗ Build failed for ${OS}/${ARCH}"
        fi
    done
done

echo ""
print_status "Build completed!"
print_status "Successfully built: $SUCCESS_COUNT of $TOTAL_COUNT"
print_status "Artifacts located in: $BUILD_DIR"

if [ $SUCCESS_COUNT -gt 0 ]; then
    echo ""
    print_status "Built files:"
    ls -la "$BUILD_DIR"/
fi

if [ $SUCCESS_COUNT -lt $TOTAL_COUNT ]; then
    print_warning "Some builds failed. Check the logs above."
    exit 1
else
    print_status "All builds completed successfully!"
fi
