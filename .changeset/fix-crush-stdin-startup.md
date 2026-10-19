---
"gh-aw": patch
---

Fix the unsupported Crush sample stalling before initialization when sandbox stdin is an open pipe. Supply the existing prompt file as finite stdin, keeping prompt contents out of process arguments. Preserve the pinned CLI, provider/model routing, and execution limits. Add open-pipe regressions and opt-in offline validation of the real CLI, native MCP tool calls, and provider errors.
