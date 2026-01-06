---
"gh-aw": patch
---

Add `allowed-github-references` safe-output configuration to restrict which
GitHub-style markdown references (e.g. `#123` or `owner/repo#456`) are
allowed when rendering safe outputs. Unauthorized references are escaped with
backticks. This change adds backend parsing, a JS sanitizer, schema
validation, and comprehensive tests.

