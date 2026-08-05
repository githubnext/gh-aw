#!/usr/bin/env bash
set +o histexpand

# Test script for download_docker_images.sh
# Tests concurrent download functionality

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DOWNLOAD_SCRIPT="${SCRIPT_DIR}/download_docker_images.sh"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "=========================================="
echo "Testing download_docker_images.sh"
echo "=========================================="
echo ""

# Test 1: Single image download
echo -e "${YELLOW}Test 1: Single image download${NC}"
if bash "$DOWNLOAD_SCRIPT" alpine:3.19 > /tmp/test1.log 2>&1; then
    echo -e "${GREEN}✓ PASS${NC}: Single image download succeeded"
else
    echo -e "${RED}✗ FAIL${NC}: Single image download failed"
    cat /tmp/test1.log
    exit 1
fi
echo ""

# Test 2: Multiple images concurrent download
echo -e "${YELLOW}Test 2: Multiple images concurrent download${NC}"
if bash "$DOWNLOAD_SCRIPT" alpine:3.18 alpine:3.17 > /tmp/test2.log 2>&1; then
    echo -e "${GREEN}✓ PASS${NC}: Multiple images download succeeded"
    # Verify concurrent behavior by checking log contains download message
    if grep -q "Starting download of 2 image(s) with max 4 concurrent downloads" /tmp/test2.log; then
        echo -e "${GREEN}✓ PASS${NC}: Concurrent download mode confirmed"
    else
        echo -e "${RED}✗ FAIL${NC}: Expected concurrent download message not found"
        cat /tmp/test2.log
        exit 1
    fi
else
    echo -e "${RED}✗ FAIL${NC}: Multiple images download failed"
    cat /tmp/test2.log
    exit 1
fi
echo ""

# Test 3: AWF digest-pinned images restore local aliases without retagging unrelated latest refs
echo -e "${YELLOW}Test 3: Digest-pinned AWF alias handling${NC}"
WORKDIR=$(mktemp -d)
cleanup_mock_docker() {
    rm -rf "$WORKDIR"
}
cat > "$WORKDIR/docker" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >> "$DOCKER_LOG"
case "$1" in
  pull)
    exit 0
    ;;
  tag)
    exit 0
    ;;
  *)
    echo "unexpected docker command: $*" >&2
    exit 1
    ;;
esac
MOCK
chmod +x "$WORKDIR/docker"
DOCKER_LOG="$WORKDIR/docker.log"
export DOCKER_LOG
NODE_DIGEST="$(printf 'a%.0s' {1..64})"
CLI_PROXY_DIGEST="$(printf 'b%.0s' {1..64})"
if PATH="$WORKDIR:$PATH" bash "$DOWNLOAD_SCRIPT" \
    "ghcr.io/github/gh-aw-node@sha256:${NODE_DIGEST}" \
    "ghcr.io/github/gh-aw-firewall/cli-proxy:0.27.44@sha256:${CLI_PROXY_DIGEST}" \
    alpine:3.17 > /tmp/test3.log 2>&1; then
    if grep -q 'tag ghcr.io/github/gh-aw-node@sha256:' "$DOCKER_LOG" \
        && grep -q 'ghcr.io/github/gh-aw-node:latest' "$DOCKER_LOG" \
        && grep -q 'tag ghcr.io/github/gh-aw-firewall/cli-proxy:0.27.44@sha256:' "$DOCKER_LOG" \
        && grep -q 'ghcr.io/github/gh-aw-firewall/cli-proxy:0.27.44 ghcr.io/github/gh-aw-firewall/cli-proxy:latest' "$DOCKER_LOG"; then
        echo -e "${GREEN}✓ PASS${NC}: AWF digest aliases restored correctly"
    else
        echo -e "${RED}✗ FAIL${NC}: Missing expected AWF alias commands"
        cat "$DOCKER_LOG"
        exit 1
    fi
    if grep -q 'alpine:3.17 alpine:latest' "$DOCKER_LOG"; then
        echo -e "${RED}✗ FAIL${NC}: Unrelated image was incorrectly aliased to latest"
        cat "$DOCKER_LOG"
        exit 1
    else
        echo -e "${GREEN}✓ PASS${NC}: Unrelated repositories were not aliased to latest"
    fi
else
    echo -e "${RED}✗ FAIL${NC}: Digest alias test failed"
    cat /tmp/test3.log
    exit 1
fi
echo ""

# Test 4: Already cached images (should be fast)
echo -e "${YELLOW}Test 4: Already cached images${NC}"
START_TIME=$(date +%s)
if bash "$DOWNLOAD_SCRIPT" alpine:3.19 alpine:3.18 > /tmp/test4.log 2>&1; then
    END_TIME=$(date +%s)
    DURATION=$((END_TIME - START_TIME))
    echo -e "${GREEN}✓ PASS${NC}: Cached images download succeeded (${DURATION}s)"
    # Cached images should complete quickly
    if [ $DURATION -lt 10 ]; then
        echo -e "${GREEN}✓ PASS${NC}: Cached download was fast (<10s)"
    else
        echo -e "${YELLOW}⚠ WARNING${NC}: Cached download took ${DURATION}s (expected <10s)"
    fi
else
    echo -e "${RED}✗ FAIL${NC}: Cached images download failed"
    cat /tmp/test4.log
    exit 1
fi
echo ""

# Test 5: Invalid image (should fail gracefully)
echo -e "${YELLOW}Test 5: Invalid image (expected to fail)${NC}"
if bash "$DOWNLOAD_SCRIPT" "nonexistent-registry.invalid/fake-image:v999" > /tmp/test5.log 2>&1; then
    echo -e "${RED}✗ FAIL${NC}: Should have failed for invalid image"
    exit 1
else
    echo -e "${GREEN}✓ PASS${NC}: Failed as expected for invalid image"
    # Check for expected error message
    if grep -q "Failed to download" /tmp/test5.log; then
        echo -e "${GREEN}✓ PASS${NC}: Error message present"
    else
        echo -e "${YELLOW}⚠ WARNING${NC}: Expected error message format not found"
    fi
fi
echo ""

# Test 6: Empty arguments (should handle gracefully)
echo -e "${YELLOW}Test 6: No images provided${NC}"
if bash "$DOWNLOAD_SCRIPT" > /tmp/test6.log 2>&1; then
    echo -e "${GREEN}✓ PASS${NC}: Handled empty arguments gracefully"
else
    # This might fail which is also acceptable behavior
    echo -e "${YELLOW}⚠ INFO${NC}: Script exited with error for empty arguments (acceptable)"
fi
echo ""

echo "=========================================="
echo -e "${GREEN}All tests passed!${NC}"
echo "=========================================="

# Cleanup
rm -f /tmp/test*.log
cleanup_mock_docker
