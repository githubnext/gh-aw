---
"gh-aw": patch
---

Hoist expression-based safe job environment variables into step `env:` blocks so compiled `run:` scripts no longer inline GitHub Actions expressions and trip the guardrail.
