## DeepReport Memory (2026-08-20, ~05:45Z cycle)

### Reconfirmed: audit/quality workflows now self-file the large majority of their own findings — DeepReport's marginal value is in the dedup check + the few gaps they miss
Of ~9 distinct findings surfaced across 10 discussions this cycle, only 5 (Schema Consistency Checker's 3 findings + Docs Noob Tester's docs gap + compiler-quality's godoc gap) lacked a self-filed issue; the rest (Workflow Skill Extractor ×3, Sergo, ESLint Refiner, LintMonster, MCP-auth-test) were already filed by their source workflow before this analysis ran. **Lesson: always run the dedup search before treating any "new-looking" finding as a filing candidate — this repo's reporting workflows increasingly file their own issues directly via safe-outputs, so DeepReport's job is verification + gap-filling, not first-filing.**

### New pattern: a scanner's own generated intermediate data can itself be the finding
Schema Consistency Checker (#54161) found bugs in the app being audited (strict-mode default, redirect docs) but *also* found that its own precomputed `schema-diff.json` field-gap data has false positives from a nested-key extraction bug — filed as a 3rd, independent issue. **Lesson: when a recurring audit tool references its own precomputed/cached intermediate artifact, treat quality problems in that artifact as a first-class finding, same as [[known_patterns]]'s existing lesson about a report's own stale config being a valid finding.**

## DeepReport Memory (2026-08-20, ~00:25Z cycle)

### Confirmed: "closed completed" can mean "only part of the named scope was done" — always verify every named target, not just one
#53788 named 5 oversized files but was closed completed after PR #53818 split only the largest one (`compiler_jobs_test.go`). The other 4 were never checked against the closing PR's actual diff. Verified via `wc -l`: 3 of the remaining 4 are still oversized (unfixed, re-filed this cycle); the 4th (`compiler_safe_outputs_config_test.go`) actually was separately fixed elsewhere (965L now) — so partial-scope closures can go either way per-file. **Lesson: when an issue names multiple targets, check the closing PR's diff (or current state) against every named target individually before assuming closure = full completion, and don't assume the unfixed ones stayed unfixed either — verify each one.**

### New tooling quirk: `gh issue list --search` fails in this sandbox; use `gh api search/issues` instead
`gh issue list --search "..." --state all --json ...` reproducibly fails with `malformed version:` here (not an auth issue — `gh api` reads work fine, `gh --version` is 2.97.0). **Lesson: for dedup searches, always use `gh api "search/issues?q=repo:OWNER/REPO+terms+in:title,body" --jq '...'` instead of `gh issue list --search`.**



### Confirmed: repo-memory's own max-patch-size limit can silently drop a full cycle's writes
Discussion #53999 (12:34Z cycle) should have updated these 3 files but didn't — its 13KB combined diff exceeded the 10KB `max-patch-size` limit and the push hard-failed, discarding all 6 changed files. Root-caused and filed as #54010 this same cycle (fix PR #54029 open, not yet merged). **Lesson: when this file's "latest cycle" doesn't match the most recent DeepReport discussion post, don't assume the cycle didn't run — check whether a push silently failed, and recover the missing baseline from the discussion body itself.** Also keep future memory-file diffs lean (append small sections, avoid full-file rewrites) until #54029 lands.

## DeepReport Memory (2026-08-19T05:45:00Z)

### New pattern: a quality/audit workflow's own configuration can go stale, and that is itself a valid finding
Daily Compiler Code Quality Check (#53892) is configured to analyze a fixed file list; 2 of those files (`compiler_activation_jobs.go`, `compiler_safe_outputs_config.go`) no longer exist in the tree after refactors, so the run silently substituted others. **Lesson: when a recurring auditor/report workflow references specific file paths or targets in its config, treat "target no longer exists" as a first-class finding about the auditing workflow itself, not just noise to filter past — otherwise its effective coverage silently shrinks over time as the codebase changes underneath it.**

### Reconfirmed: self-filing workflows still need a dedup search before DeepReport files anything on their behalf
Both Sergo (#53902, generatedyamlheredoc bug) and separately LintMonster/ESLint Refiner this cycle produced real findings that were already auto-filed by the reporting workflow itself (#53901 for Sergo's case) before DeepReport's analysis pass ran. **Lesson: always run the dedup search step even for findings that read as "new" in the report body — the workflow that found it often already filed it in the same run, just not yet reflected in the discussion post's own "issues created" section.**

## DeepReport Memory (2026-08-19T00:15:00Z)

### New pattern: persistence through re-filing can eventually work, even after 2 stalls
`compiler_safe_outputs_job.go` decomposition was originally #50515 (auto-expired unfixed), re-filed as #53612 (stalled ~6h unassigned), then finally got PR #53720 merged this cycle — 3rd attempt, ~2 days total. **Lesson: don't give up on re-filing a well-scoped, evidence-backed task after one stall — this repo's fast-turnaround norm can still apply, just with more latency for tasks that first get deprioritized.**

### New pattern: a merged fix takes time to show up in trailing-window metrics — don't re-file just because the metric hasn't moved yet
The Copilot PR Prompt Analysis report's 30-day trailing window still shows ~49% CVE-cluster success even though the pre-filter fix (#53709) merged same-day. **Lesson: when a report's headline metric is a rolling/trailing window and a relevant fix just merged, expect a lag before the metric reflects it — check the merge date against the window before concluding a fix "didn't work."**

### New pattern: detection/observability audits are strongest on their first baseline run
Detection Analysis Report, Agent Job Health Monitor, and Daily Observability Report all explicitly noted "first recorded baseline, no trend chart yet" this cycle. Each still produced real, actionable, non-duplicate findings despite having no history to compare against.

## DeepReport Memory (2026-08-18T18:23:34Z)

### New pattern: an auditor's own report can partially diagnose a scanner bug and still miss the rest of it — always generalize a root cause across all affected checks
Today's GEO Audit (#53758) correctly explained *why* its `robots_txt` check false-negatived (scanner probes domain root, not the `/gh-aw/` project path) but didn't apply that same reasoning to its `llms_txt`/`ai_discovery` checks, which are false negatives for the identical reason (or a closely related one) — verified live via `curl`, both `https://github.github.com/gh-aw/llms.txt` and `/.well-known/ai.txt`/`/ai/*.json` return 200 with real content despite the report claiming 0 scores. **Lesson: when a report explains away one false-positive/false-negative as a "scanner limitation," check whether sibling checks in the same tool share the same root cause before accepting their findings at face value — tool authors fix the symptom they noticed, not necessarily the whole bug class.** This directly caused 4+ open near-duplicate `[geo-optimizer]` issues asking to create files that already exist; a prior dedup-check fix (#48695) didn't prevent this because the root issue is the scanner producing wrong data, not a missing dedup step.

### Confirmed again: "closed completed" with a merged PR is still not proof of a working fix at the tool level
Continuing the #51113 pattern from last cycle — this cycle's GEO case (#48045/46/209, closed completed) shows the same shape: real merged PRs, files genuinely deployed and working, yet the *auditing tool* still reports them as missing. Verifying the tool's actual check (not just the target artifact) is the piece that's easy to skip.

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
