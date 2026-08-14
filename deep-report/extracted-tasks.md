## Extracted code-quality tasks (2026-08-14 cycle)

1. Investigate Design Decision Gate merge-blocking failure/cost hotspot (cross-verified by 3 monitors) — filed.
2. Fix `getParsedSchemaDoc` return type in pkg/parser/schema_compiler.go:82 (prior #50678 closure didn't land) — filed.
3. Remove dead `SkipInstructions` field from pkg/cli/compile_config.go — filed.
4. Investigate AI Moderator's ~4.7x-average token usage — filed.
5. Consolidate RunSummary/DownloadResult 14-field duplication in pkg/cli/logs_models.go (prior #47387/#47439 closures didn't land) — filed.
6. Replace `RunsOn any` with `RunsOnValue` in pkg/workflow/safe_jobs.go:21 — filed.
7. Fix/remove dead per-PR cache-memory read in pr-code-quality-reviewer.md:99 (file never written) — filed.

Not filed (already open from prior cycles, confirmed still open):
- MCPFailureSummary field duplication (#52517) — still open, not re-filed.
- PolicyCompiler seed-rule validation gap (pkg/intent/policy.go) — not re-checked this cycle.
- Sentrux god_files_ceiling enforcement gap — not re-checked this cycle (#52598 appears to be a fresh baseline, no direct evidence either way).
- httpnoctx FuncLit-boundary gap (#52627) — self-filed by sergo already, not duplicated.
- eslint-refiner require-error-code-in-thrown-error (#52643) and require-sync-exec-timeout (#52645) — self-filed by eslint-refiner already, not duplicated.

Also: added a follow-up comment to #52518 (shared PR-review infra flakiness investigation) with evidence that all 3 affected agents (Test Quality Sentinel, Matt Pocock, Ponytail Reviewer) are now at 100% success, recommending closure.

## Extracted code-quality tasks (2026-08-13 cycle)

1. Fix Sentrux `god_files_ceiling` rule not enforced ("0 rules checked") — filed.
2. Add Autonomy/WriteScope validation to PolicyCompiler's seeding rule in pkg/intent/policy.go — filed.
3. Embed AggregatedSummaryBase in MCPFailureSummary instead of hand-copying fields — filed.
4. Add graceful noop fallback to Test Quality Sentinel's pre-fetch pipefail script — filed.
5. Investigate shared PR-review pre-fetch infra flakiness (Aug 7/11/13 correlated dips) — filed.
6. Add pr-triage sub-agent fallback to Matt Pocock Skills Reviewer — filed.
7. Add Success Criteria + noop-vs-silence distinction to Ponytail Reviewer — NOT FILED this cycle (hit the 7-issue cap due to a shell-quoting mishap consuming an extra create_issue slot on a duplicate retry). Carry forward to next cycle as top priority.

Not filed (already self-filed by the reporting agent itself this cycle, cross-checked for no gap):
- Sergo: `hardcodedfilepath` linter Exported()-only filter bug — matches open #52428.
- ESLint Refiner: `require-invalid-date-check-before-compare` false positive/negative — matches open #52441/#52442.
- GitHub Remote MCP `get_repository` tool missing — matches open #52444/#52445.

Not filed (chronic, carried forward from prior cycles, still open, not re-filed):
- `coverage.findProfile` path-matching bug (#52309, filed 2026-08-12).
- `GitHubToken` field-shadowing (#52313, filed 2026-08-12).
- Fleet-wide reliability investigation (#51935-adjacent, 2026-08-11) — likely resolved per this cycle's convergent evidence (see trend_data.md), recommend closing next cycle if still open.

Previous cycles' tasks (for reference, all previously filed):
### 2026-08-12
1. Fix `coverage.findProfile` path-matching bug — filed as #52309.
2. Fix `api.individual.githubcopilot.com` misdirected hostname — filed, VERIFIED FIXED this cycle via commit b2ef1f3/#52377.
3. Add `gh-aw-detection: true` to 8 scheduled workflows — filed.
4. Fix schema-consistency tooling stale-file target — filed.
5. Remove shadowed `GitHubToken` field re-declarations — filed as #52313.
6. Re-file `agenticworkflows logs` MCP timeout — filed.
7. Document label pre-creation requirement in AI issue triage guide — filed.

### 2026-08-11
1. Fix inverted `strict:` mode documentation — filed as #52086, VERIFIED FIXED.
2. Add missing `repository_dispatch` to `user-rate-limit.events` schema enum — filed.
3. README agent-bootstrap block Copilot-default gap — filed.
4. Consolidate `JobStep`/`JobStepData` identical structs — filed.
5. Shared base type for 4 log-entry structs — filed.
6. Split `pkg/workflow/compiler_types.go` — filed.
7. Investigate 49% agent-job failure rate — filed, likely resolved per 2026-08-13 convergent evidence.
