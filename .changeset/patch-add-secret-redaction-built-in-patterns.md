---
"gh-aw": patch
---

Add built-in pattern detection and extensive tests for secret redaction in compiled logs.

This change adds built-in regex patterns for common credential types (GitHub, Azure, Google, AWS, OpenAI, Anthropic) to `redact_secrets.cjs` and includes comprehensive tests covering these patterns and combinations with custom secrets.

