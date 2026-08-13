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
