# ADR-43447: In-Loop Rate-Limit Re-Check for AIC Guardrail and Wider Cache Fallback Search Window

**Date**: 2026-07-05
**Status**: Draft
**Deciders**: Unknown (bot-authored PR by copilot-swe-agent)

---

### Context

The daily AIC (AI Credits) guardrail inspects recent workflow runs to sum consumed credits, fetching each run's usage artifact when a cache miss occurs. Under high run frequency with a cold cache, N concurrent job activations each independently compute an upfront budget (`computeMaxInspectableRuns`) from a single rate-limit snapshot taken at startup. Because all concurrent activations read the same stale snapshot simultaneously, they each believe sufficient headroom exists and collectively exhaust the shared `GITHUB_TOKEN` core rate limit. In production this pattern accounted for 99.5% of all `/actions/runs/:run_id/artifacts` API calls in a single hour, producing HTTP 403 responses that starved unrelated critical steps such as attestation persistence. A separate issue was identified on `pull_request` branches: the artifact fallback search window of 5 runs was too narrow to find a usable cache artifact on active repositories, because `actions/cache` is branch-scoped and routinely misses on PR branches.

### Decision

We will add a periodic in-loop rate-limit re-check (`RATE_LIMIT_RECHECK_INTERVAL = 10` consumed API operations, firing after every 5 cache-miss runs at `ESTIMATED_API_OPERATIONS_PER_RUN = 2`) that reads the **live** shared quota and breaks out of the inspection loop immediately if `remaining ≤ RATE_LIMIT_RESERVE`. This supplements—but does not replace—the existing upfront budget calculation, which serves as a first-pass gate. We will also widen the artifact fallback search window from 5 to 30 recent workflow runs to improve cache-hit rates on `pull_request` branches without requiring an additional API page.

### Alternatives Considered

#### Alternative 1: Distributed Rate-Limit Coordination (Centralized Lock/Semaphore)

A shared lease or semaphore (e.g., via a database, Redis, or GitHub Environments protection rules) could serialise concurrent activations so only one job draws from the rate-limit budget at a time. This would completely prevent collective overspend. It was not chosen because it requires additional infrastructure or state external to GitHub Actions, adds latency waiting to acquire the lock, and introduces a new failure mode (lock contention or deadlock) in an already-critical compliance gate.

#### Alternative 2: More Conservative Upfront Budget (Reduce `computeMaxInspectableRuns`)

Dividing the upfront budget by an estimated concurrency factor (e.g., always assume 5 concurrent jobs) would reduce per-activation spend without any runtime coordination. This was not chosen because the divisor is a hardcoded guess that either under-inspects runs during low-concurrency periods (wasting available budget) or still overspends during burst periods if concurrency exceeds the assumed value. A live re-check adapts to the actual shared state regardless of concurrency level.

#### Alternative 3: Smaller Fallback Search Window with Cache Warming

Keeping `MAX_RUNS_TO_SEARCH = 5` but adding a dedicated cache-warming step for PR branches was considered. This was not chosen because it would require additional workflow changes and a new cache upload step, adding complexity. Widening the existing window to 30 runs fits within a single `listWorkflowRuns` page at no additional API cost.

### Consequences

#### Positive
- Concurrent activations stop consuming API budget as soon as the shared reserve is actually depleted, preventing cascading HTTP 403 errors on unrelated steps.
- No additional infrastructure or external state is required; the solution is entirely self-contained within the existing guardrail script.
- Cache-hit rate improves for `pull_request` branches where `actions/cache` misses are frequent, reducing per-activation API costs on active repositories.
- The fix is covered by a new deterministic unit test that mocks the rate-limit drop and asserts the exact number of runs inspected before the loop stops.

#### Negative
- Each periodic re-check consumes one additional API call per `RATE_LIMIT_RECHECK_INTERVAL` operations (approximately 1 extra call per 5 cache-miss runs), adding modest overhead that was not present before.
- `RATE_LIMIT_RECHECK_INTERVAL` is a hardcoded constant. If `ESTIMATED_API_OPERATIONS_PER_RUN` changes in the future, the effective re-check frequency changes silently and may need manual adjustment.
- The in-loop re-check only fires after operations are already consumed; it cannot prevent the first `RATE_LIMIT_RECHECK_INTERVAL` operations in a burst from contributing to overspend.

#### Neutral
- The wider fallback window (5 → 30) does not add new API pages; `listWorkflowRuns` already returns up to 100 runs per page, so this is a client-side filter change only.
- The upfront budget calculation (`computeMaxInspectableRuns`) is preserved unchanged as a first-pass gate; this ADR adds a supplemental mechanism, not a replacement.
- Existing behaviour for cache-hit runs is unaffected: `apiCallsInLoop` is only incremented on cache misses, so fully-warm caches incur zero extra re-check calls.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
