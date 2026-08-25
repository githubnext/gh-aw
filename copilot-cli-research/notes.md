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
