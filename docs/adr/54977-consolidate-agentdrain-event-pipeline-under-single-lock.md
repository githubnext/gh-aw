# ADR-54977: Consolidate agentdrain Event Pipeline Under a Single Write Lock

**Date**: 2026-08-23
**Status**: Draft
**Deciders**: Unknown

---

### Context

`Miner.AnalyzeEvent` in the `agentdrain` package duplicated the event-preparation work — `FlattenEvent`, `Masker.Mask`, and `Tokenize` — once for pre-inference lookup and once inside `TrainEvent`. It also allocated a fresh `AnomalyDetector` on every call using `NewAnomalyDetector`, and required three separate lock scopes: a read lock for inference, a write lock for training, and a second read lock to fetch the updated cluster. This scattered state management made the control flow harder to reason about and introduced unnecessary contention risk in concurrent workloads.

### Decision

We will consolidate the event pipeline so that event preparation (flatten → mask → tokenize) happens exactly once per call in a new `prepare` helper, promote `AnomalyDetector` from a per-call ephemeral object to a long-lived field on `Miner` (initialized in `NewMiner` and refreshed in `LoadJSON`), and execute inference, training, and anomaly analysis sequentially under a single write lock. We will also relocate token-algebra helpers (`computeSimilarity`, `mergeTemplate`, `extractParams`) into a new `template.go` file to separate concerns within the package.

### Alternatives Considered

#### Alternative 1: Keep separate lock scopes but extract shared preparation

Introduce the `prepare` helper to eliminate duplicate flatten/mask/tokenize, but retain the existing pattern of an `RLock` for inference, a write lock for training, and a second `RLock` for cluster fetch. This would reduce redundant CPU work while preserving read-concurrency for the inference step.

Not chosen because the three-lock dance still produces complex, error-prone sequencing; the read-concurrency benefit only applies when many goroutines call `AnalyzeEvent` simultaneously with no writes, which is not the primary workload. The simpler single-lock approach is easier to audit for correctness.

#### Alternative 2: Split the public API into separate Infer and Train methods

Expose two public methods — one that performs inference only (under a read lock) and one that trains — letting callers decide when to call each.

Not chosen because it breaks backward compatibility with all existing call sites that expect `AnalyzeEvent` to be atomic; it also pushes the responsibility for sequencing inference before training onto every caller, making the invariant easy to violate.

### Consequences

#### Positive
- Event flatten/mask/tokenize runs exactly once per `AnalyzeEvent` or `TrainEvent` call, eliminating redundant CPU work.
- All state mutations under `AnalyzeEvent` are protected by a single write lock, making the critical section linear and easier to verify correct.
- `AnomalyDetector` threshold validation occurs at `NewMiner` and `LoadJSON` time — invalid configurations are rejected early rather than silently propagated per call.
- Template algebra is co-located in `template.go` alongside `Tokenize`, improving package cohesion.

#### Negative
- The single write lock now covers the duration of inference lookup, training, and anomaly analysis, rather than using a read lock for the inference step; under high read concurrency this is a potential throughput regression.
- Callers that previously could observe intermediate cluster state between inference and training can no longer do so; the combined lock makes the operation more atomic, which may mask timing-dependent bugs during debugging.

#### Neutral
- `trainTokens` is extracted as an unexported method that assumes `m.mu` is already held; callers (both `Train` and `TrainEvent`) must acquire the lock before calling it. This is a new documented lock-acquisition invariant within the package.
- The `template.go` file is a new file in the package; build and test tooling is unaffected.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
