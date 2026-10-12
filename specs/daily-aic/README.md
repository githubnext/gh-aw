# Daily AI Credits computation: executable TLA+ specification

`DailyAIC.tla` is the bounded decision-state model for Section 9 of the
[AI Credits specification](../../docs/src/content/docs/specs/ai-credits-specification.md).
It covers the artifact backend and the distinct repo-memory backend, including
all currently implemented accounting fallback categories.

## Model and assumptions

The model begins with a resolved configuration, then models bypass/trust checks,
snapshot restore, current-workflow lookup, complete historical membership,
per-run resolution, persistence of individually resolved observations, and the
final threshold comparison. Current/history/restore read failures cannot use
historical per-run estimates. Workflow-history 404 can switch to repository
history. A failed historical read can replace the whole run with a positive
configured maximum, but not through permission, quota, or retry restrictions.
Transient reads have at most three attempts.

Recorded cache entries require trusted producer provenance and matching
completed-run metadata. Estimated, legacy, expired, invalid, and mismatched
observations are not cache hits. Producer cases include default-branch `issues`
and `issue_comment`, existing trusted events, and separate branch, fork,
workflow, path, repository, and missing-default-branch rejection cases.

For each modeled run, Agent contributes its abstract run ID and Detection and
Evaluations each contribute one abstract AIC unit. Valid source alternatives
replace their component, not add overlapping summaries. Zero-policy cases
replace only the affected component; all-skipped/no-billable-job and explicit
all-zero records produce whole-run zero. Missing, unreadable, malformed,
unknown-pricing, stale-coverage, conflicting-request, and unverifiable-rerun
outcomes cannot be recorded as measured usage. The empty-Detection case ignores
a lower-priority summary; a negative control deliberately resurrects it.

The accounting-kind labels abstract the evidence conditions documented in
Sections 9.11-9.12. They do not themselves parse JSON or inspect logs. In
particular, legacy missing-accounting heuristics are modeled as existing policy,
not as proof that the provider charged nothing.

Ledger samples include empty/missing/unreadable input, duplicate run IDs,
equal-timestamp replacement, invalid records, old/future records, and inclusive
window boundaries. The greatest valid timestamp, then later sequence index,
wins. Ledger trust or an explicit insecure opt-in is required; artifact history
and maximum-credit repair are not used. An empty ledger can permit continuation.

`WF_vars(Next)` excludes indefinite scheduler stuttering before a terminal
decision. This is a fairness assumption, not a guarantee about Actions
scheduling or network availability. Terminal states stutter explicitly.

## Checked properties

| Property | Obligation |
|---|---|
| `TypeOK` | Valid states, sources, non-negative values, and attempt counters |
| `CompleteWindowRequired` | Artifact decisions require complete authoritative membership and all candidates counted |
| `TotalsReconcile` | Each counted run contributes once, including legitimate zero values |
| `EstimatesReconcile`, `WholeRunMaximum` | Estimates are reported separately and substitute exactly the positive configured maximum |
| `NoForbiddenEstimates` | Quota, permission, and retry restrictions cannot become estimates |
| `RecordedCacheOnly`, `CacheProducerTrusted` | Estimates are reinspected and cache hits require correct producer provenance |
| `RecordedEvidence`, `ZeroEvidence` | Measured/zero values follow the supported accounting policies |
| `ThresholdDecision` | Strictly below permits; equality and above exceed |
| `UnknownNeverUnderBudget` | A stopped scan is never a successful budget decision |
| `BypassDoesNotRead` | Disabled or bypassed checks do not account or issue reads |
| `ReadAttemptBound` | At most three attempts per read |
| `LedgerTrust`, `LedgerReconcile` | Separate ledger trust, filtering, deduplication and summation rules |
| `ContinuationPolicy` | Unknown totals permit only with explicit `continue-on-error`; exceeded totals never permit |
| `EventuallyDone` | Every modeled invocation terminates under weak fairness |

## Validate

Use Node.js, Java, and the official
[TLA+ Tools v1.7.4 JAR](https://github.com/tlaplus/tlaplus/releases/download/v1.7.4/tla2tools.jar).
The checker requires SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
It does not download tools or install dependencies.

```sh
JAVA_BIN=/absolute/path/to/java \
TLA2TOOLS_JAR=/absolute/path/to/tla2tools.jar \
node specs/daily-aic/check.mjs /absolute/path/to/validation-output
```

Omit the output argument to use a new temporary directory. SANY must successfully
parse and semantically check the copied model. Each TLC process uses two workers,
a 1 GiB heap, seed 1, and a bounded timeout.

`DailyAIC.cfg` checks every source/evidence/producer/cache category with one
historical run. `Aggregation.cfg` checks two historical runs with a smaller
source/producer/cache universe, including mixed recorded/estimated accounting,
partial snapshots, zero values, and duplicate-free aggregation. Both explore
all declared configurations, transport failures, and at most three read attempts.
These are exhaustive checks of their finite models, not simulation.

Successful positive checks require exit zero, TLC's exhaustive-success verdict,
nonzero explored states, and an empty remaining queue. Six negative controls
must return exit 12 and violate their exact named invariant: incomplete history,
quota estimation, estimated cache reuse, non-default issue-producer trust,
double counting, and resurrected Detection summary usage. Syntax failures,
timeouts, and unrelated invariant failures do not count as successful controls.

Forty-seven intentional `WitnessNotReached` counterexamples demonstrate
reachability of every named accounting outcome and the main decision/fallback
paths, including partial persistence, mixed sources, cache reinspection, quota
stops, missing maxima, ledger, history fallback, and retry recovery.

The output contains copied model/configurations, raw SANY/TLC logs and
counterexamples, and `report.json` recording source hashes, revision anchor,
tool checksum/version, commands, verdicts, and state counts. Hashes identify
working-tree bytes; the HEAD anchor does not imply those bytes were committed.
Changes to measured source bytes or HEAD during validation fail the checker.
Reusing an output directory marks its report incomplete before starting; only
successful completion changes the report to passed.

The checked configurations completed with 557,473 distinct states for the
one-run policy model and 100,890 for the two-run aggregation model, with no
remaining queued states. All safety invariants and `EventuallyDone` passed;
all six negative controls and 47 reachability witnesses produced the required
named counterexamples.

## Implementation boundary

The corresponding native suites are `daily_aic_scan`,
`check_daily_aic_workflow_guardrail`, `daily_aic_api_budget`,
`daily_aic_cache_helpers`, `daily_aic_component_coverage`,
`daily_aic_workflow_helpers`, and `daily_aic_repo_memory_ledger` under
`actions/setup/js/`.

The model abstracts timestamp/identity validity, producer authenticity, API
ordering and pagination into evidence facts; it does not prove their parsers.
It does not execute or refine the JavaScript implementation, floating-point
pricing arithmetic, legacy-log regular expressions, actual quota headers,
fresh timeout-signal construction, durable artifact/ledger delivery, concurrent
invocations, or provider billing. It checks latest metadata rather than summing
every rerun attempt. No arbitrary-history, actual-charge upper-bound, TLAPS,
or unbounded liveness claim is made. Summary-rendering errors after accounting
are outside the model.
