## DeepReport Memory (2026-08-20, ~17:50Z cycle, baseline #54233)

### New pattern: a report's own quantitative claim can be flatly wrong — always spot-check the numbers, not just whether the files exist
Discussion #54241 (Error Handling Consistency & Panic Safety) claimed "919 of 2,504 non-test `fmt.Errorf` call sites (~37%) still format an error with `%v` instead of `%w`," concentrated in `init.go`, `dispatch.go`, `run_push.go`, `upgrade_command.go`, `audit.go`. Direct `grep` found the true repo-wide count is 20/2546 (<1%), and all 5 named files have **zero** `%v`-formatted `fmt.Errorf` calls. Declined to file this sub-finding. The report's other 2 claims in the same discussion (panic() sites, brittle string-matching checks) were separately verified true via grep and filed. **Lesson: don't extend "verify files exist" to "trust the report's numbers" — a source report can be internally inconsistent (some claims accurate, one wildly wrong) in the same discussion. Spot-check the specific metric being cited (grep the actual count), not just that the named files are real, before filing.**

### New pattern: config-based firewall/network gaps are directly verifiable and make clean quick-win issues
#54290's npm-registry-block finding (PureLock, Dead Code Removal Agent, Daily AIC Consumption Report) was verified by reading each workflow's frontmatter `network.allowed` list directly — all three were missing the `node` ecosystem preset used by `pkg/workflow/domains.go` to cover `registry.npmjs.org`. This is the same shape as the earlier `proxy.golang.org`/Code Scanning Fixer allowlist gap (#54063) — a recurring, easily-verified pattern (missing ecosystem preset in `network.allowed`) worth checking directly in workflow `.md` frontmatter whenever a firewall report names specific blocked-domain/workflow pairs.

## DeepReport Memory (2026-08-20, ~12:30Z cycle)

### Reconfirmed: "closed" duplication/permission fixes can regress or never have landed — always verify against the live tree, not the issue tracker
Two of this cycle's 5 Typist-sourced findings are exact re-occurrences of previously "closed" issues: #51076 (GitHubMCPDockerOptions/GitHubMCPRemoteOptions consolidation) and #51032 (get_teams permission gating, closed 2026-08-08, symptom recurring per today's #54223 "2nd consecutive day"). Both were verified live via direct code/report inspection before re-filing, not assumed from the tracker. **Lesson: this is now a repeat pattern (3rd+ time this memory has recorded it) — for any Typist/type-consistency finding, always grep the live code before trusting a closed issue's title as evidence the gap is gone.**

### New pattern: a scanner's own root-cause section can point to a *process* fix, not just a code fix
Prompt Clustering Analysis (#54207) traced 82% of its lowest-merge PR cluster to a single mechanical fingerprint (1 commit/0 files/0 comments) and explicitly recommended a pre-assignment staleness/dedup screen for auto-generated backlog tasks — directly relevant to DeepReport's own dedup-gate discipline, since duplicate `llms.txt` requests recurred 4 times (#51616/#52115/#53158/#53760) before being queued as agent work. **Lesson: when a report's root-cause section names a concrete process gap (not just a code bug), it's still filing material — don't restrict "actionable tasks" to code diffs only.**

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

## DeepReport Memory (2026-08-20, ~23:36Z cycle, baseline #54319)

### New pattern: a narrow "closed" fix can be exactly right but never generalized, then regress at fleet scale
#41253 (2026-06-24, closed) fixed the codex-gh-aw-binary-not-found signature for one workflow (Daily Cache Strategy Analyzer). Today's audit (#54358) shows the identical root cause now causes a 100% fleet-wide outage across 10 codex-engine workflows — the original fix was correct but scoped to a single workflow's config instead of the shared engine code path. **Lesson: when a fix issue is scoped to "workflow X" but the root cause lives in shared engine/compiler code, check whether the fix actually touched the shared path or just X's config — a symptom fixed once for one caller can resurface fleet-wide as more callers exist.**

### New pattern: two independently-generated same-day reports corroborating one failure signature is strong confirmation, not coincidence
The Agentic Workflow Audit (#54358) and the Detection Analysis Report (#54377) were generated by different workflows in the same ~5h window and both independently flagged AI Moderator's 0% success on the codex engine, plus Ponytail Reviewer and Daily Go Test Parallelizer's low success rates — without referencing each other. **Lesson: when two same-day, independently-run reports converge on the same specific workflow+metric, treat that as elevated-confidence evidence, not something requiring extra verification before filing.**

### Reconfirmed: a chronic pattern with 5+ prior closed "fix" attempts that never stuck is not a good re-filing candidate — it needs a different kind of task, or none
"Instrument Copilot CLI stderr/exit code on 0-turn failure" has been proposed and closed at least 5 times since 2026-07-01 (#42789, #42876, #43814, #43906, #47349, #50304, #53180) yet still recurs (audit's own tracking: recurrence 25). Declined to file a 6th generic re-ask this cycle. **Lesson: past a certain repeat count, re-filing the same generic ask is noise — either the fix needs a fundamentally different approach (not just "add logging"), or it's a genuinely hard upstream CLI/SDK limitation not fixable from this repo. Don't keep re-filing verbatim; note the chronic-unstuck pattern itself as the finding instead.**

## DeepReport Memory (2026-08-21, ~00:39Z cycle, baseline #54396)

### Reconfirmed: source workflows self-file the large majority of their own findings; Schema Consistency Checker remains a consistent exception
This cycle's Sergo (#54430) and ESLint Refiner (#54441) reports both self-filed their own findings before DeepReport ran, matching the standing [[known_patterns]] lesson. Schema Consistency Checker (#54442) again did not self-file any of its 4 findings — same as every prior cycle this pattern has been checked (05:45Z cycle, this cycle) — confirming it's reliably a pure-reporting workflow whose findings always need DeepReport (or another agent) to convert into issues.

### New pattern: a docs-only "missing frontmatter.md section" issue and a "silently degrades instead of erroring" behavior bug can both stem from the same source field — file them separately by fix type
`max-turn-cache-misses` has two distinct, independently-filable gaps surfaced two cycles apart: a docs-coverage gap (already open #54179, docs-only fix) and a parser-behavior gap where expression input is silently dropped to default (filed this cycle, code fix). Lesson: when a field recurs across cycles with different symptom types (missing docs vs. wrong runtime behavior), check whether the existing open issue's scope actually covers the new symptom before assuming it's a duplicate — a docs fix and a behavior fix are different work items even for the same field name.
