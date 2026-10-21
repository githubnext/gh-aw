# ADR-70885: Reuse Raw Cached Jobs Across CLI Versions and Retain Typed Job Evidence in `gh aw health`

**Date**: 2026-07-14
**Status**: Draft
**Deciders**: pelikhan (PR author); [TODO: verify reviewers]

---

### Context

`gh aw health` groups failing runs into shared job/step clusters so that one investigation can cover failures across many workflows. It gets its evidence from two places: run summaries already cached in the logs directory, and inexpensive job-presence checks against the GitHub Jobs API. Before this PR, any cached summary whose producer CLI version differed from the current version was discarded completely. The live presence check also kept only a boolean, not the job and step details. As a result, a real incident (a shared failed activation step across about a dozen workflows) showed up as separate per-workflow clusters, and investigation targets had no cached log paths. The command must stay read-only and cheap: no artifact downloads, no workflow dispatch, and bounded API usage.

### Decision

We will treat cached Jobs API data (`JobDetails`) as raw observations that can be reused across CLI versions after an identity check. The command will still reject version-dependent derived data (usage, classification and sub-agent metrics) from a different producer, and will never rewrite the cached producer metadata. Presence checks will keep typed job/step evidence for each run attempt. Only failure-first representatives may expand that evidence, by at most three 100-job pages per attempt (`healthMaximumJobPages = 3`). The report will state the provenance and completeness of this evidence: producer version, report schema version, capabilities, per-run cache compatibility status, job-evidence completeness and API/job request counts. The main reason is to recover cross-workflow failure clustering from evidence the tool already has, while keeping cost bounded and making the trustworthiness of each observation visible.

### Alternatives Considered

#### Alternative 1: Keep strict CLI-version equality for all cached data

Discard any cached summary produced by a different CLI version, as before. This is the simplest and safest choice against schema drift. It was rejected because it threw away valid raw API observations after every CLI upgrade. That loss caused the missed shared-step cluster that motivated this PR.

#### Alternative 2: Re-fetch all jobs (or download artifacts) for every failing run

Ignore the cache and fetch complete job lists, or download run artifacts, for every failing run. This would give full and current evidence. It was rejected because the health command must not download artifacts, and fetching every job for every run makes API cost grow with fleet size instead of with the number of representatives.

#### Alternative 3: Reuse entire cached summaries, including derived metrics, across versions

Treat the whole cached `RunSummary` as valid no matter which version produced it. This is the cheapest option, but it would silently mix derived metrics computed under different classification or usage logic. It was rejected in favour of the "raw jobs only" (`version_mismatch_raw_jobs_only`) split.

### Consequences

#### Positive
- Failures in a shared job/step are again clustered across workflows. A regression test shows twelve workflows collapsing into one failed-step boundary without downloading artifacts.
- Investigation representatives get back their cached log paths, plus branch, event, head SHA, attempt and update time.
- Readers can judge the evidence themselves from `cache_evidence`, `job_evidence`, `capabilities`, `report_schema_version` and the `api_requests`/`job_requests` counts.
- A failing revision above the success threshold is no longer counted as healthy. New `status_counts` and `observed_failure_workflows` fields expose this.

#### Negative
- The command now assumes the raw `JobDetails` shape stays compatible across CLI versions. If that shape changes, the identity check and the "raw jobs only" rule will need revisiting.
- Representatives with more than 300 jobs get incomplete job evidence (reported as `complete: false`), so a very large run could still be missed in clustering.
- The JSON output (`HealthCollection`, health summary) grows, and consumers now see two failure-related counters. `degraded_workflows` keeps its existing sub-agent-only meaning, which users could confuse with `observed_failure_workflows`.
- Health collection must carry more in-memory state (`jobs`, `cacheSummaries` keyed by run attempt).

#### Neutral
- `report_schema_version: 1` is introduced, which sets up future schema versioning of the health report.
- CLI/MCP guidance and `docs/src/content/docs/setup/cli.md` were updated to describe cache compatibility reporting.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
