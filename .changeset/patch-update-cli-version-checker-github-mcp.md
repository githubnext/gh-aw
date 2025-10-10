---
"githubnext/gh-aw": patch
---

Update CLI version updater to support GitHub MCP server version monitoring

The CLI version checker workflow now monitors GitHub MCP server updates in both local (Docker image) and remote (hosted API) modes. This includes checking Docker image tags from ghcr.io and tracking the remote API version at api.githubcopilot.com/mcp/.
