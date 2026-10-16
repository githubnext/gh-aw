---
"gh-aw": patch
---

Restore daily AIC scan snapshots incrementally after workflow-history lookup returns 404. Inspect trusted producers before fetching older repository-history pages and stop at the first usable snapshot, avoiding unnecessary API reads and quota failures after a usable cache is already available. Preserve producer validation, bounded searches, explicit API errors, and complete authoritative spending checks.
