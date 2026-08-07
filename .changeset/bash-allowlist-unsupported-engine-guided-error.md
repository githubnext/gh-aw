---
"gh-aw": patch
---

`gh aw fix` now reports a guided error when `tools.bash` declares a restriction (a specific command list, an empty list, or `bash: false`) while the workflow uses an engine that ignores bash command allow-listing, such as `codex`.

Previously this incompatibility was only surfaced by `gh aw compile --strict`, and `gh aw fix` reported "No fixes needed". The new `bash-allowlist-unsupported-engine-guided-error` codemod does not rewrite the workflow automatically because both remediations change semantics: widening the list to `bash: ["*"]` makes the unrestricted access explicit, and switching to `copilot`, `claude`, or `gemini` changes which agent runs the workflow.
