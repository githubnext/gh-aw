---
"gh-aw": patch
---

Replace the `awmg` MCP gateway container with `gh-aw-mcpg` and add supporting
configuration.

- Replaces the `awmg` gateway container with `gh-aw-mcpg`.
- Adds gateway constants, types, and configuration extraction.
- Adds `--enable-host-access` AWF flag to allow gateway access when needed.
- Updates the health check script to target `gh-aw-mcpg`.

This is an internal/tooling change and does not change the public CLI API.

