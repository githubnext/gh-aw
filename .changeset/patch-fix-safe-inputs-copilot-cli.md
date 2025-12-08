---
"gh-aw": patch
---

Fix safe-inputs type for Copilot CLI and server startup error

- When `mode: stdio` is used with the Copilot engine, the generated MCP
  configuration now uses `"type": "local"` so the Copilot CLI can start
  the safe-inputs server correctly.
- Fix the generated JavaScript entrypoint to avoid calling `.catch()` on an
  undefined return value by using a `try/catch` pattern instead.
- Updated tests to assert Copilot uses `local` and Claude continues to use
  `stdio`.

