#!/bin/bash
# Validate a workflow before commit

set -euo pipefail

workflow="${1:-}"

if [ -z "$workflow" ]; then
  echo "Usage: $0 <workflow.md>"
  exit 1
fi

echo "Validating $workflow..."

# Extract workflow name without extension
workflow_name="${workflow%.md}"
lock_file="${workflow_name}.lock.yml"

# Compile the workflow
echo "→ Compiling workflow..."
if ! gh aw compile "$workflow"; then
  echo "✗ Compilation failed"
  exit 1
fi

# Run actionlint (if available)
if command -v actionlint &> /dev/null; then
  echo "→ Running actionlint..."
  if ! actionlint "$lock_file"; then
    echo "✗ actionlint found issues"
    exit 1
  fi
else
  echo "⚠ actionlint not installed, skipping..."
fi

# Run zizmor (if available)
if command -v zizmor &> /dev/null; then
  echo "→ Running zizmor..."
  if ! zizmor "$lock_file"; then
    echo "⚠ zizmor found issues (non-blocking)"
  fi
else
  echo "⚠ zizmor not installed, skipping..."
fi

echo "✅ Validation passed"
