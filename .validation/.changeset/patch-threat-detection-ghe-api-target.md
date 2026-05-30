---
"gh-aw": patch
---

Propagate `engine.api-target` to the threat detection AWF invocation so that on GHE Cloud with data residency the threat detection run receives the same `--copilot-api-target` flag and GHE-specific domains in `--allow-domains` as the main agent run.
