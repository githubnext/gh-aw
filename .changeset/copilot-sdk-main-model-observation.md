---
"gh-aw": patch
---

Copilot SDK mode now observes the main session's model and records `observed` or `substituted` in `copilot-model-selection.json`, as CLI mode does. A silent main-model substitution stops the run with the existing refusal diagnostic and is not retried.
