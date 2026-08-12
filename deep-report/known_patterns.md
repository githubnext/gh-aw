## DeepReport Memory (2026-08-12T15:00:00Z)

### Meta: path/persistence fix holding, and a verified real fix
Repo-memory files at `deep-report/` (one level deep) were present and readable at cycle start — the 2026-08-10 workaround holds two cycles later. Separately, this cycle directly verified in the repo that `strict:` mode docs (#52086, filed+closed 2026-08-11) were actually fixed — `frontmatter.md` now correctly states `strict:true` (default) is the stronger security mode. This is the first cycle with concrete evidence of a self-filed doc/reliability fix landing for real, rather than closing via TTL expiry with no comment (contrast with #51807's tracked chronic-closure pattern and the new `logs`-timeout recurrence below).

### `agenticworkflows logs` timeout is now a confirmed 2-bug-instance chronic pattern, not a fluke
- 2026-08-11 cycle: `logs` timed out twice (`count:100`, `count:50`+`timeout:120`), unrelated open issue #51952 already existed for an `engine`-filter variant of the same symptom.
- 2026-08-11, later: #51952 auto-closed with zero comments (TTL expiry per `gh-aw-expires`), not a merged fix.
- 2026-08-12 (this cycle): reproduced live — `{"count":100,"start_date":"2026-08-05"}` timed out at ~60s; `{"count":30}` succeeded in ~37s. Re-filed with fresh evidence, citing #51952 explicitly as an unresolved prior report of the same class.
- Takeaway: the underlying root cause (timeout budget too aggressive for filtered/larger `logs` queries) has never actually been fixed — every "closure" so far has been TTL expiry. Watch next cycle whether the new filing gets a real code fix (PR reference) vs. silently re-expiring.

### New concrete/actionable findings this cycle (all filed as issues, dedup-checked against 126 open issues first)
1. `coverage.findProfile` (`pkg/linters/internal/coverage/coverage.go:61`) path-matching bug makes the ADR-51573 perf-linter coverage gate a silent no-op on every standard checkout (module-qualified profile keys vs. OS-absolute analysis paths never match via suffix check) — found by Sergo (#52232) but never filed due to a quota-exhaustion incident (3 placeholder `create_issue` calls burned the run's quota); filed now.
2. `PR Code Quality Reviewer` workflow hits the wrong hostname `api.individual.githubcopilot.com` (should be `api.githubcopilot.com`), causing 20-32 firewall blocks/day recurring across ≥2 separate daily reports (#52117, #52213).
3. 8 scheduled audit/report/detector-named workflows missing `gh-aw-detection: true` — verified directly against `.github/workflows/*.md`, all 8 confirmed missing (#52181).
4. Schema-consistency tooling inspects stale `pkg/parser/frontmatter.go` (now just a logger stub) instead of `pkg/workflow/frontmatter_types.go`, producing false negatives; `frontmatter-full.md` generated reference is also missing `run-install-scripts` and `report-failed-jobs` as searchable fields (#52243).
5. `GitHubToken` field-shadowing bug in `create_project.go:10`, `update_project.go:27`, `create_project_status_update.go:12` — hand-declared field shadows the embedded `BaseSafeOutputConfig.GitHubToken`, leaving the embedded copy parsed-but-unread (latent divergence risk) — verified directly in source (typist #52283).
6. `agenticworkflows logs` MCP timeout recurrence (see above) — re-filed citing auto-expired #51952.
7. AI issue triage guide's example workflow uses non-default labels (`priority/p0-p2`, `needs-info`) in `add-labels.allowed` without noting they must pre-exist, causing 404s for new users following the tutorial verbatim (#52088).

### Chronic lineages — status check this cycle (not re-filed as new duplicates)
- Copilot Session Insights conversation-transcript gap: now described as a NEW 35-day sampling gap (prior sample 2026-07-08) on top of the original transcript gap since 2026-06-23 (#52255) — still chronic, folding into existing process-gate evidence.
- Fleet-wide 49% agent-job failure rate (investigation filed 2026-08-11, #52094-adjacent): team-evolution report (#52145) this cycle suggests it's dominated by already-known-broken workflows (PR Sous Chef, Linter Miner) rather than a new regression — plausible explanation surfacing, but no formal reconciliation/closure yet. Watch next cycle.
- LintMonster (#52211) self-filed #52205/#52206/#52207 (function-length backlog, 656 findings) and closed a stale duplicate (#50982) — not duplicated here.
- Cache Strategy (#52136) self-filed a Linter Miner cache-miss fix — matches previously-tracked #52134, not duplicated.
- ESLint Refiner (#52242) self-filed 2 new issues (reassignment-guard bug, chained-method-call detection gap in interpolation rules) — matches open #52240/#52241, not duplicated.
- Issue Arborist (#52236) created no new parent issues this cycle, only linked existing sub-issues — no gap.
- UK AI Resilience (#52104) explicitly filed no new issues (all findings already tracked or false positives) — no gap.

### Fleet health this cycle (small/partial samples — see flagged_items.md for `logs` tool limitation)
- `logs` MCP tool: only a 30-run mixed-window sample was obtainable (spanning 2026-08-06 to 2026-08-12, not a clean 24h slice) — 8/30 failures (26.7%), not representative of true fleet-wide rate; treat as directional only, not a trend datapoint.
- Sentrux code-quality baseline established this cycle at 5238/10000 (right at the min-5200 quality floor) — first-ever run, becomes the new baseline for future trend comparison (#52185).
- Repository Quality report (first-ever run, #52298): 35 files exceed the 800-line guideline; top 3 are `pkg/cli/update_actions.go` (1,144 — already tracked as #52054), `pkg/workflow/compiler_custom_jobs.go` (1,142, not yet tracked as a split task), `pkg/cli/audit.go` (1,095).
- Agentic Workflow Audit resumed after a 36-day gap (last run 2026-07-06 → 2026-08-11); fleet health 88.8% raw / 89.1% ex-guardrail, roughly in line with pre-gap trend (#52152).
