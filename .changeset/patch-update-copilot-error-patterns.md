---
"gh-aw": patch
---

Update error patterns for copilot agentic engine

Added 8 new error pattern categories to improve error detection in the Copilot agentic engine:
- Rate limiting errors (HTTP 429, quota exceeded)
- Timeout and deadline errors
- Network connection and DNS errors
- Token expiration errors
- Memory and resource exhaustion errors

All patterns are carefully designed to catch real production errors while avoiding false positives through appropriate error context requirements.
