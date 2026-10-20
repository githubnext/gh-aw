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

imports:
  - uses: ./serena.md
    with:
      languages: ["go"]
      path: '${{ env.GH_AW_SERENA_GO_PATH }}:\${PATH}'
pre-agent-steps:
  - name: Setup Go
    uses: actions/setup-go@v7.0.0
    with:
      go-version-file: go.mod
      cache: true
  - name: Expose Go to Serena container
    run: |
      set -euo pipefail
      unset GOROOT
      go_root="$(GOTOOLCHAIN=local go env GOROOT)"
      case "$go_root" in
        "${RUNNER_TOOL_CACHE:?RUNNER_TOOL_CACHE must be set}"/go/*) ;;
        *)
          echo "Serena Go runtime is outside its read-only runner tool-cache mount; check actions/setup-go and RUNNER_TOOL_CACHE." >&2
          exit 1
          ;;
      esac
      if [ ! -x "$go_root/bin/go" ]; then
        echo "Serena Go runtime has no executable bin/go in the mounted toolchain." >&2
        exit 1
      fi
      export GOROOT="$go_root"
      printf 'GOROOT=%s\n' "$go_root" >> "$GITHUB_ENV"
      printf 'GH_AW_SERENA_GO_PATH=%s/bin:%s\n' "$go_root" "$PATH" >> "$GITHUB_ENV"
      "$go_root/bin/go" version
---

## Go-specific constraints

1. Analyze only `.go` files.
2. Skip files ending in `_test.go`.
3. Prioritize `pkg/` as the primary analysis area.

The selected Go binary directory and runner PATH are forwarded explicitly through
the MCP gateway to Serena's PATH, preserving other setup runtimes in the mounted
tool cache (for example Node for a project's TypeScript or Bash language servers)
and appending the gateway's existing PATH. No additional host paths are mounted.
The existing runner tool-cache mount remains read-only. The
setup check verifies the runner toolchain and mount boundary, not language-server
readiness: a healthy MCP transport alone does not prove Go semantic analysis works.
Root discovery clears any earlier `GOROOT`, disables automatic toolchain selection,
and refreshes the host `GOROOT` so later builds use the same selected toolchain.
If Serena initialization fails, report the language-server error and stop tasks
that require semantic analysis; do not substitute text-only analysis or install
tools ad hoc.
