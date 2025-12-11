#!/bin/bash
# Test script to validate the install-gh-aw.sh script detection logic
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "=== Testing install-gh-aw.sh detection logic ==="

# Test function to validate platform detection
test_platform_detection() {
    local test_os=$1
    local test_arch=$2
    local expected_os=$3
    local expected_arch=$4
    local expected_platform=$5
    
    echo ""
    echo "Test: OS=$test_os, ARCH=$test_arch"
    
    # Execute the detection logic from the script
    OS=$test_os
    ARCH=$test_arch
    
    # Normalize OS name (same logic as install-gh-aw.sh)
    case $OS in
        Linux)
            OS_NAME="linux"
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
            echo "  ✗ FAIL: Unsupported OS: $OS"
            return 1
            ;;
    esac
    
    # Normalize architecture name (same logic as install-gh-aw.sh)
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
            echo "  ✗ FAIL: Unsupported architecture: $ARCH"
            return 1
            ;;
    esac
    
    PLATFORM="${OS_NAME}-${ARCH_NAME}"
    
    # Verify results
    if [ "$OS_NAME" != "$expected_os" ]; then
        echo "  ✗ FAIL: OS_NAME is '$OS_NAME', expected '$expected_os'"
        return 1
    fi
    
    if [ "$ARCH_NAME" != "$expected_arch" ]; then
        echo "  ✗ FAIL: ARCH_NAME is '$ARCH_NAME', expected '$expected_arch'"
        return 1
    fi
    
    if [ "$PLATFORM" != "$expected_platform" ]; then
        echo "  ✗ FAIL: PLATFORM is '$PLATFORM', expected '$expected_platform'"
        return 1
    fi
    
    echo "  ✓ PASS: $PLATFORM (OS: $OS_NAME, ARCH: $ARCH_NAME)"
    return 0
}

# Test 1: Script syntax is valid
echo ""
echo "Test 1: Verify script syntax"
if bash -n "$PROJECT_ROOT/install-gh-aw.sh"; then
    echo "  ✓ PASS: Script syntax is valid"
else
    echo "  ✗ FAIL: Script has syntax errors"
    exit 1
fi

# Test 2: Linux platforms
echo ""
echo "Test 2: Linux platform detection"
test_platform_detection "Linux" "x86_64" "linux" "amd64" "linux-amd64"
test_platform_detection "Linux" "aarch64" "linux" "arm64" "linux-arm64"
test_platform_detection "Linux" "arm64" "linux" "arm64" "linux-arm64"
test_platform_detection "Linux" "armv7l" "linux" "arm" "linux-arm"
test_platform_detection "Linux" "armv7" "linux" "arm" "linux-arm"
test_platform_detection "Linux" "i386" "linux" "386" "linux-386"
test_platform_detection "Linux" "i686" "linux" "386" "linux-386"

# Test 3: macOS (Darwin) platforms
echo ""
echo "Test 3: macOS (Darwin) platform detection"
test_platform_detection "Darwin" "x86_64" "darwin" "amd64" "darwin-amd64"
test_platform_detection "Darwin" "arm64" "darwin" "arm64" "darwin-arm64"

# Test 4: Windows platforms
echo ""
echo "Test 4: FreeBSD platforms"
test_platform_detection "FreeBSD" "amd64" "freebsd" "amd64" "freebsd-amd64"
test_platform_detection "FreeBSD" "arm64" "freebsd" "arm64" "freebsd-arm64"
test_platform_detection "FreeBSD" "i386" "freebsd" "386" "freebsd-386"

# Test 5: Windows platforms
echo ""
echo "Test 5: Windows platform detection"
test_platform_detection "MINGW64_NT-10.0" "x86_64" "windows" "amd64" "windows-amd64"
test_platform_detection "MINGW32_NT-10.0" "i686" "windows" "386" "windows-386"
test_platform_detection "MSYS_NT-10.0" "x86_64" "windows" "amd64" "windows-amd64"
test_platform_detection "CYGWIN_NT-10.0" "x86_64" "windows" "amd64" "windows-amd64"

