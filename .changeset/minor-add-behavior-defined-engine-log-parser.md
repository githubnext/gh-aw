---
"gh-aw": minor
---

Add `log-parser` behavior field for declarative engines (goose, opencode, crush, aider). Authors define a `parseLog(logContent)` function in YAML and the compiler wraps it with the `createEngineLogParser` bootstrap, producing normalized agent event data used by the standard rendering helpers. The write step runs with `if: always()` so logs are captured even when an earlier step fails.
