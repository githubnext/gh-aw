---
"gh-aw": patch
---

Hardcode the safeinputs MCP server to port 52000 and remove API key authentication.

This fixes environment variable expansion for the safeinputs MCP server used by the Copilot engine by
pre-assigning a stable port and simplifying local development (no API key required).

Changes:
- Hardcoded port 52000 in configuration and scripts
- Removed API key generation and Authorization headers
- Updated tests to use the new port

Signed-off-by: Changeset Generator

