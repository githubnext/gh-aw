---
"gh-aw": patch
---

Update Pi to v1.3.0, support its full thinking range in inline sub-agents, preserve installed tool selections, and align the engine catalog with native MCP support. Remove Pi-specific MCP exclusions from safe-output guidance and select CLI-only instructions by configured transport across engines. Exercise the real CLI, SDK, and RPC drivers against local inference and MCP fixtures and document the headless release feature boundary.

Keep unified Pi sessions aligned with v1.3.0: preserve reasoning-output accounting and served-model identity, and surface RPC extension failures as session errors.

Merge installed tool defaults with managed codemode and tool-search selections, and reject `max` effort before agent launch when installed Pi is older than v1.3.0.
