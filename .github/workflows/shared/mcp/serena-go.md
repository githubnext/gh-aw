---
# Serena MCP Server - Go Code Analysis
# Language Server Protocol (LSP)-based tool for deep Go code analysis
#
# Documentation: https://github.com/oraios/serena
#
# Capabilities:
#   - Semantic code analysis using LSP (go to definition, find references, etc.)
#   - Symbol lookup and cross-file navigation
#   - Type inference and structural analysis
#   - Deeper insights than text-based grep approaches
#
# Usage:
#   imports:
#     - shared/mcp/serena-go.md

tools:
  profile: go
imports:
  - uses: ./serena.md
    with:
      languages: ["go"]
      path: '${{ env.GH_AW_SERENA_GO_PATH }}:\${PATH}'
      mounts:
        - '\${GITHUB_WORKSPACE}:\${GITHUB_WORKSPACE}:rw'
        - '\${RUNNER_TOOL_CACHE}:\${RUNNER_TOOL_CACHE}:ro'
        - '\${GH_AW_TOOL_PROFILE_GO_DIR}/bin:\${GH_AW_TOOL_PROFILE_GO_DIR}/bin:ro'
pre-agent-steps:
  - name: Expose Go to Serena container
    run: |
      set -euo pipefail
      printf 'GH_AW_SERENA_GO_PATH=%s:%s\n' "${GH_AW_TOOL_PROFILE_GO_PATH:?Go tool profile must complete before Serena setup}" "$PATH" >> "$GITHUB_ENV"
---

## Go-specific constraints

1. Analyze only `.go` files.
2. Skip files ending in `_test.go`.
3. Prioritize `pkg/` as the primary analysis area.

The Go tool profile installs Go and pinned gopls and verifies semantic readiness
before Serena setup. Both binary directories and the runner PATH are forwarded
through the MCP gateway to Serena, followed by the gateway's existing PATH.
The runner tool cache and profile binaries are mounted read-only, preserving
other setup runtimes without relying on an unmounted default GOPATH/bin.
The profile's dependency-free symbol probe verifies gopls itself, not Serena's
live adapter: a healthy MCP transport alone does not prove Go semantic analysis works.
If Serena initialization fails, report the language-server error and stop tasks
that require semantic analysis; do not substitute text-only analysis or install
tools ad hoc.
