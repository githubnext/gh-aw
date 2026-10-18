---
"gh-aw": patch
---

Migrate the OpenCode sample to the pinned `@opencode/cli@2.0.26` CLI with standalone CI execution, native v2 provider and MCP configuration, separate startup/tool timeouts, and v2 tool identity and exit-status parsing. Recover and reconcile step, compaction, and reasoning accounting through bounded session exports without duplicating assistant turns, and fail explicitly when the native step limit is reached. Respect the configured working directory and require explicit MCP failure status. Preserve v1 session artifact support and document v1 plugin and LSP migration limitations.
