# Formal Notes: awf-config-sources-spec.md

**Last formalized**: 2026-08-09-15-35-55
**Notation**: TLA+ / Z3-style guard conjunction / F*
**Issue**: see notes-index.json

## Predicates

| ID | Predicate | Description |
|---|---|---|
| P1 | `formalDualSourceConsulted` | Both normative spec and published schema must be consulted (CR-01) |
| P2 | `formalNoUndocumentedFieldGeneration` | No config fields absent from both spec and schema may be generated (CR-03) |
| P3 | `formalDriftRecordStructuralValidity` | DriftRecord required fields non-empty (Section 3.1) |
| P4 | `formalDriftCategoryExhaustiveness` | drift_category enum is exhaustive |
| P5 | `formalSchemaOnlyPropertyFlaggedAsDrift` | Schema properties without gh-aw coverage flagged missing_in_ghaw (CR-02) |
| P6 | `formalCorrectionPRForActionableDrift` | missing_in_ghaw / spec_mismatch require corrective PR (CR-05) |
| P7 | `formalSLARemediationWindow` | 5-business-day SLA window computation (CR-06) |
| P8 | `formalEscalationIssueStructure` | Escalation issue must have Owner, UnblockPlan, RevisedETA (CR-06) |
| P9 | `formalSafeguardDegradedModeOnUnavailability` | Degraded mode triggers on canonical source unavailability (Section 8) |
| P10 | `formalDriftReportEmittedOnDetection` | Drift detection always emits a report (possibly empty) (Section 7.2 Step 5) |
| P11 | `formalSnapshotExpiry` | Snapshots >168h expired; MUST NOT suppress warnings; MUST mark degraded (Section 8) |
| P12 | `formalSnapshotStoragePath` | Stable snapshot path selection by runner persistence (self-hosted vs ephemeral) (Section 8) |
| P13 | `formalEscalationOwnerAssignment` | CR-06a fallback chain: last maintainer -> on-call maintainer |
| P14 | `formalEscalationOwnerNonEmpty` | Escalation issue MUST NOT be left unassigned (CR-06a(c)) |
| P15 | `formalEscalationAcknowledgementWindow` | Owner MUST acknowledge assignment within 1 business day (CR-06a(c)) |
| P16 | `formalCoverageVerificationEveryRun` | Per-run top-level schema-vs-CLI-mapping coverage check (CR-04) |

## Key Invariants

- Drift detection MUST consult both normative spec and JSON schema sources.
- Undocumented fields (absent from both) MUST NOT be generated.
- DriftRecord objects always have non-empty required fields.
- missing_in_ghaw / spec_mismatch always require corrective PR.
- SLA clock uses business days (Mon-Fri UTC).
- Snapshots >168h old MUST be treated as expired regardless of physical presence.
- Snapshots >336h (14 days) SHOULD be deleted.
- Escalation ownership must never be left unassigned; fallback last-maintainer -> on-call.
- Escalation owner MUST acknowledge within 1 business day of assignment.
- CR-04: every run SHOULD verify full top-level schema property coverage against CLI mapping table.

## Edge Cases Identified

- Snapshot age exactly at 168h boundary (not yet expired) vs. 168h+1min (expired).
- Snapshot degraded (>168h) but not yet eligible for deletion (<336h).
- Escalation assignment on a Friday requiring skip-to-Monday business-day math for 1-day ack window.
- `LastMaintainerKnown=true` but maintainer string empty — must still fall back to on-call (CR-06a).
- Partial CLI mapping coverage surfacing exactly the uncovered top-level property (CR-04).

## Notes for Future Runs

- P1-P10 already implemented in `pkg/workflow/awf_config_drift_formal_test.go` (verified present in repo).
- P11-P15 were referenced in the 2026-08-04 formal notes as "done" but had NO corresponding code in
  the repository — this run added them for real via a new test file
  `awf_config_safeguards_formal_test.go` with stub helpers marked `// stub — replace with real implementation`
  since no production drift-safeguard code exists yet in pkg/workflow/.
- Added new predicate P16 (`formalCoverageVerificationEveryRun`, CR-04) which had no prior formal coverage.
- Remaining gaps: Section 7.4.1 SLA tracking label/template requirements (issue title prefix, label pair)
  have no dedicated formal predicate yet; consider formalizing in a future run.
- IMPORTANT correction for future runs: verify claimed predicate implementations actually exist in
  `pkg/workflow/` before marking notes-index entries as done — the previous run's notes overstated
  completion of P11-P15.
