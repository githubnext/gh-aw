---
"gh-aw": patch
---

Add `allowed-github-references` safe-output field to restrict and escape unauthorized GitHub-style markdown references (e.g. `#123`, `owner/repo#456`). Includes backend parsing, JS sanitizer, schema validation, and tests.

