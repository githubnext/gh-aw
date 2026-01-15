#!/usr/bin/env bash
# Create Copilot CLI directories with proper permissions
# The Copilot CLI requires write access to ~/.copilot/pkg for package extraction
mkdir -p "$HOME/.copilot/pkg"
echo "Created $HOME/.copilot/pkg directory for Copilot CLI package extraction"
