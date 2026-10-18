---
"gh-aw": patch
---

Migrate the OpenCode sample to the pinned `@opencode/cli@2.0.26` CLI with standalone CI execution, native v2 provider and MCP configuration, separate startup/tool timeouts, and v2 tool identity and exit-status parsing. Recover missing step and usage metadata through bounded session exports and fail explicitly when the native step limit is reached. Preserve v1 session artifact support and document v1 plugin and LSP migration limitations.
