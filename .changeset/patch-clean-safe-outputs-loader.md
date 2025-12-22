---
"gh-aw": patch
---

Clean and modernize `pkg/workflow/js/safe_outputs_tools_loader.cjs` by refactoring
internal functions (`loadTools`, `attachHandlers`, `registerDynamicTools`) to use
modern JavaScript patterns (optional chaining, nullish coalescing, handler map)
and reduce nesting and complexity. No behavioral changes.

