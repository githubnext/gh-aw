## Trend Data (2026-08-16)

Baseline was 2026-08-14; this cycle's deltas:

- **Fleet reliability**: 96.93% raw / 97.26% excl. intentional-failures (293 runs) — sharp jump from 79.5-82.1%. Caveat: audit-workflows itself resumed after a 40-day gap, so partly reflects catch-up sampling, not a single fix. Corroborated by agent-job-health (0/15 failures) and #52518 closure.
- **#52518 (PR-review infra flakiness)**: CLOSED, as predicted last cycle. Retire from watchlist.
- **Design Decision Gate**: invocation-cap root cause fixed (#52836 merged); SECOND distinct root cause (pr_number/workflow_dispatch hard-fail) found still broken despite #52987 closed-without-fix. Re-filed.
- **Firewall hostname fix**: not re-checked this cycle (was slated for retirement after 3 clean cycles last time) — resume spot-check if it regresses.
- **`agenticworkflows logs` timeout**: not re-tested (used only `--count 15`, succeeded). Re-verify ~40-run ceiling next cycle.
- **Sentrux god_files_ceiling**: now confirmed enforcing (gap from 08-13 resolved). New regression found instead: `api.sentrux.dev` missing from network.allowed (3rd fix attempt after #43655/#40546 closed).
- **Monitoring-staleness meta-theme (3 agents, flagged 08-14)**: resolved via catch-up runs this cycle, not recurring. Closed as a watch item.
- **Cross-engine 0-turn crash**: signature spread from Copilot-only to Aider (first) and Crush (2 first) this window — small sample, filed as investigation.
- **Issue creation**: 7/7 filed, all dedup-checked via `gh api search/issues`. 2 were near-misses caught by direct source verification (1 false-positive not filed, 1 chronic-duplicate not re-filed).

See `last_analysis_timestamp.md` for full narrative detail on each item above.

Next cycle checks: (a) did the 2 new Design Decision Gate issues get real fixes this time, (b) does sentrux firewall regress a 4th time, (c) does Aider/Crush crash signature grow or stay noise, (d) re-confirm firewall-hostname and logs-timeout items which went unchecked this cycle.
