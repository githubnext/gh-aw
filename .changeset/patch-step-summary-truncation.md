---
"gh-aw": patch
---

Fix step summary truncation by forwarding `GITHUB_STEP_SUMMARY` into the sandbox, raising the agent text limit to 2000 characters, and showing an explicit truncation notice.
