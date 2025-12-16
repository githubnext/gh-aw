---
"gh-aw": patch
---

Update the GitHub MCP Server Docker image to `v0.25.0`.

- Bump `DefaultGitHubMCPServerVersion` to `v0.25.0` in `pkg/constants/constants.go`.
- Recompiled workflow `.lock.yml` files to reference the new image version.
- Updated tests and action pins that referenced the previous version.

This is a non-breaking patch release that updates the MCP server image and related test/lockfile references.

