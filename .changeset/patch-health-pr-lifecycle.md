---
"gh-aw": patch
---

Exclude evidence-confirmed PR merge, closure, and head-deletion interruptions from health metrics and investigations without counting them as successes. Retain unrelated cancellations, genuine failures, and runs with unavailable lifecycle evidence; bound and cache lifecycle lookups and expose per-run classifications.
