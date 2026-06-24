# ADR-41185: Parallelize audit analysis tasks with goroutines and `sync.WaitGroup`

**Date**: 2026-06-24
**Status**: Draft

## Context

`AuditWorkflowRun` (`pkg/cli/audit.go`) drives a workflow-run audit by invoking roughly fifteen independent analysis operations — log-metric extraction, job-status/detail fetches, missing-tool/missing-data/noop/MCP-failure extraction, access-log and redacted-domain analysis, GitHub rate-limit analysis, firewall log/policy analysis, MCP tool-usage and token-usage analysis, artifact listing, and created-item extraction. These ran as a strictly sequential chain, so total audit wall-clock time was the *sum* of every operation and scaled linearly with log volume. A 32-turn, ~12-minute "Static Analysis Report" run took ~72 s to audit — 2.4× over the 30 s performance target. Most of these operations are I/O-bound (reading downloaded artifact files or calling the GitHub API) and write to distinct result variables, making them candidates for concurrent execution.

## Decision

We will replace the sequential analysis chain with concurrent execution using `sync.WaitGroup` (`wg.Go`), launching each independent analysis task in its own goroutine and aggregating results after a single `wg.Wait()`. Each goroutine writes only to its own pre-declared local variable, and the main goroutine reads those variables only after the barrier, so no shared-memory synchronization beyond the `WaitGroup` is required. Fields the tasks read from `run` (`LogsPath`, `Duration`, and the `hasFirewallArtifact` predicate) are computed *before* any goroutine is spawned so every task observes a fully-initialised `run` value. Two ordering constraints are preserved by keeping the dependent calls sequential *within* a single goroutine: `fetchJobStatuses` + `fetchJobDetails` (same API endpoint, avoid duplicate concurrent requests) and `analyzeFirewallLogs` → `extractFirewallFromAgentLog` merge (the agent-log result is merged into the firewall-log result). Wall-clock audit time is now bounded by the slowest single task rather than the sum of all tasks.

## Alternatives Considered

### Alternative 1: Keep the sequential chain and optimize the dominant task

`extractLogMetrics` is typically the single largest cost for large runs, so we could have left the structure sequential and only sped up that one function (e.g. streaming parse, caching). This is lower-risk — no concurrency hazards — but it only addresses one term of a fifteen-term sum; the remaining I/O-bound calls still serialize, and any future analysis added to the chain re-grows the linear cost. Rejected because it does not structurally bound total audit time and would not reliably bring the 72 s case under the 30 s target.

### Alternative 2: Bounded worker pool / `errgroup` with a concurrency cap

We could route the tasks through `golang.org/x/sync/errgroup` (with `SetLimit`) or a hand-rolled worker pool to cap simultaneous goroutines and propagate the first error. This adds first-class error aggregation and limits peak file-descriptor / API pressure. It was rejected for this change because the task count is small and fixed (~15), each task already degrades gracefully on error (logging a warning and falling back to a zero value rather than aborting the audit), so there is no error to propagate to a caller, and an unbounded `WaitGroup` keeps the control flow closest to the original code. A concurrency cap can be layered on later if descriptor or rate-limit pressure proves problematic. *(See Negative consequences.)*

## Consequences

### Positive

- Audit wall-clock time is bounded by the slowest individual task instead of the sum of all tasks, bringing the previously-72 s complex-run case within the 30 s target.
- Adding a new independent analysis task no longer linearly increases audit latency — it overlaps with existing work as long as it writes to its own variable.
- Behaviour and per-task error handling are preserved: each goroutine retains the original verbose-mode warnings and `auditLog.Printf` diagnostics and the same zero-value fallback on failure.

### Negative

- Concurrency introduces data-race risk that the sequential version was immune to. Correctness now depends on the invariant "each goroutine writes a distinct variable and `run` is read-only after spawn"; a future edit that has a task mutate shared state (e.g. `run` or a shared map) would reintroduce a race that the type checker will not catch. The `-race` test build is the safety net.
- Up to ~15 analysis tasks (file reads plus GitHub API calls) now run simultaneously with no concurrency cap, increasing peak file-descriptor usage and the chance of GitHub API rate-limit/secondary-limit pressure under many concurrent audits.

### Neutral

- Result-variable declarations are hoisted into a single `var (...)` block and assembled into `run`/`processedRun` after `wg.Wait()`, so the assembly point moved but the report contents are unchanged.
- Two task groups remain deliberately sequential inside their goroutine (job status→details, firewall logs→agent-log merge); they do not benefit from the parallelism but also do not block the other tasks.
- Non-deterministic completion order means interleaving of verbose stderr warning lines can vary between runs; the structured report output is unaffected.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/28104198322) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
