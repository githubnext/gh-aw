---
"gh-aw": patch
---

Refactor safe output type validation into a data-driven validator engine.

Moves validation logic into `safe_output_type_validator.cjs`, generates
validation configuration from Go as a single source of truth, and updates
the JavaScript collector to use the new validator. Adds tests and keeps
the generated `validation.json` filtered and indented to reduce merge
conflicts.

