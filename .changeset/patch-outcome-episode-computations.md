---
"gh-aw": patch
---

Make `gh aw outcomes` the sole outcome-evaluation entry point. Remove the scheduled collector and duplicate JavaScript evaluators, stop evaluating live outcomes during audit, and have the impact report consume CLI classifications. Preserve dispatched run IDs and use the linked agent PR's repository for assignment outcomes.

Keep ambiguous legacy workflow calls in separate episodes and make conflicting lineage deterministic. Populate AIC per accepted outcome in per-run outcomes summaries when usage is available.

Expose parent repository, run, attempt, workflow, call identity, and event type from `aw_context` in unified session episode metadata. Preserve the current run's workflow reference, attempt, and legacy lineage inputs separately.
