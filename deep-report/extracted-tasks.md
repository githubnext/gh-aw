## Extracted code-quality tasks (2026-08-12 cycle)

1. Fix `coverage.findProfile` path-matching bug making the perf-linter coverage gate a silent no-op — filed.
2. Fix `api.individual.githubcopilot.com` misdirected hostname in PR Code Quality Reviewer — filed.
3. Add `gh-aw-detection: true` to 8 scheduled audit/report/detector workflows — filed.
4. Fix schema-consistency tooling stale-file target + `frontmatter-full.md` missing fields — filed.
5. Remove shadowed `GitHubToken` field re-declarations in 3 project-config structs — filed.
6. Re-file `agenticworkflows logs` MCP timeout (recurrence of auto-expired #51952) — filed.
7. Document label pre-creation requirement in AI issue triage guide — filed.

Not filed (chronic, folding into existing process-gate/tracking evidence instead of re-filing):
- Copilot Session Insights: new 35-day sampling gap on top of original transcript gap since 2026-06-23 (#52255).
- Fleet-wide 49% agent-job failure rate: team-evolution report (#52145) offers a plausible explanation but no formal reconciliation yet — investigation issue from 2026-08-11 stays open, not re-filed.

Not filed (already self-filed by the reporting agent itself this cycle, cross-checked for no gap):
- LintMonster: function-length backlog (#52205/#52206/#52207), closed stale duplicate #50982.
- Cache Strategy: Linter Miner cache-miss fix (matches tracked #52134).
- ESLint Refiner: reassignment-guard bug + chained-method-call detection gap (matches open #52240/#52241).

Not filed (duplicate of existing open issue found during dedup check):
- `pkg/cli/update_actions.go` split — already tracked as #52054 (file-diet).
- `JobStep`/`JobStepData` consolidation, log-entry-struct base type, compiler_types.go split — all filed in 2026-08-11 cycle, still open.

Previous cycles' tasks (for reference, all previously filed):
### 2026-08-11
1. Fix inverted `strict:` mode documentation — filed as #52086, VERIFIED FIXED this cycle (see known_patterns.md).
2. Add missing `repository_dispatch` to `user-rate-limit.events` schema enum — filed.
3. README agent-bootstrap block silently defaults Claude Code users to Copilot-oriented artifacts — filed.
4. Consolidate `JobStep`/`JobStepData` identical structs in `pkg/cli` — filed.
5. Give the 4 independent log-entry structs a shared base type — filed.
6. Split `pkg/workflow/compiler_types.go` — filed.
7. Investigate 49% agent-job failure rate — filed, still open/unreconciled.

### 2026-08-10
1. Require verified-merged evidence before closing self-filed reliability/doc issues (process gate).
2. Land the DeepReport repo-memory path fix from #51172 for real.
3. Add gh-aw-detection: true to Q and ESLint Monster.
4. Reformat audit-workflows recommendations.json/workflow-trends.json to indent=2.
5. Add native-counterpart doc comments to progress_wasm.go / spinner_wasm.go.
6. Investigate firewall/MCP log retention via upload-side glob/path-depth hypothesis.
