# Formal Notes: awf-config-sources-spec.md

**Last formalized**: 2026-08-04-16-10-28
**Notation**: TLA+ / Z3-style guard conjunction / F*
**Issue**: TBD (see notes-index.json)

## Predicates

| ID | Predicate | Description |
|---|---|---|
| P1 | `formalDualSourceConsulted` | Both normative spec and published schema must be consulted (CR-01) |
| P2 | `formalNoUndocumentedFieldGeneration` | No config fields absent from both spec and schema may be generated (CR-03) |
| P3 | `formalDriftRecordStructuralValidity` | DriftRecord required fields non-empty (Section 3.1) |
| P4 | `formalDriftCategoryExhaustiveness` | drift_category enum is exhaustive: missing_in_ghaw / missing_in_schema / spec_mismatch |
| P5 | `formalSchemaOnlyPropertyFlaggedAsDrift` | Schema properties without gh-aw coverage are flagged missing_in_ghaw (CR-02) |
| P6 | `formalCorrectionPRForActionableDrift` | missing_in_ghaw / spec_mismatch require corrective PR (CR-05) |
| P7 | `formalSLARemediationWindow` | 5-business-day SLA window computation (CR-06) |
| P8 | `formalEscalationIssueStructure` | Escalation issue must have Owner, UnblockPlan, RevisedETA (CR-06) |
| P9 | `formalSafeguardDegradedModeOnUnavailability` | Degraded mode triggers on canonical source unavailability (Section 8) |
| P10 | `formalDriftReportEmittedOnDetection` | Drift detection always emits a report (possibly empty) (Section 7.2 Step 5) |
| P11 | `formalSnapshotExpiry` | Snapshots >168h are expired, MUST NOT suppress warnings, MUST mark degraded (Section 8.1) |
| P12 | `formalSnapshotStoragePath` | Stable snapshot path selection by runner persistence (Section 8.1) |
| P13 | `formalEscalationOwnerAssignment` | CR-06a owner fallback chain: last maintainer -> on-call maintainer |
| P14 | `formalEscalationOwnerNonEmpty` | Escalation issue MUST NOT be left unassigned (CR-06a(c)) |
| P15 | `formalEscalationAcknowledgementWindow` | Owner MUST acknowledge assignment within 1 business day (CR-06a(c)) |

## Key Invariants

- Drift detection MUST consult both normative spec and JSON schema sources (never one alone).
- Undocumented fields (absent from both schema and spec) MUST NOT be generated.
- `DriftRecord` objects MUST always contain non-empty property_path, drift_category, suggested_action, detected_at.
- `missing_in_ghaw` and `spec_mismatch` categories always require a corrective PR; `missing_in_schema` does not.
- SLA clock uses business days (Mon-Fri UTC), skipping weekends.
- Snapshot-based safeguards must never silently pass checks when canonical sources are stale/expired (>7 days).
- Escalation ownership must never be left unassigned; fallback is last-maintainer -> on-call.

## Edge Cases Identified

- Boundary snapshot age exactly at 168 hours (not yet expired) vs. 168.01h (expired).
- Snapshot old enough to be flagged degraded (>7 days) but not yet eligible for deletion (<14 days).
- SLA/acknowledgement deadlines that fall on a Friday, requiring skip-to-Monday business-day math.
- Escalation owner input where `LastMaintainerKnown` is true but the maintainer string is empty (must still fall back to on-call).
- Empty `SuggestedAction` on an otherwise well-formed DriftRecord (structural invalidity).

## Notes for Future Runs

- Prior run (2026-07-26) covered `specs/awf-config-sources-compliance/README.md` (P1-P10, compliance variant).
- This run covers the sibling spec `specs/awf-config-sources-spec.md`, extending its existing
  `pkg/workflow/awf_config_drift_formal_test.go` (P1-P10) with 5 new predicates (P11-P15) derived
  from Section 8 (Safeguards) and CR-06a (Escalation Owner Assignment), which were previously
  unformalized.
- Future runs could formalize CR-04 (per-run top-level property coverage verification) and the
  7.4.1 SLA tracking label/template requirements as additional predicates if deeper coverage is
  desired.
