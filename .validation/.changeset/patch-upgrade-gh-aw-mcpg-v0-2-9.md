---
"gh-aw": patch
---

Upgrade `gh-aw-mcpg` to `v0.2.9`, updating the default MCP gateway image version and recompiling workflow lock files. This includes security hardening (`pin_issue`/`unpin_issue` classified as write ops, `transfer_repository` unconditionally blocked), `copilot-swe-agent` recognized as trusted first-party bot, and bug fixes for duplicate log entries.
