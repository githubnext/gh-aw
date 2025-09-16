#!/bin/bash

# Script to download and install gh-aw binary for the current architecture
# Usage: ./install-gh-aw.sh [version]
# If no version is specified, it will use the latest release

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if curl is available
if ! command -v curl &> /dev/null; then
    print_error "curl is required but not installed. Please install curl first."
    exit 1
fi

# Check if jq is available (optional, we'll use grep/sed as fallback)
HAS_JQ=false
if command -v jq &> /dev/null; then
    HAS_JQ=true
fi

# Determine architecture
ARCH=$(uname -m)
case $ARCH in
    x86_64)
        PLATFORM="linux-amd64"
        ;;
    aarch64|arm64)
        PLATFORM="linux-arm64"
        ;;
    armv7l)
        PLATFORM="linux-arm"
        ;;
    *)
        print_error "Unsupported architecture: $ARCH"
        print_info "Supported architectures: x86_64, aarch64/arm64, armv7l"
        exit 1
        ;;
esac

print_info "Detected architecture: $ARCH -> $PLATFORM"

# Get version (use provided version or fetch latest)
VERSION=${1:-""}
REPO="githubnext/gh-aw"

if [ -z "$VERSION" ]; then
    print_info "Fetching latest release information..."
    
    if [ "$HAS_JQ" = true ]; then
        # Use jq for JSON parsing
        LATEST_RELEASE=$(curl -s "https://api.github.com/repos/$REPO/releases/latest")
        VERSION=$(echo "$LATEST_RELEASE" | jq -r '.tag_name')
        RELEASE_NAME=$(echo "$LATEST_RELEASE" | jq -r '.name')
    else
        # Fallback to grep/sed
        LATEST_RELEASE=$(curl -s "https://api.github.com/repos/$REPO/releases/latest")
        VERSION=$(echo "$LATEST_RELEASE" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
        RELEASE_NAME=$(echo "$LATEST_RELEASE" | grep '"name"' | sed -E 's/.*"name": *"([^"]+)".*/\1/')
    fi
    
    if [ -z "$VERSION" ] || [ "$VERSION" = "null" ]; then
        print_error "Failed to fetch latest release information"
        exit 1
    fi
    
    print_info "Latest release: $RELEASE_NAME ($VERSION)"
else
    print_info "Using specified version: $VERSION"
fi

# Construct download URL
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/$PLATFORM"
BINARY_NAME="gh-aw"

print_info "Download URL: $DOWNLOAD_URL"

# Check if binary already exists
if [ -f "$BINARY_NAME" ]; then
    print_warning "Binary '$BINARY_NAME' already exists. It will be overwritten."
    read -p "Continue? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_info "Installation cancelled."
        exit 0
    fi
fi

# Download the binary
print_info "Downloading gh-aw binary..."
if curl -L -f -o "$BINARY_NAME" "$DOWNLOAD_URL"; then
    print_success "Binary downloaded successfully"
else
    print_error "Failed to download binary from $DOWNLOAD_URL"
    print_info "Please check if the version and platform combination exists in the releases."
    exit 1
fi

# Make it executable
print_info "Making binary executable..."
chmod +x "$BINARY_NAME"

# Verify the binary
print_info "Verifying binary..."
if ./"$BINARY_NAME" --help > /dev/null 2>&1; then
    print_success "Binary is working correctly!"
else
    print_error "Binary verification failed. The downloaded file may be corrupted or incompatible."
    exit 1
fi

# Show file info
FILE_SIZE=$(ls -lh "$BINARY_NAME" | awk '{print $5}')
print_success "Installation complete!"
print_info "Binary location: $(pwd)/$BINARY_NAME"
print_info "Binary size: $FILE_SIZE"
print_info "Version: $VERSION"

# Suggest adding to PATH
print_info ""
print_info "To use gh-aw from anywhere, consider adding it to your PATH:"
print_info "  sudo mv $BINARY_NAME /usr/local/bin/"
print_info "Or run it directly: ./$BINARY_NAME"

# Show version
print_info ""
print_info "Running gh-aw version check..."
./"$BINARY_NAME" version