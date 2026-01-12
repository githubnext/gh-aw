#!/usr/bin/env bash
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

# Test 3: Already cached images (should be fast)
echo -e "${YELLOW}Test 3: Already cached images${NC}"
START_TIME=$(date +%s)
if bash "$DOWNLOAD_SCRIPT" alpine:3.19 alpine:3.18 > /tmp/test3.log 2>&1; then
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
    cat /tmp/test3.log
    exit 1
fi
echo ""

# Test 4: Invalid image (should fail gracefully)
echo -e "${YELLOW}Test 4: Invalid image (expected to fail)${NC}"
if bash "$DOWNLOAD_SCRIPT" "nonexistent-registry.invalid/fake-image:v999" > /tmp/test4.log 2>&1; then
    echo -e "${RED}✗ FAIL${NC}: Should have failed for invalid image"
    exit 1
else
    echo -e "${GREEN}✓ PASS${NC}: Failed as expected for invalid image"
    # Check for expected error message
    if grep -q "Failed to download" /tmp/test4.log; then
        echo -e "${GREEN}✓ PASS${NC}: Error message present"
    else
        echo -e "${YELLOW}⚠ WARNING${NC}: Expected error message format not found"
    fi
fi
echo ""

# Test 5: Empty arguments (should handle gracefully)
echo -e "${YELLOW}Test 5: No images provided${NC}"
if bash "$DOWNLOAD_SCRIPT" > /tmp/test5.log 2>&1; then
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
