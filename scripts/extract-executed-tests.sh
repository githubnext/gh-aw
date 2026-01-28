#!/bin/bash
# Extract executed test names from JSON test result files
# Parses the JSON output from 'go test -json' format

set -euo pipefail

if [ $# -eq 0 ]; then
  echo "Usage: $0 <test-result-dir>"
  echo "Extracts executed test names from JSON test result files in the specified directory"
  exit 1
fi

TEST_RESULT_DIR="$1"

if [ ! -d "$TEST_RESULT_DIR" ]; then
  echo "Error: Directory $TEST_RESULT_DIR does not exist"
  exit 1
fi

# Find all JSON test result files and extract test names
# Look for lines with "Action":"run" and extract the "Test" field
# Use grep with || true to prevent exit on no matches
find "$TEST_RESULT_DIR" -name "*.json" -type f | while read -r file; do
  grep '"Action":"run"' "$file" 2>/dev/null | \
    grep -o '"Test":"[^"]*"' | \
    sed 's/"Test":"\([^"]*\)"/\1/' || true
done | sort -u
