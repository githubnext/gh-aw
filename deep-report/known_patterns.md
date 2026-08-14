## DeepReport Memory (2026-08-14T15:00:00Z)

### Verified recovery: 3 chronic PR-review agents (Test Quality Sentinel, Matt Pocock, Ponytail Reviewer)
All three, carried forward from the 2026-08-13 "shared PR-review infra flakiness" investigation (#52518), are now at 15/15 (100%) success in fresh samples this cycle, up from 38.9%/54.5%/33.3%. Commented on #52518 with the evidence, recommending closure. This is the 3rd consecutive cycle with a directly-verified real fix/recovery (after the firewall hostname fix on 2026-08-12/13 and strict-mode docs on 2026-08-12) — continuing to pay off the practice of spot-checking carried-forward items against live data each cycle rather than assuming an issue being "open" means the problem is still active.

### New top finding: Design Decision Gate merge-blocking hotspot
Cross-verified by 3 independent monitors this cycle with no prior root-cause issue open:
- `audit-workflows` (#52590): 28.6% failure rate, `Turns=0` (engine-crash signature, not logic failure)
- `copilot-session-insights` (#52668): a gate-workflow cluster including this one at 0/126 successes across 3 consecutive sampled days ("provenance-inversion" pattern)
- `api-consumption` (#52697): top-5 REST API consumer at 7,699 calls across only 9 runs
Filed as a new issue. This is a merge-blocking gate, so a silent failure mode here has more direct impact than a reporting-only agent — prioritize checking on this next cycle.

### Firewall hostname fix: 3rd consecutive verified-holding cycle
`api.githubcopilot.com` (correct hostname) exclusively used, 0 blocks, in a fresh 15-run PR Code Quality Reviewer sample. Consider retiring this specific spot-check after one more confirming cycle.

### `agenticworkflows logs` timeout bug: reconfirmed (9th time), deliberately NOT re-filed
Hard ~60s wall-clock cap regardless of `--timeout` param; effective ceiling ~40 runs per call. 8 prior issues (#49279, #51952, #47928, #47728, #48921, #48780, #45510, #42284, #37079) all closed without a durable fix. This cycle's workflow-logs sub-agent independently reproduced it with `--count 60` and `--start_date ...--count 100` both failing while `--count 40` succeeded. Decision: do not spend a 9th create_issue slot on an exact duplicate of a repeatedly-closed-without-fix bug; instead surfaced prominently in the discussion report body as a standing operational constraint. Revisit this decision next cycle — if it's still unfixed after another cycle, consider a different escalation (e.g. explicitly asking a human maintainer, since agent-filed issues on this specific bug have a 100% closed-without-fix track record).

### Source-verification lesson this cycle: 2 "closed = fixed" assumptions were wrong
- `getParsedSchemaDoc` (`pkg/parser/schema_compiler.go:82`) still returns `(any, error)` despite #50678 being closed for exactly this fix.
- `RunSummary`/`DownloadResult` (`pkg/cli/logs_models.go:225,252`) still share all 14 overlapping fields verbatim despite #47387 and #47439 both being closed for exactly this consolidation.
Both re-filed this cycle with direct source-line citations and explicit notes that the prior closures didn't land the fix. **New standing practice going forward: when a candidate code-quality task matches a CLOSED issue title, grep/read the actual current source before treating it as "already fixed" — closed status is not proof of a landed fix in this repo's issue-closing culture.**

### New concrete findings filed this cycle (dedup-checked against open issues first via `gh api search/issues`)
1. **Design Decision Gate merge-blocking hotspot investigation** — cross-verified by 3 monitors, no prior root-cause issue. Filed.
2. **`getParsedSchemaDoc` still `(any, error)`** despite #50678 closed — filed with source verification.
3. **Dead `SkipInstructions` field** in `pkg/cli/compile_config.go`, 20+ no-op call sites (more than the 4 the source discussion mentioned) — filed.
4. **AI Moderator abnormal token usage** (~123.7k avg over 2 runs, ~4.7x fleet's 26.4k avg) — filed as an investigate-with-larger-sample task, noted small sample size explicitly.
5. **`RunSummary`/`DownloadResult` 14-field duplication** in `pkg/cli/logs_models.go` — filed, noting 2 prior closed attempts (#47387/#47439) didn't land; flagged for the next agent to check why those attempts were abandoned before re-attempting the same approach.
6. **`RunsOn any` should be `RunsOnValue`** in `pkg/workflow/safe_jobs.go:21` — filed, source-verified the target type already exists in `pkg/workflow/repo_config.go:69`.
7. **PR Code Quality Reviewer reads a cache-memory file that's never written** — `.github/workflows/pr-code-quality-reviewer.md:99` reads `/tmp/gh-aw/cache-memory/pr-*.json` for continuity but no step in the 191-line file ever writes it (verified by reading the full file, not just grepping for "memory"). Filed.

Not filed (already self-filed/open from prior cycles, confirmed still open, not re-filed):
- MCPFailureSummary field duplication (#52517, filed 2026-08-13) — still open.
- PolicyCompiler seed-rule validation gap (pkg/intent/policy.go, filed 2026-08-13) — not re-checked this cycle, assume still open.
- Sentrux god_files_ceiling enforcement gap (filed 2026-08-13) — not re-checked this cycle; #52598 (daily-sentrux) this cycle appears to be a fresh/first-tracked baseline run per the discussion-mining sub-agent, so no direct evidence either way this cycle.
- httpnoctx FuncLit-boundary gap (#52627, open) — self-filed by sergo (#52628) already, matches independently, not duplicated.
- eslint-refiner require-error-code-in-thrown-error false positive (#52643, open) and require-sync-exec-timeout timeout:0 bug (#52645, open) — both self-filed already by eslint-refiner (#52646), not duplicated.

### Process note: dedup workflow that worked well this cycle
For each candidate: (1) `gh api "search/issues?q=repo:github/gh-aw+is:issue+<keywords>"` to check title/keyword matches, (2) if a CLOSED issue matches, grep/read the actual current source to verify whether the fix really landed before deciding to skip, (3) only skip filing if an OPEN issue already covers the same root cause/component. This caught 2 stale closures this cycle that would otherwise have been wrongly skipped as "already covered."

### Chronic lineages — status check this cycle (not re-filed as duplicates)
- Code Scanning Fixer: 6/15 (40%) this cycle but trending up sharply day-by-day (0%→20%→75%→100% Aug 11-14) — possible recovery in progress, worth one more cycle to confirm before declaring fixed.
- Agent Performance Analyzer - Meta-Orchestrator: single new engine-crash (0 turns, "Execute GitHub Copilot CLI" step), already auto-filed as #52720 — not yet chronic, watch for recurrence.
- Fleet-wide reliability: 79.5-82.1% adjusted this cycle, consistent with 2026-08-13's 82.5%/84.6% — the 2026-08-11 49% figure remains unreconciled/stale, not re-investigated this cycle (already flagged for closure last cycle).

### Repository Quality / Sentrux baselines
- Not independently re-measured this cycle beyond what's noted in trend_data.md — see that file for the discussion-sourced figures (audit-workflows 87.39%, safe-outputs 0/200 failures 3rd consecutive clean cycle, etc).
