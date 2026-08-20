## Trend Data (2026-08-20, ~05:45Z cycle)

Window since 00:25Z baseline (#54107), 10 new discussions, all read in full.

- **Issue activity**: 5 new issues filed (strict-mode default gap, redirect/frontmatter docs bundle, schema-diff extractor fix, docs-noob docs bundle, compiler_safe_outputs_job.go godoc gap) + 0 comments. 0 duplicates — most other candidate findings this cycle were already self-filed by their source workflows (Workflow Skill Extractor ×3, Sergo, ESLint Refiner, LintMonster, MCP-auth-test), confirmed via `gh api search/issues` dedup.
- **Issue backlog** (issues-analyst): 217 open / 283 closed. Unlabeled: 2 (down sharply from 10 two cycles ago — confirms that was triage lag). 0 open >7 days.
- **Live 30-run sample** (`start_date: -1d`): 26/30 success (86.7%), 4 failures (Daily VulnHunter Scan, AI Moderator, Daily Container Image Security Scan, Ponytail Reviewer) — no systemic pattern, none matched intentional-failure list.
- **Firewall** (#54123, 24h): 2.07% block rate (362/17,483), still dominated by proxy.golang.org/Code Scanning Fixer (269 blocks) — already tracked, open #54063.
- **Firewall Escape Test** (#54149): SECURE, 11/11 novel techniques blocked.
- **Compiler quality** (#54126): avg 80/100, all 3 files above threshold (up from 77/100 avg 05:45Z 08-19 cycle) — first sign of the 08-19 filed doc/error-wrap gaps improving the score.

Next cycle checks: (a) does the strict-mode CLI/MCP divergence get confirmed as a real bug or a false alarm, (b) do the 5 issues filed this cycle land at the usual fast pace, (c) does the unlabeled-issue count (2) stay low.

## Trend Data (2026-08-20, ~00:25Z cycle)

Window since 17:50Z baseline (#53999@12:34:27Z carried forward), 11 new discussions, #54066 excluded as duplicate re-run.

- **Issue activity**: 2 new issues filed (report window-collapse bug, 3 remaining oversized test files) + 3 comments (#53925, #54009, #53871). 0 duplicates — all dedup-checked via `gh api search/issues`.
- **Issue backlog** (issues-analyst): 205 open / 295 closed. Top labels similar to prior cycles; 3 unlabeled (back down from the 10-count spike last cycle — confirms that was triage lag, not a new backlog trend).
- **Audit-workflows fleet health**: 354 runs sampled, 90.7% raw / 90.9% adjusted success.
- **Detection coverage**: 313/344 (91%) — healthy, consistent with prior baselines.
- **Copilot PR prompt analysis** (30-day, #54079): 78.3% merge rate, declining from ~82% a few cycles ago — conciseness-correlates-with-success finding actioned via comment on #53925.
- **Report window-collapse bug**: reconfirmed 2nd occurrence (#53828 08-18 → #54081 08-20), live example in #54080 (90d target collapsed to ~4d) — now filed as an issue instead of just a recurring caveat.

Next cycle checks: (a) does the unlabeled-issue count stay low (confirming last cycle's spike was lag not trend), (b) does the report-window-collapse issue get picked up, (c) do the 3 remaining oversized test files get split, (d) does the Copilot PR merge-rate decline (82%→78.3%) continue or stabilize.

## Trend Data (2026-08-19, ~17:50Z cycle)

Baseline was 2026-08-19T12:34:27Z (discussion #53999; ~5.3h gap, 11 new discussions since baseline, all read in full).

- **Issue activity**: 4 new issues filed (engine:claude docs gap, proxy.golang.org allowlist, add_package_manifest.go/import_field_extractor.go split, Metrics Collector partial-window). 0 comments. 0 duplicates created — all 4 candidates confirmed unique via `gh api search/issues` dedup checks; several other candidates (bad-redirect-check, GraphQL interpolation, benchmark "regressions") were correctly declined as already-tracked or non-issues.
- **Live 20-run workflow-log sample** (`start_date: -1d`): 19/20 success (95%), engine mix pi/codex/copilot/aider, 1 failure (Linter Miner, agent_logic_failure, already auto-filed #54056).
- **Issue backlog** (issues-analyst, 500-issue window): 211 open / 289 closed. Top labels: agentic-workflows (235), automation (179), cookie (153), code-quality (74), improvement (59). Unlabeled: 10 (up from 3-5 prior cycles, all fresh, 0 open >7d).
- **Firewall** (#54053, ~4.5h capture window): 96.1% allow rate, 205 blocked/5259 total; 89% of blocks from one workflow (Code Scanning Fixer / proxy.golang.org) — addressed via issue filed this cycle. No DIFC integrity-filtered events.
- **Merge velocity** (#54039 Repository Chronicle): 53 PRs merged in 24h, 100 issues opened / 13 closed same window.
- **Repo-memory reliability**: this cycle's own baseline required recovering from a lost prior-cycle write (#54010/#54029, see known_patterns) — first confirmed instance of the patch-size-drop failure mode.

Next cycle checks: (a) does PR #54029 (max-patch-size raise) merge, restoring reliable per-cycle memory writes, (b) do the 4 issues filed this cycle land at the usual fast pace, (c) does the unlabeled-issue count (10) recede or keep growing.

## Trend Data (2026-08-19, ~05:45Z cycle)

Baseline was 2026-08-19T00:15:00Z (~5.5h gap, narrowed scope to 10 new discussions since baseline, excl. own prior briefing #53874).

- **Issue activity this cycle**: 5 new issues filed (safe-outputs.runs-on type mismatch, frontmatter docs bundle, Quick Start auth-accordion, compiler.go error-wrapping/split, compiler-quality-check stale file list). 1 comment added (to #53464, recurring MCP auth-test occurrence). 0 exact-duplicate issues created — 1 candidate (generatedyamlheredoc bug) was found already self-filed as #53901 by the Sergo workflow itself, so skipped.
- **Turnaround/linking confirmed this cycle** (via Issue Arborist Daily Report #53910): all 7 of the prior cycle's filed issues (#53867-#53873) correctly auto-linked as sub-issues of parent #53376 same day — org pipeline healthy, though too early in this short cycle for merge-turnaround data.
- **Live 15-run workflow-log sample** (`start_date: -1d`, engine mix claude/copilot/pi): 13/15 success (86.7%), 3 failures (Daily Container Image Security Scan — already auto-filed #53923, Daily AgentRx Trace Optimizer, Daily Cli Tools Tester) — consistent with prior 3.33% fleet-failure baseline, no systemic signal.
- **Issue backlog** (issues-analyst pass): 186 open / 314 closed. Top labels: agentic-workflows (240), automation (177), cookie (153), code-quality (69), improvement (49). Unlabeled: 3 (#53670, #53489, #53136) — same chronic set as prior cycles. 0 open >7 days.
- **Firewall baseline** (#53891): 0.5% block rate (94/18,820), Google-auth-domain noise pattern confirmed again (Daily Model Inventory Checker, Slide Deck Maintainer).
- **Firewall escape test** (#53906): SECURE, 11/11 novel techniques tested this run, all correctly blocked.
- **Schema consistency check** (#53917, 4 findings): 1 real parser/schema/docs contract bug (safe-outputs.runs-on) + 3 docs gaps — all 4 addressed via 2 issues filed this cycle.
- **Compiler quality baseline** (#53892): 3 files averaged 77/100 (compiler.go 74, compiler_jobs.go 80, compiler_safe_outputs_job.go 76); compiler.go is the only one of the 3 with 0 `%w` error-wrap usages.

Next cycle checks: (a) does the safe-outputs.runs-on parser fix land (highest-priority filing this cycle), (b) do the 2 docs bundles get picked up, (c) does #53464's MCP auth-test issue accumulate enough occurrences to warrant a different remediation approach than "keep commenting", (d) first trend comparison for firewall/detection/observability baselines from the 00:15Z cycle now that a second data point exists.

## Trend Data (2026-08-19, ~00:15Z cycle)

Baseline was 2026-08-18T18:23:34Z (~5.5h gap, narrowed scope to 11 new discussions since baseline).

- **Issue activity this cycle**: 7 new issues filed (squad detection gap, gateway.jsonl/safeoutputs.jsonl telemetry gaps, Daily Status sample-window doc, MCP per-engine tracking, discussions:write permission audit, workflow_dispatch parity, fast-track criteria docs). 1 comment added (to #52253). 0 exact-duplicate candidates found via `gh api search/issues` dedup checks.
- **Turnaround confirmed this cycle** (via Daily Team Evolution Insights #53819): 5 of the prior cycle's flagged items merged same-day — GEO scanner fix (#53800), logs stale-data fix (#53719), pr-triage message fix (#53798), ai-credits blog fix (#53797), and compiler_safe_outputs_job.go decomposition (#53720, 3rd attempt succeeded).
- **Fleet health baseline** (Agent Job Health Monitor #53854, first baseline): 300 runs/80 workflows, 3.33% run-weighted agent-job failure rate, failures concentrated not systemic. 8/10 failures already tracked (#52253, #53191).
- **Detection coverage** (Detection Analysis Report #53851, first baseline): 259/298 sampled runs (86.9%) detection-enabled; only 2 misconfigured workflows found (filed this cycle).
- **Observability baseline** (Daily Observability Report #53859, first sample): 20/20 runs have critical logs present; gap is richness (0/20 `gateway.jsonl`, 17/20 `safeoutputs.jsonl`) — filed this cycle.
- **Lockfile fleet snapshot** (#53820, first baseline): 286 workflows, 38.9MB, 60% copilot/21% claude/19% other engines; 97.6% workflow_dispatch coverage; create_issue (138) more common than create_discussion (91) despite 86% of discussions being "audits" category.
- **Copilot PR prompt analysis** (#53824, 30-day window): 76.3% merge rate; CVE cluster confirmed weak (49.3% vs 79.9%), fix (#53709) just merged, too early to see improvement.
- **Performance baseline** (Daily Performance Summary #53825, first baseline): 5.21h avg PR merge, 10.15h avg issue resolution — healthy.
- **Cross-report consistency** (Daily Regulatory Report #53828): 66 reports reviewed, 98% health score, 1 minor discrepancy (addressed via issue filed this cycle).

Next cycle checks: (a) do the 7 newly-filed issues get picked up at the usual fast pace, (b) does the CVE-prefilter fix start moving the trailing-window success rate up, (c) does #52253 get the 4 additional workflows linked, (d) first trend charts for detection-analysis/agent-job-health/observability now that baselines exist.

## Trend Data (2026-08-18, 12:26Z cycle) — condensed

7 issues filed (logs-tool stale-data bug, transcript "fix that didn't fix it", 2 Typist fixes, container/CVE pre-filter, NLP data-gap, MCP health-struct dedup). Issue backlog 133 open/367 closed, 5 unlabeled. See git history for full text.

## Trend Data (2026-08-18, 06:23Z cycle) — condensed

Baseline 2026-08-18T00:31Z. 4 new issues filed + 1 comment. 20-run live sample: 2 errors (1 driver_exit_failure, 1 agent_logic_failure) out of 20 — both PR Sous Chef's own most-recent runs were "success", consistent with a separate `safe_outputs`-step-only failure mode. 16 open duplicate PR Sous Chef issues discovered as a side effect of investigation.
