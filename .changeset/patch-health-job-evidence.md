---
"gh-aw": patch
---

Fix health failure clustering by retaining bounded Jobs API step evidence and reusing identity-checked raw cached jobs across CLI versions without trusting incompatible derived metrics. Expose cache compatibility, evidence completeness, report provenance, representative context, and reconcilable workflow status counts; do not count above-threshold failing revisions as healthy.
