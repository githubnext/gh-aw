---
"gh-aw": patch
---

Refactor safe output handlers to the handler factory pattern.

All safe output handlers were updated to export `main(config)` which returns a
message handler function. This is an internal refactor to improve handler
composition, state management, and testability. No user-facing CLI behavior
changes are expected.

