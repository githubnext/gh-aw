---
"gh-aw": patch
---

`gh aw fix` now reports a guided error when a workflow's `post-steps` (directly or via `imports`) accesses the safe outputs file directly (such as `$GH_AW_SAFE_OUTPUTS` or `/tmp/gh-aw/agent_output.json`).

Previously, this incompatibility was only surfaced during `gh aw compile --strict` and `gh aw fix` reported "No fixes needed". The new `post-steps-safe-outputs-guided-error` codemod flags the step with manual fix instructions and documentation reference without altering the file automatically.
