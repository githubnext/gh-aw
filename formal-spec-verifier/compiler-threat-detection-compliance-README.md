# Formal Notes: compiler-threat-detection-compliance/README.md

**Last formalized**: 2026-08-23-15-33-56
**Notation**: Z3-style guard conjunction / TLA+-style state machine
**Issue**: (created this run via create_issue safe output)

## Predicates

| ID | Predicate | Description |
|---|---|---|
| P1 | `RuleTestIDBijection` | Every active CTR-* rule maps to exactly one T-CTR-* test ID and vice versa (2026-08-15) |
| P2 | `TestIDFormatWellFormed` | Rule IDs match `^CTR-\d{3}$`; test IDs match `^T-CTR-\d{3}$` (2026-08-15) |
| P3 | `NoOrphanTestID` | No T-CTR-* test ID exists without a corresponding CTR-* rule (2026-08-15) |
| P4 | `DeterministicDiagnostic` | Same malicious input always yields the same diagnostic ID (2026-08-15) |
| P5 | `StableDiagnosticIDPresence` | Compiler error output contains the rule's stable diagnostic ID (2026-08-15) |
| P6 | `DeprecatedRuleExclusion` | Deprecated rules marked [DEPRECATED] and excluded from required gate (2026-08-15) |
| P7 | `NewRuleRequiresTestID` | New rule in §5.1 requires new test ID in §8.1 in same change set (2026-08-15) |
| P8 | `ActiveRuleCoverageComplete` | Count of active rules equals count of mapped test IDs (2026-08-15) |
| P9 | `SuppressionRequiresRuleAndReason` | Suppression rejected without non-empty `reason` (T-CTR-024) (2026-08-23) |
| P10 | `SuppressionRuleFormatWellFormed` | Suppression `rule` must match `CTR-\d{3}` (T-CTR-024) (2026-08-23) |
| P11 | `SuppressionExpiresISO8601OrAbsent` | Optional `expires` must be ISO 8601 or absent (T-CTR-024) (2026-08-23) |
| P12 | `ActiveSuppressionRetainsAuditFields` | Parsed suppression retains exact `rule`/`reason`/`expires` (T-CTR-025) (2026-08-23) |
| P13 | `ExpiredSuppressionTreatedAsAbsent` | Suppression past its `expires` date is treated as non-existent (T-CTR-029) (2026-08-23) |
| P14 | `SuppressionBoundaryDayStillActive` | Suppression active through its own expires calendar day (T-CTR-029 edge case) (2026-08-23) |
| P15 | `DiagnosticSuppressionRequiresMatchingRule` | Suppression only applies to its own rule ID, no cross-rule leakage (2026-08-23) |
| P16 | `DeprecatedRuleRetainsCatalogRow` (stub) | §5.4: deprecated rule row retained, not deleted (2026-08-23) |
| P17 | `DeprecatedRuleExcludedFromRequiredGate` (stub) | §5.4: deprecated rule excluded from required conformance gate (2026-08-23) |

## Key Invariants

- 1:1 bijection between the 23 active CTR-* rules and T-CTR-* test IDs (currently CTR-001..CTR-023, plus CTR-025/CTR-026 mapped to T-CTR-039/T-CTR-041).
- All rule/test IDs are strictly formatted (`CTR-\d{3}`, `T-CTR-\d{3}`).
- Determinism: identical malicious input must always produce identical diagnostic across repeated compiler runs.
- Stable diagnostic IDs (e.g. `CTR-006`) must appear verbatim in compiler error text for CI mechanical verification.
- Deprecated rules are excluded from the required compliance gate but retain historical test ID with `[DEPRECATED]` marker; the catalog row itself is never deleted (§5.4).
- Suppression validation requires a well-formed `CTR-\d{3}` rule ID and a non-empty `reason`; `expires` is optional but must be ISO 8601 when present.
- Suppressions are audited with full fidelity (`rule`, `reason`, `expires`) while active, and automatically become inert (treated as absent) once their `expires` date has passed — but remain active through the expires day itself (inclusive boundary).
- Suppression scope is strictly per-rule: a suppression for one `CTR-*` rule must never mask diagnostics for a different rule.

## Edge Cases Identified

- Empty/no malicious input presented to a detector — must error explicitly rather than silently pass or false-positive.
- Unknown/nonexistent rule ID lookup — must report not-found, never silently match another rule.
- Duplicate test ID accidentally assigned to two different rules — violates the bijection invariant and must be detectable.
- New rule added to §5.1 without a corresponding new test ID in §8.1 in the same change set — violates P7.
- Suppression with empty/whitespace-only `reason` — must be rejected (P9).
- Suppression with malformed `rule` ID (missing digits, wrong prefix, empty) — must be rejected (P10).
- Suppression `expires` in a non-ISO-8601 format or an invalid calendar date (e.g. month 13) — must be rejected (P11).
- Suppression exactly on its `expires` boundary day — still active; the following day it is expired (P14).
- Suppression for rule A must not accidentally suppress a diagnostic for rule B (P15).

## Notes for Future Runs

- Test file target (this run): `pkg/workflow/threat_detection_suppression_lifecycle_formal_test.go` (internal `package workflow` — required for access to unexported `parseThreatDetectionSuppressions` / `isThreatDetectionRuleSuppressed` helpers in `pkg/workflow/threat_detection_suppression.go`).
- Prior run's test file target: `pkg/workflow/compiler_threat_detection_compliance_formal_test.go` (catalog bijection predicates only — already implemented in the codebase as of this run, confirmed via `TestFormal_RuleTestIDBijection` etc.).
- §5.4 Deprecation Policy (P16/P17) currently has **no concrete Go implementation** in `pkg/workflow` — it is a documentation/process obligation on the spec text and the Section 7.1 mapping table, not a runtime API. Tests use a stub `deprecationRegistry`. A future run should check whether a deprecation-registry helper has been added and replace the stub with real calls.
- Section 6.6 Optimizer Failure Safeguards (`OPTIMIZER_DEGRADED`, `OPTIMIZER_TIMEOUT`, `OPTIMIZER_RATE_LIMITED`, `OPTIMIZER_MISSED_CRON`) are implemented and tested in `pkg/workflow/compiler_threat_optimizer_protocol_test.go` (T-CTR-030 through T-CTR-040) — not re-formalized this run since they already have dedicated coverage; a future pass could still add formal predicate framing for completeness.
- CTR-025/CTR-026 (mapped to T-CTR-039/T-CTR-041) were not individually formalized this run — their specific detection triggers live in the parent spec §8.1 and would be a good target for a future per-rule formalization pass, as previously noted.
- Per-rule semantic formalization (specific detection trigger/action pairs for CTR-001..CTR-023 individually) remains an open follow-up noted since 2026-08-15; this run instead deepened the process/lifecycle layer (suppression + deprecation) rather than the per-rule semantic layer.
