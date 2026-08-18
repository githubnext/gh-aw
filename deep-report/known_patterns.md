## DeepReport Memory (2026-08-18T06:23:00Z)

### New lesson: an auto-expired "not_planned" closure is not evidence a problem was fixed — always re-check live state
Issue #50515 (compiler_safe_outputs_job.go decomposition) auto-expired 2026-08-06 with `state_reason: not_planned` and zero fix PRs in its timeline. Six weeks later, an independent compiler-quality report rediscovered the *exact same* 144-line function, unchanged, at the same file/line. **Lesson: before treating a closed issue as "handled" when it comes up again in a new report, check `state_reason` and the timeline for an actual merged fix — an auto-expiry is a no-op, not a resolution.** Re-filed with this evidence attached.

### New pattern: auto-filed "[aw] Failed jobs: X" issues do not self-consolidate even when clearly duplicate
Found 16 open `[aw] Failed jobs: PR Sous Chef` issues (#53245-#53446) spanning 5 days, mostly the same `safe_outputs`-step failure, never linked or root-caused — confirmed independently by this same cycle's Issue Arborist run, which also declined to auto-link them (ambiguous root cause across excerpts). **Lesson: the failed-jobs auto-filer has no dedup/consolidation step; when a chronic per-run failure issue type recurs >5-10 times without resolution, that itself is the actionable finding (root-cause + consolidate), not something to wait out.**

## DeepReport Memory (2026-08-18T00:31:00Z)

### CONFIRMED: the day-keyed-cache fix (PR #53486) is working, and it fixed a *class* of bugs, not just deep-report's instance
This cycle's pre-fetched discussions/issues data had `updatedAt` timestamps within ~2 minutes of the live clock — no stale-cache workaround needed. But the same root cause (gating `cache-memory` reuse on an exact `${TODAY}` filename match, defeating cross-run reuse for any workflow whose cadence isn't "every run same calendar day") recurred independently in `Copilot Opt` and `Copilot Agent PR Analysis`, surfaced by the Daily Cache Strategy Analyzer (discussion #53466). **Lesson: when a root-cause fix lands for one workflow, check whether the same anti-pattern exists elsewhere in the fleet — the Daily Cache Strategy Analyzer is a good recurring detector for this specific class.** Filed as a new task this cycle.

### Reconfirmed: filed issues in this repo get fixed fast
All 4 substantive issues from the 18:23Z cycle (stale cache, CI regression, schema drift, large-file decomposition) were merged within 1–5 hours of filing. This is strong evidence the create-issue → agent-fixes-it pipeline is working well; continue prioritizing well-evidenced, narrowly-scoped filings over broad/vague ones.

### New pattern: narrow the scope of "standardize X fleet-wide" recommendations to the specific instances with measured evidence
The Daily Regulatory Report's generic recommendation ("standardize window anchoring for all 24h reports") was too broad to file as a 1-3 day task. Instead, filed the narrower, evidence-backed version: add window_start/window_end only to the two specific workflows (Daily Status, Daily Team Evolution Insights) that showed an actual measured discrepancy (50 vs 22 merged PRs). **Lesson: when a source report gives both a broad recommendation and specific supporting evidence, file the narrow evidence-backed version, not the broad one — matches the standing "code metrics inline-comments fleet-wide" precedent from 2026-08-17's decision to decline overly-broad tasks.**

### Confirmed pattern (re-verified this cycle, backlog shrinking): "label the unlabeled issues" remains a non-productive loop, but the backlog is shrinking organically
Down to 3 unlabeled open issues (#53532, #53489, #53136) from 6 as of the 12:22Z cycle — shrinking without any dedicated labeling task being filed. Continue declining to re-file this task type; it appears to resolve itself over time as issues get triaged/closed through normal workflow.

### New risk identified: the 100-entry discussions.json window causes permanent loss of unmined discussions if a cycle defers mining
At ~13-63 discussions created per 6h window, anything not mined in the cycle it appears in is likely to roll off the 100-entry cap within 1-2 days and become unrecoverable without an expensive per-number re-fetch. The 18:23Z cycle's ~55-discussion "not yet mined" backlog is now permanently lost (confirmed this cycle: only 1 of the ~55 numbers checked was still present in the dataset). **Lesson: always mine every new/updated discussion in the cycle it first appears — never defer "mine this later," since later may not have the data anymore.**

### Reconfirmed practice: verify merge status directly, don't trust search-API fields
`gh api search/issues` unreliably populates `merged_at` for PRs — always confirm via the direct `gh api repos/.../pulls/{n}` endpoint before concluding a linked fix didn't land, especially before deciding whether to file a duplicate. (Used this cycle to confirm all 4 of last cycle's fixes.)

### `agenticworkflows logs` throughput guidance
`count: 25` completed in ~39s this cycle (MCP tool call elapsed ~39992ms). Consistent with prior guidance (`count<=50` reasonable, ~1.3-1.6s/run). Good for a quick health spot-check without a full fleet audit.
