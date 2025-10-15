---
"gh-aw": minor
---

Add secret validation steps to agentic engines (Claude, Copilot, Codex)

Added secret validation steps to all agentic engines to fail early with helpful error messages when required API secrets are missing. This includes new helper functions `GenerateSecretValidationStep()` and `GenerateMultiSecretValidationStep()` for single and multi-secret validation with fallback logic.
