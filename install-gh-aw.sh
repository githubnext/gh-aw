#!/bin/bash
set +o histexpand


# Script to download and install gh-aw binary for the current OS and architecture
# Supports: Linux, macOS (Darwin), FreeBSD, Windows (Git Bash/MSYS/Cygwin)
# If no version is specified, it will use "latest"
# Note: Checksum validation is currently skipped by default (will be enabled in future releases)
# 
# Usage: ./install.sh [version] [options]
#
# Examples:
#   ./install.sh                           # Install latest version
#   ./install.sh v1.0.0                    # Install specific version
#   ./install.sh --skip-checksum           # Skip checksum validation
#
# Options:
#   --skip-checksum                   Skip checksum verification
#   --gh-install                      Try gh extension install first

set -e  # Exit on any error

# Parse arguments
SKIP_CHECKSUM=true  # Default to true until checksums are available in releases
TRY_GH_INSTALL=false  # Whether to try gh extension install first
VERSION=""

# Check if INPUT_VERSION is set (GitHub Actions context)
if [ -n "$INPUT_VERSION" ]; then
    VERSION="$INPUT_VERSION"
    TRY_GH_INSTALL=true  # In GitHub Actions, try gh install first
    SKIP_CHECKSUM=false  # Enable checksum validation in GitHub Actions
fi

for arg in "$@"; do
    case $arg in
        --skip-checksum)
            SKIP_CHECKSUM=true
            ;;
        --gh-install)
            TRY_GH_INSTALL=true
            ;;
        *)
            if [ -z "$VERSION" ]; then
                VERSION="$arg"
            fi
            ;;
    esac
done

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

# Check if HOME is set
if [ -z "$HOME" ]; then
    print_error "HOME environment variable is not set. Cannot determine installation directory."
    exit 1
fi

# Check if curl is available
if ! command -v curl &> /dev/null; then
    print_error "curl is required but not installed. Please install curl first."
    exit 1
fi

# Check if sha256sum or shasum is available (for checksum verification)
HAS_CHECKSUM_TOOL=false
CHECKSUM_CMD=""
if command -v sha256sum &> /dev/null; then
    HAS_CHECKSUM_TOOL=true
    CHECKSUM_CMD="sha256sum"
elif command -v shasum &> /dev/null; then
    HAS_CHECKSUM_TOOL=true
    CHECKSUM_CMD="shasum -a 256"
fi

if [ "$SKIP_CHECKSUM" = false ] && [ "$HAS_CHECKSUM_TOOL" = false ]; then
    print_warning "Neither sha256sum nor shasum is available. Checksum verification will be skipped."
    print_warning "To suppress this warning, use --skip-checksum flag."
    SKIP_CHECKSUM=true
fi

# Determine OS and architecture
OS=$(uname -s)
ARCH=$(uname -m)

# Normalize OS name
case $OS in
    Linux)
        if [ -n "$ANDROID_ROOT" ]; then
            OS_NAME="android"
        else
            OS_NAME="linux"
        fi
        ;;
    Darwin)
        OS_NAME="darwin"
        ;;
    FreeBSD)
        OS_NAME="freebsd"
        ;;
    MINGW*|MSYS*|CYGWIN*)
        OS_NAME="windows"
        ;;
    *)
        print_error "Unsupported operating system: $OS"
        print_info "Supported operating systems: Linux, macOS (Darwin), FreeBSD, Windows, Android (Termux)"
        exit 1
        ;;
esac

# Normalize architecture name
case $ARCH in
    x86_64|amd64)
        ARCH_NAME="amd64"
        ;;
    aarch64|arm64)
        ARCH_NAME="arm64"
        ;;
    armv7l|armv7)
        ARCH_NAME="arm"
        ;;
    i386|i686)
        ARCH_NAME="386"
        ;;
    *)
        print_error "Unsupported architecture: $ARCH"
        print_info "Supported architectures: x86_64/amd64, aarch64/arm64, armv7l/arm, i386/i686"
        exit 1
        ;;
esac

# Construct platform string
PLATFORM="${OS_NAME}-${ARCH_NAME}"

# Add .exe extension for Windows
if [ "$OS_NAME" = "windows" ]; then
    BINARY_NAME="gh-aw.exe"
else
    BINARY_NAME="gh-aw"
fi

print_info "Detected OS: $OS -> $OS_NAME"
print_info "Detected architecture: $ARCH -> $ARCH_NAME"
print_info "Platform: $PLATFORM"

# Get version (use provided version or default to "latest")
# VERSION is already set from argument parsing
REPO="github/gh-aw"

if [ -z "$VERSION" ]; then
    print_info "No version specified, using 'latest'..."
    VERSION="latest"
else
    print_info "Using specified version: $VERSION"
fi

