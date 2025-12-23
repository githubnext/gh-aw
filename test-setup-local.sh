#!/usr/bin/env bash
# Local test script for setup.sh
# This script tests the setup action locally to ensure it works correctly

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "=== Testing setup.sh locally ==="
echo ""

# Step 1: Check if we're in the right directory
if [ ! -f "actions/setup/setup.sh" ]; then
  echo -e "${RED}Error: Must run from repository root${NC}"
  exit 1
fi

echo -e "${GREEN}✓${NC} Found actions/setup/setup.sh"

# Step 2: Build the actions if js/ directory doesn't exist
if [ ! -d "actions/setup/js" ]; then
  echo ""
  echo -e "${YELLOW}js/ directory not found. Running 'make actions-build'...${NC}"
  if ! make actions-build; then
    echo -e "${RED}Error: Failed to build actions${NC}"
    exit 1
  fi
  echo -e "${GREEN}✓${NC} Built actions successfully"
else
  echo -e "${GREEN}✓${NC} js/ directory already exists"
fi

# Step 3: Verify js/ directory has files
FILE_COUNT=$(ls -1 actions/setup/js/*.cjs 2>/dev/null | wc -l)
if [ "$FILE_COUNT" -eq 0 ]; then
  echo -e "${RED}Error: No .cjs files found in actions/setup/js/${NC}"
  exit 1
fi
echo -e "${GREEN}✓${NC} Found $FILE_COUNT .cjs files in actions/setup/js/"

# Step 4: Create a temporary destination directory
TEST_DEST=$(mktemp -d)
echo ""
echo "Test destination: $TEST_DEST"

# Step 5: Run setup.sh
echo ""
echo "Running setup.sh..."
export INPUT_DESTINATION="$TEST_DEST"
export GITHUB_OUTPUT="$TEST_DEST/output.txt"

if bash actions/setup/setup.sh; then
  echo -e "${GREEN}✓${NC} setup.sh executed successfully"
else
  echo -e "${RED}Error: setup.sh failed${NC}"
  rm -rf "$TEST_DEST"
  exit 1
fi

# Step 6: Verify files were copied
COPIED_COUNT=$(ls -1 "$TEST_DEST"/*.cjs 2>/dev/null | wc -l)
if [ "$COPIED_COUNT" -eq 0 ]; then
  echo -e "${RED}Error: No files were copied to destination${NC}"
  rm -rf "$TEST_DEST"
  exit 1
fi

echo -e "${GREEN}✓${NC} Copied $COPIED_COUNT files to destination"

# Step 7: Check output file
if [ -f "$GITHUB_OUTPUT" ]; then
  OUTPUT_VALUE=$(grep "files_copied=" "$GITHUB_OUTPUT" | cut -d'=' -f2)
  echo -e "${GREEN}✓${NC} Output: files_copied=$OUTPUT_VALUE"
fi

# Step 8: List some of the copied files
echo ""
echo "Sample of copied files:"
ls -1 "$TEST_DEST"/*.cjs | head -5

# Step 9: Cleanup
rm -rf "$TEST_DEST"
echo ""
echo -e "${GREEN}=== All tests passed! ===${NC}"
