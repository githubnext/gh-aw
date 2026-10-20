# ADR-70552: Failure-First, Source-Aware Workflow Health Collection

**Date**: 2026-10-20
**Status**: Draft
**Deciders**: pelikhan (PR author); [TODO: verify reviewers]

---

### Context

`gh aw health` samples recent runs for each registered agentic workflow under a shared GitHub API budget (about 1,000 requests). The earlier collector walked each workflow's history chronologically. It spent much of the budget on successful runs before reaching failures, so it often missed failures that maintainers needed to act on. It also compared revisions only through lock-file `gh-aw-metadata` hashes. If someone pushed a Markdown source edit without recompiling the lock file, old failures still counted as current. Health reports then sent investigators to failures that a later source edit had already superseded. Live checks over the same window showed that collection order strongly affected how many actionable failures appeared before the budget ran out.

### Decision

We will collect health evidence in two passes and track workflow source revisions:

1. **Failure-first pass**: A single repository-wide pass (`collectHealthFailures`, using `actions/runs?status=failure`) collects registered AW runs whose conclusion is explicitly `failure`. The pass checks that each run has jobs and keeps up to five failures for each workflow/branch/event scope.
2. **Chronological pass**: The existing per-workflow sampling runs next. It visits workflows with recent failures first (`prioritizeHealthFailureWorkflows`) and deduplicates runs it already collected (`mergeHealthRunSamples`).

For each supported run, we will compare the Markdown source's blob SHA at the run's head SHA with its blob SHA on the pushed branch. When they differ, collection stops for older history in that branch/event scope, and the superseded failures are excluded from current clusters and investigation targets. The report exposes `collection.failure_pass_complete` and `source_sha` on revision cohorts so readers can see how complete the sample is and which source each cohort used.

### Alternatives Considered

#### Alternative 1: Keep chronological per-workflow sampling and raise the API budget

The collector would keep its current order and simply be allowed more requests. This is simple and keeps the sample unbiased. We rejected it because the cost grows with fleet size, rate limits still apply, and successful runs would still use most of the requests before failures appeared.

#### Alternative 2: Detect superseded revisions from lock-file metadata hashes only

The collector would keep the existing `frontmatter_hash`/`body_hash` comparison from the compiled `.lock.yml`. This needs no extra API calls. We rejected it because source edits often land before recompilation, so failures from an old source would still look current. Comparing Markdown blob SHAs catches these edits for one extra contents lookup per ref.

#### Alternative 3: Stop collection on any repository commit after a run

The collector would treat any newer commit on the branch as a revision boundary. This is cheap, but unrelated commits would stop sampling too early and hide real current failures. We chose per-workflow source blob comparison, which ignores unrelated repository changes.

### Consequences

#### Positive
- Under the same budget, more actionable failures appear: live checks found 97 failed runs and 46 clusters, compared with 60 runs and 36 clusters before, and took about 7 minutes instead of about 12.5 minutes.
- Investigators no longer chase failures from superseded sources. A pushed source with no matching runs is reported as `no_current_revision_runs` (awaiting evidence), not as healthy or fixed.
- `failure_pass_complete` and `source_sha` show how the sample was collected and which source each cohort used.

#### Negative
- The sample is deliberately biased toward failures, so it is not a fleet-wide failure rate. In the live checks, overall workflow coverage dropped (182 workflows uncollected, compared with 147).
- Source lookups add contents-API requests per run and per branch, which takes budget away from sampling.
- Collection now depends on a `.lock.yml` → `.md` path convention. Unavailable source revisions produce warnings and unknown revision evidence.

#### Neutral
- `HealthCollection` adds a public JSON field (`failure_pass_complete`), and `HealthRevisionCohort` adds `source_sha`. The text output of health revisions now includes `source_sha`.
- The collector only treats a run's head SHA as its source revision for events that run from that SHA (push, schedule, dispatch, issues, discussions, and similar). For other events, revision evidence remains unknown.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
