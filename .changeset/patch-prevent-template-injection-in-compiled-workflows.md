---
"gh-aw": patch
---

Fixed a template injection vulnerability in compiled workflows by moving `${{ }}` expressions out of `run:` blocks into step `env:` variables for safe outputs config, guard policy values, and OTEL endpoint rendering.
