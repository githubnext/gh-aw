#!/bin/bash
# Example shell script handler for safe-inputs
# Gets basic system information

set -euo pipefail

# Get system information
OS=$(uname -s)
HOSTNAME=$(hostname)
USER=$(whoami)

# Write outputs to GITHUB_OUTPUT file (GitHub Actions convention)
echo "os=$OS" >> "$GITHUB_OUTPUT"
echo "hostname=$HOSTNAME" >> "$GITHUB_OUTPUT"
echo "user=$USER" >> "$GITHUB_OUTPUT"

# Also print to stdout for visibility
echo "System Information:"
echo "  OS: $OS"
echo "  Hostname: $HOSTNAME"
echo "  User: $USER"