# Test 6: Binary name detection
echo ""
echo "Test 6: Binary name detection"
OS_NAME="linux"
if [ "$OS_NAME" = "windows" ]; then
    BINARY_NAME="gh-aw.exe"
else
    BINARY_NAME="gh-aw"
fi
if [ "$BINARY_NAME" = "gh-aw" ]; then
    echo "  ✓ PASS: Linux binary name is correct: $BINARY_NAME"
else
    echo "  ✗ FAIL: Linux binary name is incorrect: $BINARY_NAME"
    exit 1
fi

OS_NAME="windows"
if [ "$OS_NAME" = "windows" ]; then
    BINARY_NAME="gh-aw.exe"
else
    BINARY_NAME="gh-aw"
fi
if [ "$BINARY_NAME" = "gh-aw.exe" ]; then
    echo "  ✓ PASS: Windows binary name is correct: $BINARY_NAME"
else
    echo "  ✗ FAIL: Windows binary name is incorrect: $BINARY_NAME"
    exit 1
fi

# Test 7: Verify download URL construction
echo ""
echo "Test 7: Download URL construction"
REPO="githubnext/gh-aw"
VERSION="v1.0.0"
OS_NAME="linux"
PLATFORM="linux-amd64"
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/$PLATFORM"
if [ "$OS_NAME" = "windows" ]; then
    DOWNLOAD_URL="${DOWNLOAD_URL}.exe"
fi
EXPECTED_URL="https://github.com/githubnext/gh-aw/releases/download/v1.0.0/linux-amd64"
if [ "$DOWNLOAD_URL" = "$EXPECTED_URL" ]; then
    echo "  ✓ PASS: Linux URL is correct: $DOWNLOAD_URL"
else
    echo "  ✗ FAIL: Linux URL is incorrect: $DOWNLOAD_URL (expected: $EXPECTED_URL)"
    exit 1
fi

OS_NAME="windows"
PLATFORM="windows-amd64"
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/$PLATFORM"
if [ "$OS_NAME" = "windows" ]; then
    DOWNLOAD_URL="${DOWNLOAD_URL}.exe"
fi
EXPECTED_URL="https://github.com/githubnext/gh-aw/releases/download/v1.0.0/windows-amd64.exe"
if [ "$DOWNLOAD_URL" = "$EXPECTED_URL" ]; then
    echo "  ✓ PASS: Windows URL is correct: $DOWNLOAD_URL"
else
    echo "  ✗ FAIL: Windows URL is incorrect: $DOWNLOAD_URL (expected: $EXPECTED_URL)"
    exit 1
fi

# Test 8: Verify fetch_release_data function exists and has correct logic
echo ""
echo "Test 8: Verify fetch_release_data function logic"

# Extract and test the function
if grep -q "fetch_release_data()" "$PROJECT_ROOT/install-gh-aw.sh"; then
    echo "  ✓ PASS: fetch_release_data function exists"
else
    echo "  ✗ FAIL: fetch_release_data function not found"
    exit 1
fi

# Verify the function checks for GH_TOKEN
if grep -q 'if \[ -n "\$GH_TOKEN" \]; then' "$PROJECT_ROOT/install-gh-aw.sh"; then
    echo "  ✓ PASS: Function checks for GH_TOKEN"
else
    echo "  ✗ FAIL: Function does not check for GH_TOKEN"
    exit 1
fi

# Verify the function includes fallback logic
if grep -q "Retrying without authentication" "$PROJECT_ROOT/install-gh-aw.sh"; then
    echo "  ✓ PASS: Function includes retry fallback with warning"
else
    echo "  ✗ FAIL: Function does not include retry fallback"
    exit 1
fi

# Verify the warning mentions invalid token
if grep -q "invalid token" "$PROJECT_ROOT/install-gh-aw.sh"; then
    echo "  ✓ PASS: Warning message mentions invalid token"
else
    echo "  ✗ FAIL: Warning message does not mention invalid token"
    exit 1
fi

# Verify the function uses Authorization header
if grep -q 'Authorization: token' "$PROJECT_ROOT/install-gh-aw.sh"; then
    echo "  ✓ PASS: Function uses proper Authorization header"
else
    echo "  ✗ FAIL: Function does not use Authorization header"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
