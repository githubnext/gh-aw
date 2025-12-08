---
"gh-aw": patch
---

Fix environment variable expansion for the safeinputs MCP server by pre-assigning
port `3002` and updating related configuration, scripts, and tests.

This is an internal bugfix affecting MCP server setup for the Copilot engine and
does not change public APIs.

