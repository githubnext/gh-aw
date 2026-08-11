## Trend Data (2026-08-11)

Baseline established 2026-08-10 (see prior cycle); this cycle's deltas:

- **Fleet agent-job reliability**: 49.0% failure rate (103/210 runs, 24h window ending 2026-08-11 04:23 UTC, discussion #51935) — up sharply from the 10% figure in a 50-run spot sample cited 2026-08-10. Not yet reconciled whether this is a real regression or a sampling/known-chronic-workflow artifact (investigation issue filed).
- **Safe-outputs job health**: 100% success among executed jobs (197/197), 6.2% clean skip rate, 0% failure — consistent with/slightly better than 2026-08-10's 98.4%-of-attempted figure (different window/sample, same clean bill of health).
- **Firewall** (7d window ending 2026-08-10, discussion #51835): 100 firewall-enabled runs, 4,514 requests, 3.32% block rate (150 blocked) — comparable in shape to 2026-08-10's 91-run/5,775-request/0.7%-blocked figure from a narrower window; both point to `proxy.golang.org` and Copilot API domains as the dominant blocked destinations.
- **Issues** (500-sample, this cycle via issues-analyst subagent): 141 open / 359 closed — up slightly from 2026-08-10's 134 open / 366 closed in the same 500-sample methodology. Daily Issues Report's larger 1000-sample (#52081) shows 165 open / 835 closed, 83.5% closure, ~13h avg close time — roughly flat vs. 2026-08-10's 147 open / 853 closed, 13h55m.
- **Copilot session transcripts**: still 0/50 files this cycle (#51985) — chronic gap now described as ~4.5 months old by the source agent (longer than "44+ days" stated 2026-08-10 — inconsistent counting, flagged not re-filed).
- **Agent Performance Report** (#52052, partial 24h collection, 66 workflows in snapshot): top performers Test Quality Sentinel (83%), Matt Pocock Skills Reviewer (78%), Ponytail Reviewer (71%); worst performers PR Sous Chef (16%), Issue Monster (20%), several chronic 0%-success workflows (Code Scanning Fixer, PR Triage Agent, Auto-Triage Issues, ESLint Monster) — same names as prior cycles, no sign of remediation yet.
- **agenticworkflows logs tool**: timed out both attempts this cycle (see flagged_items.md) — could not independently verify raw run counts/token totals; relied on discussion-report secondary sourcing instead. Going forward: if this becomes a persistent pattern, worth its own issue.

Going forward: use 2026-08-11 as the updated trend baseline. Next cycle should diff against these numbers, especially the 49% failure-rate figure (is it trending down after the investigation issue lands?).
