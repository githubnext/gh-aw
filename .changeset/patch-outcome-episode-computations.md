---
"gh-aw": patch
---

Reevaluate pending, unknown, and failed outcome checks from durable cached manifests instead of permanently marking them seen. Preserve dispatched run IDs and use the linked agent PR's repository for assignment outcomes.

Keep ambiguous legacy workflow calls in separate episodes and make conflicting lineage deterministic. Populate AIC per accepted outcome in per-run CLI and audit summaries when usage is available.

Expose parent repository, run, attempt, workflow, call identity, and event type from `aw_context` in unified session episode metadata. Preserve the current run's workflow reference, attempt, and legacy lineage inputs separately.
