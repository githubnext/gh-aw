---
"gh-aw": patch
---

Remove detection of missing tools using error patterns

Removed fragile error pattern matching logic that attempted to detect missing tools from log parsing infrastructure. This detection is now the exclusive responsibility of coding agents. Cleaned up 569 lines of code across Claude, Copilot, and Codex engine implementations while maintaining all error pattern functionality for legitimate use cases (counting and categorizing errors/warnings).
