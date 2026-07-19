# ADR-46554: Dynamic Timeout Computation for the MCP Logs Tool

**Date**: 2026-07-19
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The `agenticworkflows logs` MCP tool timed out (hitting the MCP gateway's 60-second hard limit) whenever it was called without a `--workflow_name` filter. Three compounding issues caused this: (1) `BatchSizeForAllWorkflows` was set to 250, which paginated as three sequential GitHub API calls per batch (the GitHub API caps per-page results at 100), producing 45–60+ seconds of latency on large repositories; (2) `fetchWorkflowRunBatch` called the GitHub API with `Context: nil`, meaning the subprocess had no deadline and could not be cancelled by the caller's context; (3) a static `timeout` schema default was registered in the MCP tool schema, causing the go-sdk to pre-fill `args.Timeout` before the handler ran, bypassing the per-request runtime computation that would have applied a higher floor for all-workflow queries.

### Decision

We will compute the MCP logs tool timeout dynamically at request time based on two parameters: the effective `count` and whether `workflowName` is empty. When no workflow filter is provided, a minimum floor of 5 minutes (`defaultMCPLogsMinTimeoutMinutesAllWorkflows`) is enforced, because unfiltered GitHub API queries scan all workflow runs and are significantly slower on large repositories. The static MCP schema default for `timeout` is removed so the go-sdk cannot pre-fill the value and short-circuit the runtime computation. Additionally, `BatchSizeForAllWorkflows` is reduced from 250 to 100 (the GitHub API's `per_page` maximum) so each batch requires only a single API round-trip, and the request context is threaded into `fetchWorkflowRunBatch` to allow graceful cancellation distinguishing internal deadline expiry from external context cancellation.

### Alternatives Considered

#### Alternative 1: Increase or Remove the MCP Gateway Timeout Limit

Configure the MCP gateway to allow a longer (or unlimited) per-tool timeout so that the existing logic with `BatchSizeForAllWorkflows = 250` and no context propagation could still succeed on large repositories. This was not chosen because the 60-second limit is an external constraint of the MCP gateway infrastructure that the tool cannot unilaterally change. Relying on a large external timeout also hides performance problems rather than fixing them; a slow tool would remain slow for users even if it eventually completed.

#### Alternative 2: Parallelise GitHub API Calls Within a Batch

Instead of reducing `BatchSizeForAllWorkflows` from 250 to 100, keep the larger batch and issue the three required API pages concurrently. This would reduce the wall-clock time for a 250-run batch to approximately the latency of one API call. This was not chosen because it adds concurrency complexity and risk (rate-limiting, partial-failure handling) to code that was otherwise sequential by design. Aligning with the API's per-page maximum of 100 is simpler, predictable, and sufficient to keep individual batch fetches within acceptable latency bounds.

#### Alternative 3: Cache Workflow Run Listings Between Requests

Introduce a short-lived cache of recent `gh run list` results so that consecutive calls (e.g. from retried or paginated MCP requests) avoid redundant API round-trips. This would reduce latency for repeated queries at the cost of potentially serving stale data. This was not chosen because it introduces state management complexity and cache invalidation risk in a tool that needs to return current data; the simpler approach of aligning batch size with API limits is sufficient for the latency problem at hand.

### Consequences

#### Positive
- The MCP logs tool no longer times out on large repositories when `--workflow_name` is omitted, making the common fleet-wide log inspection use-case reliable.
- The distinction between internal deadline expiry (`context.DeadlineExceeded`) and external cancellation (`context.Canceled`) is now surfaced to callers, enabling better error handling and partial-result recovery.
- Removing the static schema default makes the timeout computation self-consistent: the value callers see in the MCP schema (no default) matches the actual runtime behaviour (computed per-request).

#### Negative
- Reducing `BatchSizeForAllWorkflows` from 250 to 100 means that retrieving a large number of runs requires more loop iterations, which slightly increases total latency when the repository has many workflow runs and the result count is large.
- Removing the static schema default for `timeout` means MCP clients that relied on schema introspection to determine a default timeout value will no longer find one; they must accept that the timeout is opaque and server-determined.

#### Neutral
- The `effectiveMCPLogsToolTimeoutMinutes` function signature now accepts a `workflowName string` parameter; all call sites must be updated accordingly (done in this PR).
- Tests for timeout computation are extended with a `workflowName` dimension, increasing test coverage but also test verbosity.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
