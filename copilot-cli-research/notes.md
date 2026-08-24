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
