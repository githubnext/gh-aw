---
"gh-aw": minor
---

Add top-level `model` field that replaces and deprecates `engine.model`.

- New top-level `model` field for specifying the LLM model used by the agentic engine. Takes precedence over `engine.model` when both are set.
- Deprecation warning emitted at compile time when `engine.model` is used.
- `gh aw fix` codemod (`engine-model-to-top-level`) automatically migrates `engine.model` to top-level `model`.
- Updated main JSON schema: top-level `model` added, `engine.model` marked as deprecated.
