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

import-schema:
  languages:
    type: array
    items:
      type: string
    default: ["go"]
    description: Languages to analyze with the mounted Go toolchain, optionally including other languages.

imports:
  - uses: ./serena.md
    with:
      languages: ${{ github.aw.import-inputs.languages }}
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
      go_root="$(go env GOROOT)"
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
      gopls_bin="$RUNNER_TOOL_CACHE/gh-aw/gopls/v0.23.0/bin"
      GOBIN="$gopls_bin" GOTOOLCHAIN=local "$go_root/bin/go" install golang.org/x/tools/gopls@v0.23.0
      if [ ! -x "$gopls_bin/gopls" ]; then
        echo "Serena Go language server has no executable gopls in the mounted tool cache." >&2
        exit 1
      fi
      export PATH="$go_root/bin:$gopls_bin:$PATH"
      "$go_root/bin/go" version
      "$gopls_bin/gopls" version
      probe_dir="$(mktemp -d)"
      trap 'rm -rf "$probe_dir"' EXIT
      printf 'module serena-readiness\n\ngo 1.26.0\n' > "$probe_dir/go.mod"
      printf 'package readiness\n\nfunc SerenaGoReady() {}\n' > "$probe_dir/probe.go"
      if ! (cd "$probe_dir" && GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off timeout 60s \
        "$gopls_bin/gopls" -remote= symbols "$probe_dir/probe.go") > "$probe_dir/symbols"; then
        echo "Serena Go language server failed its semantic-symbol readiness check." >&2
        exit 1
      fi
      if ! grep -q '^SerenaGoReady ' "$probe_dir/symbols"; then
        echo "Serena Go language server did not return the readiness symbol." >&2
        exit 1
      fi
      printf 'GH_AW_SERENA_GO_PATH=%s\n' "$PATH" >> "$GITHUB_ENV"
---

## Go-specific constraints

1. For Go analysis, analyze only `.go` files.
2. Skip files ending in `_test.go`.
3. Prioritize `pkg/` as the primary analysis area.

The setup step installs pinned `gopls` v0.23.0 into the runner tool cache and
requires a successful semantic-symbol request against a disposable Go module
before MCP startup. Go, `gopls`, and the runner PATH are forwarded explicitly through
the MCP gateway to Serena's PATH, preserving other setup runtimes in the mounted
tool cache (for example Node for a project's TypeScript or Bash language servers)
and appending the gateway's existing PATH. No additional host paths are mounted;
the runner tool-cache mount remains read-only. This checks the dependencies Serena
requires, not production MCP startup. For mixed-language analysis, pass
`languages: ["go", "typescript"]` when importing this wrapper.
If Serena initialization fails, report the language-server error and stop tasks
that require semantic analysis; do not substitute text-only analysis or install
tools ad hoc.
