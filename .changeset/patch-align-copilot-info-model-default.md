---
"gh-aw": patch
---

Aligned `GH_AW_INFO_MODEL` fallback for the Copilot engine with the `COPILOT_MODEL` fallback. Both now use `'claude-sonnet-4.6'` (matching `CopilotBYOKDefaultModel`) so that the model recorded in run metadata agrees with the model actually used by the Copilot CLI.
