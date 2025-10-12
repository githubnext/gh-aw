---
"gh-aw": minor
---

Add --json flag to status command and jq filtering to MCP server

Adds new command-line flags to the status command:
- `--json` flag renders the entire output as JSON
- Optional `jq` parameter allows filtering JSON output through jq tool

The jq filtering functionality has been refactored into dedicated files (jq.go) with comprehensive test coverage.
