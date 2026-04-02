---
# Go Source Code Analysis Base
# Bundles Serena Go LSP analysis + standard bash permissions for Go source navigation.
#
# Usage:
#   imports:
#     - shared/go-source-analysis.md

imports:
  - shared/mcp/serena-go.md
  - shared/reporting.md

tools:
  bash:
    - "find pkg -name '*.go' ! -name '*_test.go' -type f"
    - "find pkg -type f -name '*.go' ! -name '*_test.go'"
    - "find pkg/ -maxdepth 1 -ls"
    - "find pkg/workflow/ -maxdepth 1 -ls"
    - "wc -l pkg/**/*.go"
    - "head -n * pkg/**/*.go"
    - "grep -r 'func ' pkg --include='*.go'"
    - "cat pkg/**/*.go"
---

## Go Source Code Analysis Setup

Serena Go LSP analysis is configured for this workspace. Standard bash tools for Go source navigation are available.

### Bash Navigation Tools

Use these bash tools to supplement Serena's semantic analysis:

- `find pkg -name '*.go' ! -name '*_test.go' -type f` — list all non-test Go source files
- `find pkg/ -maxdepth 1 -ls` / `find pkg/workflow/ -maxdepth 1 -ls` — explore directory structure
- `wc -l pkg/**/*.go` — measure file sizes
- `head -n * pkg/**/*.go` / `cat pkg/**/*.go` — read file contents
- `grep -r 'func ' pkg --include='*.go'` — find all function definitions
