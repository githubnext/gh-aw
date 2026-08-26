# Copilot CLI Research Notes

## Run: 2026-08-23 (workflow-run-id: 32616605386)
First comprehensive analysis. Baseline established.

Key gaps identified:
- repo-memory underused vs cache-memory for trend workflows
- no shared Copilot engine defaults snippet
- --share, --disable-builtin-mcps, --add-dir manual usage near-zero
- model slug inconsistency (small/large aliases vs concrete slugs)

Follow-up: track adoption of these recommendations in next run.

## Run: 2026-08-24 (workflow-run-id: 32688298194)
Second analysis. Compared to 2026-08-23 baseline.

Changes since last run:
- copilot_workflows metric jumped from 39 to 109 (measurement scope widened to include `id: copilot` nested form, not necessarily new adoption)
- max-continuations now used in 10 workflows (up from untracked baseline)
- cache-memory (105) and repo-memory (29) usage both grew
- copilot-sdk: true now in 61 workflows — strong adoption of SDK mode
- Still zero workflows pin `engine.version` for copilot
- `--share` and `--allow-all-paths` remain completely unused in any workflow
- `--disable-builtin-mcps` manual usage still only 2 workflows
- Custom agent files (`engine.agent`) genuinely used in only ~7 workflows; most `agent:` matches are template comments, not live config

New opportunities identified:
- No shared/otlp.md-style shared snippet exists yet for common Copilot engine defaults (version pin + model + harness retry tuning)
- Plugins feature (Agent Plugins) still at 0 adoption across all workflows despite being documented as supported

Follow-up: track whether shared Copilot defaults snippet gets created; monitor plugins adoption trend.

## Run: 2026-08-25 (workflow-run-id: 32806958803)
Third analysis. Compared to 2026-08-23 and 2026-08-24 baselines.

Changes since last run:
- copilot_workflows steady at 109 (no further scope changes)
- copilot-sdk: true flat at 61 (no growth since run 2) — plateaued
- max-tool-denials newly measured at 55 workflows
- engine.agent custom persona usage still stuck at ~7 workflows — three runs, no growth despite being flagged twice
- engine.version pinning: still 0 across all 109 Copilot workflows — three consecutive runs confirm this is not organically improving
- --share flag: still 0 manual adoption (only used internally by compiler for detection-job commands)
- plugins: still 0% adoption across all 292 workflows, three runs running
- No shared/copilot-defaults.md snippet created yet, despite being recommended in run 2

Escalation: the three most persistent, non-improving gaps (version pinning, --share, plugins, agent personas) are being escalated as concrete action items in this run's issue rather than repeated vague opportunities, since 3 cycles of flagging without action indicates these need either a decision to intentionally deprioritize or a committed implementation plan.

Follow-up: check whether shared/copilot-defaults.md gets created before next run; check if copilot-sdk adoption resumes growth or has truly plateaued at 61.

## Run: 2026-08-26 (workflow-run-id: 32928414627)
Fourth analysis. Compared to 2026-08-23, 08-24, and 08-25 baselines.

Changes since last run:
- total_workflows grew 292 -> 294
- copilot_workflows (id: copilot) dropped 109 -> 89; cache-memory dropped 105 -> 80. Both drops coincide with this cycle using a stricter/different grep pattern than prior cycles — flagged as a likely methodology discrepancy, NOT a real adoption regression. Next cycle should standardize and record the exact query pattern used.
- copilot-sdk: true confirmed flat at 61 for a THIRD consecutive measurement — genuinely plateaued, not noise.
- max-tool-denials held steady at 55 (same as run 3) — stable moderate adoption.
- engine.agent persona usage re-measured with a stricter query (matching `agent:` nested directly under `engine:` block) and found 0, versus ~7 in runs 2-3 which used a looser top-level `agent:` match. Likely the prior ~7 conflated agent-file *imports* (`imports: [.github/agents/x.md]`) with the actual `--agent` CLI flag trigger (`engine.agent`). This is a metric-definition fix, not necessarily an adoption drop.
- shared/copilot-defaults.md: STILL does not exist. This is now 2 full cycles (08-24 recommendation, 08-25 escalation, 08-26 this run) with zero action — the single most overdue, actionable item in the backlog.
- engine.version pinning, --share manual use, and plugins adoption: all confirmed still at 0/89 (or 0/294 for plugins) for the 4th consecutive cycle.

Escalation (continued from run 3): version pinning, --share, plugins, and agent personas remain unactioned after 3-4 cycles of flagging. This run explicitly recommends a decide-or-implement framing for shared/copilot-defaults.md specifically, since it is the lowest-effort, highest-leverage item and unblocks systemic fixes for the other three.

Follow-up for next cycle:
1. Standardize and document the exact grep pattern for `id: copilot` and `cache-memory:` counts to stop cross-cycle discrepancies.
2. Check whether shared/copilot-defaults.md was created (3rd cycle asking).
3. Manually spot-check 3-5 workflows previously flagged as "agent users" (e.g. archie.md) to determine ground truth on engine.agent vs. agent-file imports.
4. Confirm with a maintainer whether `plugins` is still a supported/relevant feature before continuing to track its 0% adoption a 5th time.
