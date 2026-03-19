---
"gh-aw": patch
---

Fix pipeline failure handling when Copilot CLI exits non-zero after successfully producing safe outputs.

The Copilot execution step now continues on error so downstream output collection and inference checks still run, and the conclusion reporting now distinguishes "agent produced outputs but returned a non-zero exit code" from missing-safe-outputs failures.
