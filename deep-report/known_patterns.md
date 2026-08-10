## DeepReport Memory (2026-08-10T15:00:00Z)

### Meta: path/persistence fix applied this cycle
Previous memory (read from the old `memory/deep-report/` path) was frozen at 2026-04-03 despite claims in #51116 (2026-08-07) of a fix. Root cause confirmed: `.github/workflows/deep-report.md` writes to a two-level-deep path (`memory/deep-report/`) which the slashless `file-glob: ["*.md"]` pattern silently drops (matches only one level deep). Issue #51172 diagnosed this correctly on 2026-08-07 but was closed with no linked merged PR — the source file is still unfixed as of 2026-08-10. This cycle writes to the corrected one-level path (`deep-report/`) as a workaround; filed issue "Actually land the DeepReport repo-memory path fix from #51172" to fix the source file itself. See `flagged_items.md`.

### Systemic pattern discovered this cycle (headline finding)
5 independent recurring bugs have each been closed as "fixed" 1-7+ times without the underlying defect resolving, confirmed via fresh reports on 2026-08-10:
1. DeepReport repo-memory persistence (above).
2. Copilot Session Insights conversation transcripts — 44+ consecutive days at 0 files (6 prior closures since Feb).
3. Firewall/MCP raw log retention — 0.0% coverage today (6 prior closures since Feb, most recently #51111 on 2026-08-07).
4. Quick Start docs jargon (frontmatter/engine/safe-outputs undefined) — flagged again today (7+ prior closures since April).
5. Quick Start engine-choice guidance — flagged again today (6 prior closures).
Filed a process issue proposing a "verified-merged-evidence" gate before closing this class of issue.

### Fleet health (from cross-referenced discussions, 2026-08-09/10)
- Agentic Workflow Audit (#51643, window 08-08→08-09): 323 runs, 96.6% raw / 96.9% adjusted success (excl. 2 intentional-failure stress tests). 34-day gap since prior audit (07-06) — now resolved (audit ran again).
- Safe Output Health (#51688): 202 runs, safe_outputs job 98.4% success of attempted (184/187); sole cluster (`resolve_pull_request_review_thread` stale node ID, PR Sous Chef) already fixed same-day via commit 59c283e / PR #51630.
- Fresh 50-run logs sample (this cycle, 2026-08-09→10): 5/50 (10%) driver_exit_failures across claude/codex/copilot/pi/goose — consistent with the audit's reconfirmed cross-engine `copilot-sdk-driver-failures` known issue; not re-filed (chronic, broader root-cause effort per prior cycle's judgment).
- Open P0 today: #51789 Copilot CLI harness segfault (exit 139) killing Agent Performance Analyzer — already tracked, not duplicated.
- Issues: 500-sample window (7d) — 134 open / 366 closed. Top labels: agentic-workflows (210), automation (176), cookie (114), testing (55), code-quality (38).

### Active/fresh findings this cycle
- Detection Analysis (#51652): "Q" and "ESLint Monster" missing `gh-aw-detection: true` — filed.
- Audit-workflows' own `recommendations.json`/`workflow-trends.json` stuck at 2026-07-06 snapshot due to minified-JSON patch-size reverts — filed (reformat to indent=2).
- Typist (#51765): codebase type-safety strong overall (0 unintentional dup types, ~0 raw interface{}); tiny doc-comment gap on 2 wasm files — filed.
- Security Observability (#51613): PR Code Quality Reviewer responsible for 32/38 fleet-wide firewall blocks (`api.individual.githubcopilot.com`) — commented on related open issue #51802 rather than filing a dup.
