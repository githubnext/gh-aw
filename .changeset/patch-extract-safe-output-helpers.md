---
"gh-aw": patch
---

Extract duplicate safe-output validation logic into shared helpers

Extracted ~120 lines of identical validation logic from `add_labels.cjs` and `add_reviewer.cjs` into a new `safe_output_helpers.cjs` module. The new module provides three reusable functions: `parseAllowedItems()`, `parseMaxCount()`, and `resolveTarget()`, reducing code duplication and improving maintainability.
