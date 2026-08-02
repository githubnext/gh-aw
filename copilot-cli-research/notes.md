# Copilot CLI Research Notes

## Run History

| Date | Run ID | Total WFs | Copilot WFs | Key Findings |
|------|--------|-----------|-------------|--------------|
| 2026-07-16 | 29472014909 | 257 | 125 | engine.driver at 0, mcp-scripts at 1, model overrides at 10 |
| 2026-07-17 | 29555564313 | 258 | 125 | engine.driver at 6, mcp-scripts at 3, model overrides at 79, block-domains at 0 |

## Persistent Zero-Usage Features (Never Adopted)

These features have been at 0 usage for multiple consecutive runs:
- `engine.args` - Custom CLI arguments (--add-dir, --verbose, etc.)
- `engine.cwd` - Working directory override
- `engine.harness.max-attempts` / `engine.harness.delay` - Retry tuning
- `network.blocked` / `block-domains` - Domain blocklist (new feature, 0 adoption)

## Positive Trends

- `engine.driver`: 0→6 (new pattern emerging, 3 languages: Python, Node, TypeScript)
- Model overrides: 10→79 (massive adoption in 1 day - likely bulk addition)
- Version pinning: 18→38 (doubled)
- `bare` mode: 11→21 (+10)
- `mcp-scripts`: 1→3 (+2, recovering from regression)

## Recommendations Tracking

| Recommendation | Status | Issue |
|----------------|--------|-------|
| Add tracker-id to daily workflows | Pending | Created 2026-07-17 issue |
| Add block-domains to security workflows | Pending | Created 2026-07-17 issue |
| max-continuations for long workflows | Pending | Created 2026-07-17 issue |
| Harness retry tuning | Pending | Created 2026-07-17 issue |
| LSP servers for code analysis workflows | Pending | Created 2026-07-17 issue |
| Document engine.driver pattern | Pending | Created 2026-07-17 issue |
| 2026-07-29 | 30423121135 | 265 | 129 | mcp-scripts 3→13, block-domains 0→2, model overrides 79→86, share flag still 0 (3rd run) |

## Update 2026-07-29

- `mcp-scripts` adoption accelerated (3→13) — recommendation may be gaining traction.
- `block-domains` moved off zero for the first time (0→2), still marginal.
- `--share` flag remains at 0 usage across 3 consecutive runs — consider deprioritizing or providing a concrete worked example to unblock adoption.
- New finding: 6 of 11 custom agent files in `.github/agents/` are orphaned (never referenced by any workflow's `agent:` field).
- `harness` retry tuning still flat (0→1).

## Run 30514465830 (2026-07-30)
- 133/266 workflows use Copilot engine (~50%).
- --share flag usage remains very low (2), unchanged trend from prior run (0).
- 5 of 9 custom agent files (.github/agents/*.agent.md) remain orphaned/unreferenced.
- block-domains usage is 0 despite 68 workflows using network config - opportunity for tighter egress control.
- engine.args customization still rare (3 workflows) vs available flags like --no-ask-user, --allow-all-paths, --no-custom-instructions, --disable-builtin-mcps.

## Run 30605101336 (2026-07-31)
- 132/266 workflows use Copilot engine (~50%), stable vs prior run (133).
- `--share` flag usage dropped back to 1 (was 2) — 4+ consecutive runs near-zero; recommend deprioritizing or adding a worked example.
- `block-domains`/network `blocked:` usage grew 2→4, still marginal vs 142 workflows using `network:` config at all.
- Custom agent orphan rate unchanged: 5 of 9 `.github/agents/*.agent.md` files still never referenced (`create-safe-output-type`, `custom-engine-implementation`, `grumpy-reviewer`, `interactive-agent-designer`, `w3c-specification-writer`).
- `harness:` (retry tuning) usage still effectively flat (1).
- Positive: model overrides steady at 85, bare mode 23, max-autopilot-continues climbed to 22 (from 11 two runs ago) — strong continued adoption.
- engine.args (custom CLI args like --add-dir, --no-ask-user, --allow-all-paths, --no-custom-instructions, --disable-builtin-mcps) remains rare at 3 workflows despite rich flag surface in copilot_engine_execution.go.

## 2026-08-01 (Run 30684585899)
- Confirmed 5 orphaned agent files persist unchanged since last run (2026-07-31): create-safe-output-type, interactive-agent-designer, grumpy-reviewer, custom-engine-implementation, w3c-specification-writer.
- `--share` flag still only used in 1 workflow (copilot-cli-deep-research.md itself).
- `engine.args` remains completely unused (0 workflows) - a real gap for exposing new upstream CLI flags.
- Recommendation carried forward: convert these into a tracked PR/issue instead of leaving as recurring research findings.

## Run 30732964595 (2026-08-02)
- 268 total workflows; 134 reference `id: copilot` in engine block (99 files match `engine: copilot` shorthand pattern — counting methodology varies by pattern used).
- `max-autopilot-continues` dropped to 0 this run (was 22-24 in recent runs) — needs verification, possible measurement/pattern change or real regression; flag as anomaly to double check next run.
- `--share` flag usage still stuck at 1 (only the deep-research workflow itself) across 5+ consecutive runs. Recommend closing out this recommendation as "won't fix" or demonstrating a concrete example PR.
- `engine.args` (custom CLI args) still 0 across all tracked runs — persistent, longest-standing real gap alongside orphaned agents.
- `harness:` (retry tuning) still 0.
- `max-tool-denials` now widely adopted: 66 workflows (up from long-stuck 0 in earlier tracked history) — major positive trend, recommendation resolved organically.
- `block-domains`/`blocked:` still marginal: 2 workflows vs 143 using `network:` config — persistent security gap.
- Orphaned custom agents unchanged: 5 of 9 agent files never referenced (create-safe-output-type, custom-engine-implementation, grumpy-reviewer, interactive-agent-designer, w3c-specification-writer). This has now persisted for 6+ runs — recommend either wiring them into workflows or deleting them.
- `mcp-servers:` custom MCP usage down to 1 workflow (was higher in earlier runs at 8-13) — investigate whether this is a real decline or pattern-matching difference.
