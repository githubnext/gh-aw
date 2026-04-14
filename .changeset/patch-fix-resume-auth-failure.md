---
"gh-aw": patch
---

Fix copilot-driver `--resume` authentication failures: detect "No authentication information found" as non-retryable, add GITHUB_TOKEN/GH_TOKEN fallback for COPILOT_GITHUB_TOKEN, and log auth token availability for diagnostics.
