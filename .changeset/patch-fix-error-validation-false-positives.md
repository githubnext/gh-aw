---
"gh-aw": patch
---

Fix false positives in error validation from environment variable dumps in logs

The audit workflow was failing due to false positives in error pattern matching. The error validation script was matching error pattern definitions that appeared in GitHub Actions logs as environment variable dumps, creating a recursive false positive issue. Added a `shouldSkipLine()` function that filters out GitHub Actions metadata lines (environment variable declarations and section headers) before validation, allowing the audit workflow to successfully parse agent logs without false positives.
