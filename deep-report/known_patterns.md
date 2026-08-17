## DeepReport Memory (2026-08-17T00:26:00Z)

### Standing practice (reconfirmed again this cycle)
"Closed status / a passing report is not evidence a detection or fix actually works — spot-check source directly." This cycle it caught a much more concrete bug than usual: the Daily Cache Strategy Analyzer's core jq detection filter checks `.tools.cache_memory`, a field that `actions/setup/js/generate_aw_info.cjs` (verified directly) never writes into `aw_info.json` at all. That means the analyzer has been structurally incapable of finding any cache-memory-enabled workflow via that snippet — explaining its "0 workflows" report despite 101/285 lockfiles (35% of fleet) using the feature. Filed with a concrete two-option fix. Same discipline applied (more cautiously) to the observability report's MCP `type`-field claim — cross-referenced against the real parser code, but filed as an explicit "needs verification" investigation rather than a confident bug, since it could equally be the report's detection that's wrong.

### Avenger has (at least) two independent chronic failure modes — track separately
(1) Bun runtime segfault, `#51984` (P0, open) — crash with varying memory address, new recurrence via #53238 (2026-08-16) commented as duplicate. (2) `err-config-no-structured-logs` driver_exit, `Turns=0`/`ErrorCount=0`, no crash — 19 recurrences since ~06-08, 4 prior closures (#44303, #40145, #41885, #39141) that didn't hold, filed fresh this cycle. Do not conflate a fix to one with progress on the other in future cycles.

### New meta-category: monitoring-the-monitors
`audit-workflows` (the fleet's own reliability monitor) silently missed its own schedule for 41 days (2026-07-06 → 2026-08-16), and nothing else caught it. This is a distinct failure class from "an agent's repo-memory went stale" (resolved last cycle) — it's about a *schedule/trigger* silently not firing at all, with zero alerting. Filed fresh, with the audit report's own recommendation (a lightweight heartbeat check across schedule-triggered "daily"/monitor workflows) as the suggested fix shape. Watch whether this actually gets built, or recurs a 2nd time, next cycle.

### Prompt-writing guidance: multi-week trend, not a one-off signal
Copilot PR prompt-analysis has shown the same "concise + concrete prompts merge more" pattern consistently from 2026-07-01 through 2026-08-16 (6+ weeks). Filed this cycle as a quick docs task. Worth checking next cycle whether it landed and whether the declining merge-rate trend (81.2% → 76.7% over that window) responds at all.

### Lesson reinforced: verify report claims against source in BOTH directions
This cycle: (a) cache-strategy claim ("0 cache-memory workflows") verified TRUE and root-caused via source (a real bug); (b) MCP `type`-field claim verified as *plausible but unconfirmed* — didn't have a raw sample to check, so filed hedged rather than either dismissing or overclaiming. Neither extreme (blind trust, blind dismissal) served correctly here — the middle path (cite what was checked, flag what wasn't) is the right pattern when full verification isn't feasible in one cycle.

### Resolved / no update needed this cycle (from 2026-08-16 cycle, still holding)
- #52518 shared PR-review infra flakiness — remains closed, not re-checked this cycle (no new signal).
- Sentrux god_files_ceiling — not covered in this cycle's 14-discussion window (no sentrux report appeared); nothing new to report.
- Design Decision Gate pr_number bug (filed 2026-08-16) — not re-verified this cycle (no Design Decision Gate report appeared in the window); re-check next cycle whether it got a real fix.

### Chronic pattern, still deliberately not re-filed
Docs "jargon before first use" complaint — did not resurface in this cycle's 14-discussion window at all, so nothing new either way. Standing note carried forward: if it recurs, consider recommending a different escalation path (direct maintainer ping) rather than a 17th agent-filed duplicate.

### Full findings-file cross-reference
See `extracted-tasks.md` for the 5 issues + 2 comments filed this cycle and not-filed rationale; `flagged_items.md` for next-cycle watch items; `trend_data.md` for quantitative deltas; this file's narrative above for full detail per finding.
