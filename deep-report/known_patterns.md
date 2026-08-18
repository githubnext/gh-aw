## DeepReport Memory (2026-08-18T12:26:00Z)

### New pattern: "completed" fix issues can still be lying — verify against a fresh occurrence of the exact symptom, not just merge status
Issue #51113 was closed `completed` 2026-08-07 after PR #51195 merged, claiming to fix the copilot-session-data-fetch conversation-transcript bug. 11 days later the exact same symptom (0 transcript files) still occurs, per today's Copilot Session Insights report. **Lesson: "merged PR + closed completed" is necessary but not sufficient evidence a bug is fixed — when a report resurfaces a symptom that supposedly already has a merged fix, check whether the fix's actual code path is the one hit in production before assuming this is a "new" occurrence of an old, unrelated bug.** This is a stronger version of the [[flagged_items]] "auto-expired not_planned closures aren't evidence of a fix" lesson from last cycle — this time the closure *was* backed by a real merged PR, and it *still* didn't work.

### New pattern: a tool bug can look identical to the day-keyed-cache bug class even when the root cause is different code
`agenticworkflows logs` with no date params served ~11-day-stale data despite a freshly-written local cache file — same *symptom* (stale-looking data with a fresh-looking cache) as the discussions/issues day-keyed cache bug fixed by PR #53486, but a distinct tool/code path. **Lesson: when reproducing a "fresh file, stale content" symptom in a new tool, don't assume it's the same bug as a previously-fixed one — verify by testing the tool's parameterized (date-range) code path separately from its default path, since the default path is often the one carrying the latent bug.**

## DeepReport Memory (2026-08-18T06:23:00Z)

### New lesson: an auto-expired "not_planned" closure is not evidence a problem was fixed — always re-check live state
Issue #50515 (compiler_safe_outputs_job.go decomposition) auto-expired 2026-08-06 with `state_reason: not_planned` and zero fix PRs in its timeline. Six weeks later, an independent compiler-quality report rediscovered the *exact same* 144-line function, unchanged, at the same file/line. **Lesson: before treating a closed issue as "handled" when it comes up again in a new report, check `state_reason` and the timeline for an actual merged fix — an auto-expiry is a no-op, not a resolution.** Re-filed with this evidence attached. Still unassigned as of the 12:26Z cycle (~6h later) — watch for a 2nd stall.

### New pattern: auto-filed "[aw] Failed jobs: X" issues do not self-consolidate even when clearly duplicate
Found 16 open `[aw] Failed jobs: PR Sous Chef` issues (#53245-#53446) spanning 5 days, mostly the same `safe_outputs`-step failure, never linked or root-caused — confirmed independently by this same cycle's Issue Arborist run, which also declined to auto-link them (ambiguous root cause across excerpts). **Lesson: the failed-jobs auto-filer has no dedup/consolidation step; when a chronic per-run failure issue type recurs >5-10 times without resolution, that itself is the actionable finding (root-cause + consolidate), not something to wait out.** Filed as #53615, now in-progress (PR #53676 WIP) as of 12:26Z.

## DeepReport Memory (2026-08-18T00:31Z) — condensed

- Day-keyed-cache fix (PR #53486) confirmed working, fixed a *class* of bugs — but the same root-cause pattern (gating reuse on an exact-match key) recurred in a different tool this cycle (the `logs` tool, see above).
- Filed issues in this repo consistently get fixed fast when well-scoped and evidence-backed — median turnaround well under a day for the 06:23Z cycle's docs bundle (#53614, ~5h48m). Continue prioritizing narrow, evidence-backed filings over broad ones.
- "Label the unlabeled issues" remains a standing declined task type — backlog fluctuates (3→5→...) but resolves itself through normal triage without a dedicated task.
- The 100-entry discussions.json window causes permanent loss of unmined discussions if a cycle defers mining — always mine every new/updated discussion in the cycle it first appears.
- `gh api search/issues` unreliably populates `merged_at` for PRs — always confirm via the direct `gh api repos/.../pulls/{n}` or `gh pr view` endpoint before concluding a linked fix didn't land.
