---
# gh-aw Extension - Shared Component
# Installs the gh-aw CLI extension and copies the binary
# for MCP server containerization.
#
# This component replaces the compiler-generated "Install gh-aw extension"
# step with a robust installation that handles pre-installed binaries
# (e.g., from copilot-setup-steps.yml curl-based installs).
#
# Usage:
#   imports:
#     - uses: shared/mcp/gh-aw.md

steps:
  - name: Install gh-aw extension
    env:
      GH_TOKEN: ${{ secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN || secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}
    run: |
      # Install gh-aw if not already available
      if ! gh aw --version >/dev/null 2>&1; then
        echo "Installing gh-aw extension..."
        curl -fsSL https://raw.githubusercontent.com/github/gh-aw/refs/heads/main/install-gh-aw.sh | bash
      fi
      gh aw --version
      # Copy the gh-aw binary to ${RUNNER_TEMP}/gh-aw for MCP server containerization
      mkdir -p "${RUNNER_TEMP}/gh-aw"
      GH_AW_BIN=""
      GH_AW_BIN=$(command -v gh-aw 2>/dev/null) || true
      if [ -z "$GH_AW_BIN" ]; then
        GH_AW_BIN=$(find "${HOME}/.local/share/gh/extensions/gh-aw" -name 'gh-aw' -type f 2>/dev/null | head -1) || true
      fi
      if [ -n "$GH_AW_BIN" ] && [ -f "$GH_AW_BIN" ]; then
        cp "$GH_AW_BIN" "${RUNNER_TEMP}/gh-aw/gh-aw"
        chmod +x "${RUNNER_TEMP}/gh-aw/gh-aw"
        echo "Copied gh-aw binary to ${RUNNER_TEMP}/gh-aw/gh-aw"
      else
        echo "::error::Failed to find gh-aw binary for MCP server"
        exit 1
      fi
---
