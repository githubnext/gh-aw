# ADR-69225: Ignore Unexecuted Workflow Runs in Health Sampling

**Date**: 2026-10-15
**Status**: Proposed (PR #69225 is not yet accepted)
**Deciders**: Not recorded; pending maintainer review of PR #69225. This proposal does not establish approval.
**Author**: pelikhan (PR author)

## Context

`gh aw health` samples recent runs for the current workflow revision within a resolved time window. Runs that never execute can occupy the sample before older executions are reached, and a failed run with no jobs can look like an execution failure. Skipped and approval-gated runs therefore distort execution-health results if they count toward either the sample limit or failure metrics.

Filtering only the first API page is insufficient: an arbitrarily long sequence of ignored runs can hide eligible executions. GitHub's filtered workflow-run API also limits a search window to 1,000 results. Health must search farther back without crossing explicit date bounds, claiming exhaustive coverage, losing runs at timestamp boundaries, or treating an unavailable jobs response as evidence that no jobs ran.

## Decision

The proposal in PR #69225 is to apply one eligibility policy before consuming the per-workflow sample quota and when calculating health metrics:

- Ignore skipped runs and runs explicitly marked `action_required`.
- Treat `waiting` as ambiguous. Ignore it only after a successful jobs query confirms that the run attempt has no jobs; retain waiting runs with jobs as pending work.
- Ignore failed runs only when a successful jobs query confirms zero jobs. A failed run with jobs remains eligible, even when an early job failed before the agent started.
- Keep ignored runs out of execution-health totals, failure rates, and the requested eligible-run count. Continue applying current-revision selection rather than mixing historical revisions to fill the sample.

Fetch additional pages until enough eligible runs are collected, the available history within the resolved time window is exhausted, or a documented API/search bound prevents further progress. Keep the resolved start and end bounds fixed for the collection.

When a search window reaches the 1,000-result limit, continue with an older window that overlaps the oldest returned timestamp. Deduplicate runs across the overlap so a run cannot consume the quota twice. If the boundary timestamp is itself saturated and the search cannot advance without potentially omitting runs, stop and report partial coverage rather than silently skipping that timestamp or looping.

Cache successful jobs-presence results by run ID and run attempt during collection, so overlapping windows do not repeat successful probes or reuse evidence from a different attempt. Jobs API errors must remain explicit collection errors or partial-coverage diagnostics, not become cached zero-job results. Neither filtering nor sampling authorizes reruns or bypasses approval gates.

This approach favors execution evidence over interpreting ambiguous status names, while keeping collection bounded and its uncertainty visible.

## Alternatives Considered

### Count every returned run

This preserves the simplest pagination and existing status-based totals, but measures scheduling and approval activity as execution health. Recent skipped or unexecuted runs can displace the executions the command is intended to assess.

### Filter only the initial page or increase its fixed size

This reduces API calls but leaves the result dependent on how many ignored runs happen to precede eligible runs. Increasing a fixed fetch limit does not address the 1,000-result search-window ceiling.

### Ignore every waiting or failed run without querying jobs

This avoids extra API requests but hides genuine pending executions and real failures. Conversely, querying jobs for every run adds avoidable latency; probe only cases requiring execution evidence and cache successful results.

### Exhaustively enumerate history or skip overlapping timestamps

Unbounded enumeration conflicts with the explicit time window and bounded sample contract. Advancing past a saturated timestamp makes progress but can silently omit eligible runs; explicit partial coverage is preferable to an unsupported completeness claim.

## Consequences

Eligible execution samples are less sensitive to approval and scheduling noise, and confirmed zero-job failures no longer inflate execution-failure rates. Genuine failures and waiting runs with jobs remain visible.

Sampling may require more API requests and latency when many runs are ignored. Attempt-scoped caching limits repeated probes, but permission failures and API errors still require visible diagnostics. Timestamp saturation can leave a sample incomplete, and reports must preserve that limitation rather than imply healthy or exhaustive results.

These changes alter the meaning of health totals without changing workflow execution, approval policy, or the resolved time window. Regression coverage must distinguish ambiguous waiting states, failed runs with and without jobs, probe errors, overlapping windows, rerun attempts, and saturated timestamp boundaries.
