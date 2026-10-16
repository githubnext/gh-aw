---
"gh-aw": patch
---

Store GitHub Actions dependency job IDs, results, and observed durations in unified agent sessions. Reuse conclusion job listings for timing, exclude job outputs, and preserve failures, cancellations, and skipped jobs separately from agent execution and detection verdicts.

Duration collection is always enabled with no frontmatter opt-out. Every generated conclusion job now requests at least `actions: read`, including workflows with both failure reporters disabled that previously could omit Actions access; existing required `actions: write` grants are preserved.