# Try gh extension install if requested (and gh is available)
if [ "$TRY_GH_INSTALL" = true ] && command -v gh &> /dev/null; then
    print_info "Attempting to install gh-aw using 'gh extension install'..."

    # On Windows, `gh extension install` has been observed to hang for the full
    # job timeout (~10 min) without producing any output.  Root-cause hypothesis:
    # the gh CLI executes the newly downloaded gh-aw.exe for verification, and
    # Windows Defender Real-Time Protection blocks that execution while scanning
    # the new binary.  Apply a generous-but-bounded timeout so we fall through to
    # the direct-download path when this happens instead of hanging indefinitely.
    GH_INSTALL_CMD_TIMEOUT=""
    if [[ "$OS_NAME" == "windows" ]] && command -v timeout &>/dev/null; then
        GH_INSTALL_CMD_TIMEOUT="timeout 90"
        print_info "Windows detected: wrapping gh extension install with a 90s timeout"
    fi

    # Call gh extension install directly to avoid command injection
    install_result=0
    if [ -n "$VERSION" ] && [ "$VERSION" != "latest" ]; then
        # shellcheck disable=SC2086  # GH_INSTALL_CMD_TIMEOUT is intentionally unquoted
        $GH_INSTALL_CMD_TIMEOUT gh extension install "$REPO" --force --pin "$VERSION" 2>&1 | tee /tmp/gh-install.log
        install_result=${PIPESTATUS[0]}
    else
        # shellcheck disable=SC2086  # GH_INSTALL_CMD_TIMEOUT is intentionally unquoted
        $GH_INSTALL_CMD_TIMEOUT gh extension install "$REPO" --force 2>&1 | tee /tmp/gh-install.log
        install_result=${PIPESTATUS[0]}
    fi

    # Exit code 124 means the timeout command fired; treat it the same as a
    # failed install so we fall through to the direct-download path below.
    if [ $install_result -eq 124 ]; then
        print_warning "gh extension install timed out (90s) — falling back to manual installation."
        print_warning "This is a known issue on Windows where Defender scans the new binary."
        install_result=1
    fi
    
    if [ $install_result -eq 0 ]; then
        # Verify the installation succeeded
        if gh aw version &> /dev/null; then
            INSTALLED_VERSION=$(gh aw version 2>&1 | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' | head -1)
            
            # Ensure we could parse an installed version; empty means verification failed
            if [ -z "$INSTALLED_VERSION" ]; then
                print_warning "gh extension install completed but the installed gh-aw version could not be determined"
                print_info "Falling back to manual installation..."
            else
                # Verify the installed version matches the requested version (if specific version was requested)
                if [ "$VERSION" != "latest" ] && [ "$INSTALLED_VERSION" != "$VERSION" ]; then
                    print_warning "Version mismatch: requested $VERSION but gh extension install installed $INSTALLED_VERSION"
                    print_info "Falling back to manual installation to install the correct version..."
                else
                    print_success "Successfully installed gh-aw using gh extension install"
                    print_info "Installed version: $INSTALLED_VERSION"
                    
                    # Set output for GitHub Actions
                    if [ -n "${GITHUB_OUTPUT}" ]; then
                        echo "installed_version=${INSTALLED_VERSION}" >> "${GITHUB_OUTPUT}"
                    fi
                    
                    exit 0
                fi
            fi
        else
            print_warning "gh extension install completed but verification failed"
            print_info "Falling back to manual installation..."
        fi
    else
        print_warning "gh extension install failed, falling back to manual installation..."
        if [ -f /tmp/gh-install.log ]; then
            cat /tmp/gh-install.log
        fi
    fi
elif [ "$TRY_GH_INSTALL" = true ]; then
    print_info "gh CLI not available, proceeding with manual installation..."
fi

# Construct download URL and paths
if [ "$VERSION" = "latest" ]; then
    DOWNLOAD_URL="https://github.com/$REPO/releases/latest/download/$PLATFORM"
    CHECKSUMS_URL="https://github.com/$REPO/releases/latest/download/checksums.txt"
else
    DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/$PLATFORM"
    CHECKSUMS_URL="https://github.com/$REPO/releases/download/$VERSION/checksums.txt"
fi
if [ "$OS_NAME" = "windows" ]; then
    DOWNLOAD_URL="${DOWNLOAD_URL}.exe"
fi
INSTALL_DIR="$HOME/.local/share/gh/extensions/gh-aw"
BINARY_PATH="$INSTALL_DIR/$BINARY_NAME"
CHECKSUMS_PATH="$INSTALL_DIR/checksums.txt"

print_info "Download URL: $DOWNLOAD_URL"
print_info "Installation directory: $INSTALL_DIR"

# Create the installation directory if it doesn't exist
if [ ! -d "$INSTALL_DIR" ]; then
    print_info "Creating installation directory..."
    mkdir -p "$INSTALL_DIR"
fi

# Check if binary already exists
if [ -f "$BINARY_PATH" ]; then
    print_warning "Binary '$BINARY_PATH' already exists. It will be overwritten."
fi

# Download the binary with retry logic
print_info "Downloading gh-aw binary..."
MAX_RETRIES=3
RETRY_DELAY=2

for attempt in $(seq 1 $MAX_RETRIES); do
    if curl -L -f -o "$BINARY_PATH" "$DOWNLOAD_URL"; then
        print_success "Binary downloaded successfully"
        break
    else
        if [ "$attempt" -eq "$MAX_RETRIES" ]; then
            print_error "Failed to download binary from $DOWNLOAD_URL after $MAX_RETRIES attempts"
            print_info "Please check if the version and platform combination exists in the releases."
            exit 1
        else
            print_warning "Download attempt $attempt failed. Retrying in ${RETRY_DELAY}s..."
            sleep $RETRY_DELAY
            RETRY_DELAY=$((RETRY_DELAY * 2))
        fi
    fi
done

# Download and verify checksums if not skipped
if [ "$SKIP_CHECKSUM" = false ]; then
    print_info "Downloading checksums file..."
    CHECKSUMS_DOWNLOADED=false
    
    for attempt in $(seq 1 $MAX_RETRIES); do
        if curl -L -f -o "$CHECKSUMS_PATH" "$CHECKSUMS_URL" 2>/dev/null; then
            CHECKSUMS_DOWNLOADED=true
            print_success "Checksums file downloaded successfully"
            break
        else
            if [ "$attempt" -eq "$MAX_RETRIES" ]; then
                print_warning "Failed to download checksums file after $MAX_RETRIES attempts"
                print_warning "Checksum verification will be skipped for this version."
                print_info "This may occur for older releases that don't include checksums."
                break
            else
                print_warning "Checksum download attempt $attempt failed. Retrying in 2s..."
                sleep 2
            fi
        fi
    done
    
    # Verify checksum if we downloaded it successfully
    if [ "$CHECKSUMS_DOWNLOADED" = true ]; then
        print_info "Verifying binary checksum..."
        
        # Determine the expected filename in the checksums file
        EXPECTED_FILENAME="$PLATFORM"
        if [ "$OS_NAME" = "windows" ]; then
            EXPECTED_FILENAME="${PLATFORM}.exe"
        fi
        
        # Extract the expected checksum from the checksums file
        EXPECTED_CHECKSUM=$(grep "$EXPECTED_FILENAME" "$CHECKSUMS_PATH" | awk '{print $1}')
        
        if [ -z "$EXPECTED_CHECKSUM" ]; then
            print_warning "Checksum for $EXPECTED_FILENAME not found in checksums file"
            print_warning "Checksum verification will be skipped."
        else
            # Compute the actual checksum of the downloaded binary
            ACTUAL_CHECKSUM=$($CHECKSUM_CMD "$BINARY_PATH" | awk '{print $1}')
            
            if [ "$ACTUAL_CHECKSUM" = "$EXPECTED_CHECKSUM" ]; then
                print_success "Checksum verification passed!"
                print_info "Expected: $EXPECTED_CHECKSUM"
                print_info "Actual:   $ACTUAL_CHECKSUM"
            else
                print_error "Checksum verification failed!"
                print_error "Expected: $EXPECTED_CHECKSUM"
                print_error "Actual:   $ACTUAL_CHECKSUM"
                print_error "The downloaded binary may be corrupted or tampered with."
                print_info "To skip checksum verification, use: ./install-gh-aw.sh $VERSION --skip-checksum"
                rm -f "$BINARY_PATH"
                exit 1
            fi
        fi
        
        # Clean up checksums file
        rm -f "$CHECKSUMS_PATH"
    fi
else
    print_warning "Checksum verification skipped (--skip-checksum flag used)"
fi

# Make it executable
print_info "Making binary executable..."
chmod +x "$BINARY_PATH"

# Verify the binary
print_info "Verifying binary..."
if "$BINARY_PATH" --help > /dev/null 2>&1; then
    print_success "Binary is working correctly!"
else
    print_error "Binary verification failed. The downloaded file may be corrupted or incompatible."
    exit 1
fi

# Show file info
FILE_SIZE=$(ls -lh "$BINARY_PATH" | awk '{print $5}')
print_success "Installation complete!"
print_info "Binary location: $BINARY_PATH"
print_info "Binary size: $FILE_SIZE"
print_info "Version: $VERSION"

# Show usage info
print_info ""
print_info "You can now use gh-aw with the gh CLI:"
print_info "  gh aw --help"
print_info "  gh aw version"

# Show version
print_info ""
print_info "Running gh-aw version check..."
"$BINARY_PATH" version

# Set output for GitHub Actions
if [ -n "${GITHUB_OUTPUT}" ]; then
    echo "installed_version=${VERSION}" >> "${GITHUB_OUTPUT}"
fi
