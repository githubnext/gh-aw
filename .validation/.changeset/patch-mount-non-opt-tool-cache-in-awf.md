---
"gh-aw": patch
---

Fixed AWF chroot tool-cache mounting so runners that use `RUNNER_TOOL_CACHE` or the legacy `_tool` path can still find Node during startup.
