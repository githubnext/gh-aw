---
"gh-aw": patch
---

Fix safe-inputs MCP config for Copilot CLI: convert `type: stdio` to `type: local` when generating Copilot fields; fix server startup JS to avoid calling `.catch()` on undefined; update tests to assert behavior for Copilot and Claude.

This is a non-breaking bugfix that ensures the Copilot CLI receives a compatible MCP `type` and that the generated server entrypoint handles errors correctly.

