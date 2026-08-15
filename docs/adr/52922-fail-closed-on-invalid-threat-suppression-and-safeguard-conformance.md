# ADR-52922: Fail Closed on Invalid Threat Suppression and Enforce Safeguard Conformance via Formal Tests

**Date**: 2026-08-15
**Status**: Accepted
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The AWF config and compiler threat-detection specifications defined safeguards — including snapshot expiry, suppression validation, optimizer failure diagnostics, and forecast fixture traceability — without any corresponding conformance test coverage. The compiler previously accepted `threat-detection-suppress` frontmatter annotations without schema validation, meaning invalid or expired suppression entries could silently reach the compiled lock file and suppress real threat-detection rules. The daily compiler threat spec optimizer also ran only weekly, creating a coverage frequency gap against the spec's intent. Together, these gaps meant the system's stated safety invariants could be violated without any automated detection.

### Decision

We will enforce `threat-detection-suppress` entries through a validated JSON schema (added to `main_workflow_schema.json`) and carry all validated suppressions into the compiled lock-file manifest. Retaining expired entries keeps compilation deterministic and preserves the complete audit record; runtime and audit consumers determine whether a suppression is active. The compiler (`generateWorkflowHeader`) now returns an error when suppression data is invalid, failing closed at compile time rather than producing a potentially unsafe lock file. We will back every spec safeguard (snapshot expiry, degraded-run marking, retrieval warning completeness, scheduled persistence threshold, and all T-CTR-024–T-CTR-038 threat-optimizer protocol requirements) with formal conformance tests so that the spec's guarantees are machine-verifiable. The threat optimizer schedule is corrected from weekly to daily to match the spec's coverage-frequency requirement.

### Alternatives Considered

#### Alternative 1: Runtime-only validation in the agent

Validate `threat-detection-suppress` entries only at agent execution time, not at compile time, and leave the lock file unaffected. This avoids changing the compiler's error surface. It was rejected because the compiler is the authoritative gating point for what reaches the lock file; allowing invalid suppression entries into the lock file creates an audit trail that falsely implies reviewed suppressions were in effect.

#### Alternative 2: Documentation-only coverage without formal tests

Update the spec documents to state the safeguard requirements without adding Go conformance tests. This was rejected because the prior coverage gaps were discovered precisely because specifications were not backed by tests; a documented-but-untested invariant provides no automated enforcement and drifts silently.

### Consequences

#### Positive
- The compiler fails closed on invalid suppression configuration: a workflow with a malformed `threat-detection-suppress` entry cannot be compiled to a lock file.
- All validated suppressions are persisted in the lock-file manifest, creating a deterministic audit record without wall-clock-dependent lock-file churn.
- Formal conformance tests (T-DR-SAFE-001 through T-DR-SAFE-004, T-CTR-024 through T-CTR-038) make all spec safeguard guarantees machine-verifiable and regression-proof.
- The optimizer runs daily, closing the coverage-frequency gap with the spec.

#### Negative
- `generateWorkflowHeader` now returns an error, which is a behavioral change for all callers; callers that previously ignored the return value (treating it as infallible) must now handle the error or propagate it.
- Adding `threat-detection-suppress` to the JSON schema is a breaking validation change: existing workflow files that contain malformed suppression entries (e.g., missing `reason`, bad `rule` pattern, non-ISO-8601 `expires`) will fail compilation after this change.

#### Neutral
- The snapshot expiry and retrieval-warning logic in `schema-consistency-checker` is now guarded against stale or expired snapshots being used to suppress drift warnings, which is a behavioral tightening that may surface previously hidden drift events.
- The `optimizerWorkflowSource` helper in the new protocol test file uses `runtime.Caller` to locate the workflow markdown at test time, creating a filesystem dependency on the repository layout.

---

*ADR created by [adr-writer agent].*
