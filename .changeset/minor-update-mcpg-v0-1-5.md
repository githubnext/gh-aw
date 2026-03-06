---
"gh-aw": major
---

Bump the MCP Gateway dependency to v0.1.5 and sync validation with the new
breaking rules (Docker-only TOML stdio servers, explicit mount modes, and no
mounts on HTTP MCP servers).

**⚠️ Breaking Change**: MCP Gateway v0.1.5 introduces stricter validation rules that reject configurations that were previously valid.

**Migration guide:**
- **stdio MCP servers** must now use Docker-only TOML configuration; non-Docker stdio servers are no longer supported
- **Mount configurations** must now specify an explicit mount mode; implicit mount modes are no longer accepted
- **HTTP MCP servers** must not include any mount configurations; remove any `mounts:` sections from HTTP MCP server definitions
- Example migration for stdio server:
  ```yaml
  # Before (non-Docker stdio)
  mcp:
    servers:
      my-server:
        type: stdio
        command: my-server-binary

  # After (Docker-only stdio)
  mcp:
    servers:
      my-server:
        type: stdio
        image: my-server-image:latest
  ```
- Validate your MCP server configurations against the new rules before upgrading
