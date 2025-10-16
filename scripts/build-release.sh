#!/bin/bash
# Custom build script for gh-extension-precompile to set version correctly
# This script is called during the release process to build binaries with proper version info
set -e

VERSION="$1"

if [ -z "$VERSION" ]; then
  echo "error: VERSION argument is required" >&2
  exit 1
fi

platforms=(
  darwin-amd64
  darwin-arm64
  freebsd-386
  freebsd-amd64
  freebsd-arm64
  linux-386
  linux-amd64
  linux-arm
  linux-arm64
  windows-386
  windows-amd64
  windows-arm64
)

echo "Building binaries with version: $VERSION"

# Create dist directory if it doesn't exist
mkdir -p dist

IFS=$'\n' read -d '' -r -a supported_platforms < <(go tool dist list) || true

for p in "${platforms[@]}"; do
  goos="${p%-*}"
  goarch="${p#*-}"
  
  # Check if platform is supported
  if [[ " ${supported_platforms[*]} " != *" ${goos}/${goarch} "* ]]; then
    echo "warning: skipping unsupported platform $p" >&2
    continue
  fi
  
  ext=""
  if [ "$goos" = "windows" ]; then
    ext=".exe"
  fi
  
  echo "Building $p..."
  GOOS="$goos" GOARCH="$goarch" go build \
    -trimpath \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o "dist/${p}${ext}" \
    ./cmd/gh-aw
done

echo "Build complete. Binaries:"
ls -lh dist/
