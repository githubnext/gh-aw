---
imports:
- shared/mcp/serena-go.md
- shared/reporting.md
tools:
  bash:
  - find pkg -name "*.go" ! -name "*_test.go" -type f
  - find pkg -type f -name "*.go" ! -name "*_test.go"
  - find pkg/ -maxdepth 1 -ls
  - find pkg/workflow/ -maxdepth 1 -ls
  - wc -l pkg/**/*.go
  - head -n * pkg/**/*.go
  - grep -r "func " pkg --include="*.go"
  - cat pkg/**/*.go
---
## Go Source Code Analysis Setup

Serena Go LSP analysis is configured for this workspace. Standard bash tools for Go source navigation are available.

Use Serena semantic tools first; use the listed bash commands only for quick file discovery, sizing, and raw content checks.