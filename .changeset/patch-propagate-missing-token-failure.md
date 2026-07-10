---
"gh-aw": patch
---

Propagate missing token failure to the conclusion job failure issue. When a required secret (e.g., `COPILOT_GITHUB_TOKEN`) is missing, the activation job now fails with `secret_verification_result=failed`, causing the conclusion job to run and create an actionable failure issue.
