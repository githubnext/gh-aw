#!/usr/bin/env bash
# Setup Action
# Copies activation job files to the agent environment

set -e

# Get destination from input or use default
DESTINATION="${INPUT_DESTINATION:-/tmp/gh-aw/actions}"

echo "::notice::Copying activation files to ${DESTINATION}"

# Create destination directory if it doesn't exist
mkdir -p "${DESTINATION}"
echo "::notice::Created directory: ${DESTINATION}"

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
JS_SOURCE_DIR="${SCRIPT_DIR}/js"

# Check if js directory exists
if [ ! -d "${JS_SOURCE_DIR}" ]; then
  echo "::error::JavaScript source directory not found: ${JS_SOURCE_DIR}"
  exit 1
fi

# Copy all .cjs files from js/ to destination
FILE_COUNT=0
for file in "${JS_SOURCE_DIR}"/*.cjs; do
  if [ -f "$file" ]; then
    filename=$(basename "$file")
    cp "$file" "${DESTINATION}/${filename}"
    echo "::notice::Copied: ${filename}"
    FILE_COUNT=$((FILE_COUNT + 1))
  fi
done

# Copy any .json files as well
for file in "${JS_SOURCE_DIR}"/*.json; do
  if [ -f "$file" ]; then
    filename=$(basename "$file")
    cp "$file" "${DESTINATION}/${filename}"
    echo "::notice::Copied: ${filename}"
    FILE_COUNT=$((FILE_COUNT + 1))
  fi
done

echo "::notice::Successfully copied ${FILE_COUNT} files to ${DESTINATION}"

# Set output
echo "files_copied=${FILE_COUNT}" >> "${GITHUB_OUTPUT}"
