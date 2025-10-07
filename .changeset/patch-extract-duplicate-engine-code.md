---
"gh-aw": patch
---

Refactor: Extract duplicate code patterns from engine implementations

This refactoring addresses approximately 137 lines of duplicated code across engine implementations (Claude, Codex, Copilot, Custom) by extracting common patterns into shared helper functions in `agentic_engine.go`. The changes include:

- Custom step handling logic consolidated into `ProcessCustomSteps()`
- Environment variable setup helpers for safe-outputs and max-turns configuration
- Separate helper functions for map-based engines (Codex, Copilot), line-based engines (Claude), and custom engine (`map[string]any`)

This refactoring improves maintainability by establishing a single source of truth for common functionality, reducing bug risk from copy-paste errors, and making future updates more consistent across all engine types.
