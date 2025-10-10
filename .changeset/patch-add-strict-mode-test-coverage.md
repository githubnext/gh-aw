---
"gh-aw": patch
---

Add test coverage for shorthand write permissions in strict mode

Added comprehensive test cases to verify that shorthand write permissions (`permissions: write` and `permissions: write-all`) are correctly rejected in strict mode, while read-only permissions (`permissions: read-all`) are allowed. Also added test coverage for inline comments in YAML to ensure they don't bypass strict mode validation.
