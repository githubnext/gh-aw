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

## Run: 2026-08-27 (workflow-run-id: 33042746396)
Fifth analysis. Compared to 08-23, 08-24, 08-25, 08-26 baselines.

Changes since last run:
- copilot_workflows measured at 108 this cycle using combined `engine: copilot` OR `id: copilot` grep — closer to run 2/3's 109 than run 4's stricter 89. Recommending this combined query as the standard going forward to end the cross-cycle count discrepancy noted in run 4.
- engine.agent count corrected: run 4 reported 0 due to an overly strict query (matching only literal `agent:` directly under `engine:` in a narrow sed window). Direct ground-truth grep this cycle confirms 7 genuine engine.agent custom-persona usages: archie, contribution-check, daily-file-diet, glossary-maintainer, hourly-ci-cleaner, technical-doc-writer, workflow-generator — consistent with runs 2-3's ~7 estimate. Treat run 4's "0" as a query bug, not a real regression.
- copilot-sdk: true confirmed flat at 61 for a 4th consecutive measurement — plateau is now very well established.
- max-tool-denials steady at 55 (3rd consecutive matching measurement).
- shared/copilot-defaults.md: STILL not created. This is now the 3rd cycle since the original recommendation (run 2) and 2nd since escalation (runs 3-4). No other single item has been flagged for this long without any action.
- engine.version pinning, --share, plugins: all confirmed at 0 adoption for a 5th consecutive cycle.

Follow-up for next cycle:
1. Use combined `engine: copilot` OR `id: copilot` grep as the standard copilot_workflows count (108 this cycle) to stop the run-to-run methodology drift.
2. Check whether shared/copilot-defaults.md was finally created (4th cycle asking) — if not, this item may warrant direct escalation outside the research issue (e.g., a dedicated tracking issue) rather than repeated mention here.
3. Verify whether --share is still a real, wired-up CLI flag in the current Copilot CLI version, since it was not found directly in copilot_engine_execution.go's flag-construction code this cycle.
4. Re-check copilot-sdk adoption in 08-28+ cycle; if still flat at 61, consider this fully settled and stop asking, unless a maintainer requests reinvestigation.

## Run: 2026-08-29 (workflow-run-id: 33232092722)
Seventh analysis. Compared to runs 1-6 (08-23 through 08-27... note run6 08-28 data was read this cycle too).

Changes since last run:
- copilot_workflows steady at 111 (combined engine:/id: copilot grep now standard per run5/6 recommendation) — count stability across cycles confirms the methodology fix worked.
- engine.version pinning: genuine growth trend continues, now 38 workflows (was 25 last cycle, 0 through runs 1-4). This is real, organic adoption and should be called out positively.
- cache-memory grew slightly 80->82; repo-memory flat at 30.
- copilot-sdk: true confirmed flat at 61 for a 5th consecutive cycle — fully plateaued, recommend maintainers explicitly decide to stop tracking this metric unless requested, since 5 cycles of no change confirms saturation/decision-not-to-adopt-further rather than an in-progress trend.
- engine.agent custom persona usage flat at 7 for a 4th consecutive cycle (same 7 named workflows) — genuinely stalled, no organic growth despite repeated flagging.
- shared/copilot-defaults.md: STILL not created after 5 cycles of flagging (run2 origin, escalated run3/4/5/6). This is now the single most overdue unactioned item across all cycles of this research. Recommending this be escalated as a standalone tracked GitHub issue outside the recurring research report, since repeating it here has not produced action for 5 consecutive runs.
- plugins: still 0% adoption, 7th consecutive cycle — recommend a maintainer explicitly decide whether to keep pursuing this or deprioritize it in future research to avoid repeating a permanently-0 metric indefinitely.
- --block-domains: confirmed present in compiler code but 0 adoption in any workflow markdown for a 2nd cycle — newer feature, still very early, worth one more cycle of tracking before treating as stalled.

Follow-up for next cycle:
1. Confirm engine.version pinning keeps growing (38 this cycle) — if it continues, this is a good-news trend to eventually retire from "opportunity" framing and reclassify as an adopted best practice.
2. Decide (with a maintainer) whether plugins and copilot-sdk adoption tracking should continue being repeated indefinitely, since both are now fully flat/0 for 5+ cycles.
3. If shared/copilot-defaults.md is still absent next cycle, stop repeating it here and instead open a dedicated tracking issue referencing this research history.

## Run: 2026-09-02 (workflow-run-id: 33588246419)
Tenth analysis. Compared to all prior cycles (08-23 through 09-01).

Changes since last run:
- copilot_workflows steady at 107 (combined grep standard holds).
- engine.agent re-verified via strict AWK scan (engine: block only, excludes sandbox.agent) = 7, confirming the same 7 named workflows for 6+ consecutive cycles - genuinely stalled with no organic growth.
- copilot-sdk: true flat at 61 for an 8th consecutive cycle - fully settled/plateaued; recommend maintainers explicitly stop tracking this unless a re-investigation is requested.
- plugins: 0 adoption for a 10th consecutive cycle - escalating as a required maintainer go/no-go decision rather than a repeated observation.
- shared/copilot-defaults.md: STILL not created after 8 cycles since origin (run2, 08-24). Recommending this be moved OUT of the recurring research report into a standalone dedicated tracking issue, since repetition here has not produced action across 8 cycles.
- --share confirmed used in exactly 1 workflow: this research workflow itself (copilot-cli-deep-research.md). All other workflows: 0 adoption, now 5+ cycles confirmed.
- engine.version pinning measured strictly this cycle at 14 - notably lower than run7's looser-pattern 38. Flagged as a methodology discrepancy (grep needs to scan full engine: block, not just adjacent lines) rather than a regression - needs reconciliation next cycle.
- network.allowed at 183 (61% of all workflows) vs --block-domains at 0 (5+ cycles flat) - recommend docs deprioritize block-domains promotion.

Follow-up for next cycle:
1. Check whether shared/copilot-defaults.md was finally created after being escalated to a standalone issue this cycle.
2. Reconcile engine.version pinning count (14 vs 38) using a standardized full-engine-block grep pattern; record the exact pattern used in repo-memory for future consistency.
3. Confirm with a maintainer whether plugins and copilot-sdk tracking should continue every cycle or move to a quarterly check, given both are now fully flat for 8-10 cycles.
4. If shared-defaults still absent, consider this research format itself needs a "escalated items" section that persists rather than being re-derived each cycle.
