---
"gh-aw": patch
---

Enable CLI fallback for the GitHub MCP server by removing it from the `INTERNAL_SERVERS` exclusion list in `mount_mcp_as_cli.cjs`. When the native MCP HTTP `initialize` handshake fails (e.g. protocol-version mismatch between the agent and gateway), the CLI bridge now mounts `github` tools via a shell wrapper — the same recovery path `safeoutputs` already uses — so agents retain access to GitHub MCP tools in every session.
