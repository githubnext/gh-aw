---
"gh-aw": patch
---

Fix generated lock files to avoid secret-scanner false positives by routing the dummy Copilot API key through `COPILOT_DUMMY_BYOK` indirection instead of emitting the literal token-shaped value inline.
