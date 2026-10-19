---
"gh-aw": major
---

Remove legacy episode reconstruction and ambiguous root aliases from logs reporting.

**Breaking change:** Episode rollups require a recorded nonempty `context.episode_id`.
Call-ID grouping, synthetic standalone IDs, revision/actor-based inference,
`workflow_run` inference, and bridges through uncorrelated runs are removed.
Runs without canonical identity remain in `.runs[]` and overall totals but no
longer contribute to episode counts, rollups, or edges.

**Migration:** Replace episode-level `root_run_id` and `primary_workflow` with
`representative_run_id` and `representative_workflow` for selected-data identity.
Use `recorded_roots[]` for canonical root claims. Recompile workflows to record
canonical context for future runs; historical runs without it support per-run
analysis only. Compact JSON now retains all canonical episodes, including
single-run episodes. Coverage remains `unknown` or `partial`, never complete.
