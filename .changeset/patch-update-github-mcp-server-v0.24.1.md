---
"gh-aw": patch
---

Update GitHub MCP Server from v0.24.0 to v0.24.1. This updates the default MCP server
version and recompiles workflow lock files to pick up the bugfix that includes empty
properties in the `get_me` schema for OpenAI compatibility.

Changes:
- Updated `pkg/constants/constants.go` to set `DefaultGitHubMCPServerVersion` to `v0.24.1`.
- Recompiled workflow lock files to use `v0.24.1`.

Fixes githubnext/gh-aw#5877

