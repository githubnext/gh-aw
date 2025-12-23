#!/usr/bin/env bash
# Setup Action
# Copies activation job files to the agent environment
#
# This script copies JavaScript (.cjs) and JSON files from the js/ directory
# to the destination directory. The js/ directory is created by running
# 'make actions-build' which copies files from pkg/workflow/js/*.cjs
#
# Note: The js/ directory is in .gitignore as it's a build artifact.
# Workflows must ensure 'make actions-build' is run before using this action,
# or the js/ directory must be populated by another mechanism.

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

echo "::debug::Script directory: ${SCRIPT_DIR}"
echo "::debug::Looking for JavaScript sources in: ${JS_SOURCE_DIR}"

# Debug: List the contents of the script directory to understand the file layout
echo "::debug::Contents of ${SCRIPT_DIR}:"
ls -la "${SCRIPT_DIR}" || echo "::warning::Failed to list ${SCRIPT_DIR}"

# Check if js directory exists
if [ ! -d "${JS_SOURCE_DIR}" ]; then
  echo "::error::JavaScript source directory not found: ${JS_SOURCE_DIR}"
  echo "::error::This typically means 'make actions-build' was not run to populate the js/ directory"
  echo "::error::The js/ directory is a build artifact (in .gitignore) and must be created before running this script"
  
  # Additional debugging: show what's in the parent directory
  echo "::debug::Contents of parent directory $(dirname "${SCRIPT_DIR}"):"
  ls -la "$(dirname "${SCRIPT_DIR}")" || echo "::warning::Failed to list parent directory"
  
  exit 1
fi

# List files in js directory for debugging
echo "::debug::Files in ${JS_SOURCE_DIR}:"
ls -1 "${JS_SOURCE_DIR}" | head -10 || echo "::warning::Failed to list files in ${JS_SOURCE_DIR}"
FILE_COUNT_IN_DIR=$(ls -1 "${JS_SOURCE_DIR}" 2>/dev/null | wc -l)
echo "::notice::Found ${FILE_COUNT_IN_DIR} files in ${JS_SOURCE_DIR}"

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

# Copy shell scripts from sh/ directory with executable permissions
SH_SOURCE_DIR="${SCRIPT_DIR}/sh"
if [ -d "${SH_SOURCE_DIR}" ]; then
  echo "::debug::Found shell scripts directory: ${SH_SOURCE_DIR}"
  for file in "${SH_SOURCE_DIR}"/*.sh; do
    if [ -f "$file" ]; then
      filename=$(basename "$file")
      cp "$file" "${DESTINATION}/${filename}"
      chmod +x "${DESTINATION}/${filename}"
      echo "::notice::Copied shell script: ${filename}"
      FILE_COUNT=$((FILE_COUNT + 1))
    fi
  done
else
  echo "::debug::No shell scripts directory found at ${SH_SOURCE_DIR}"
fi

echo "::notice::Successfully copied ${FILE_COUNT} files to ${DESTINATION}"

# Set output
if [ -n "${GITHUB_OUTPUT}" ]; then
  echo "files_copied=${FILE_COUNT}" >> "${GITHUB_OUTPUT}"
else
  echo "::debug::GITHUB_OUTPUT not set, skipping output"
fi
