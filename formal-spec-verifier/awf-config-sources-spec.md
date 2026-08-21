# Formal Notes: awf-config-sources-spec.md

**Last formalized**: 2026-08-21-15-38-08
**Notation**: TLA+ / Z3-style guard conjunction
**Issue**: created via safe-output (number resolved post-run)

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
| P17 | `formalEscalationLabelPairComplete` | Escalation issue MUST carry both `workflow` + `bug` labels (Section 7.4.1) |
| P18 | `formalEscalationTitlePrefix` | Escalation issue title MUST begin with `[Schema Drift SLA]` (Section 7.4.1) |
| P19 | `formalEscalationTemplateFieldsComplete` | Minimum required escalation template fields non-empty; waiver rationale optional (Section 7.4.1) |

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
- Section 7.4.1: escalation issues MUST carry both `workflow` and `bug` labels, MUST have title prefixed `[Schema Drift SLA]`, and MUST populate all template fields except the optional waiver rationale.

## Edge Cases Identified

- Snapshot age exactly at 168h boundary (not yet expired) vs. 168h+1min (expired).
- Snapshot degraded (>168h) but not yet eligible for deletion (<336h).
- Escalation assignment on a Friday requiring skip-to-Monday business-day math for 1-day ack window.
- `LastMaintainerKnown=true` but maintainer string empty — must still fall back to on-call (CR-06a).
- Partial CLI mapping coverage surfacing exactly the uncovered top-level property (CR-04).
- Escalation title exactly equal to the required `[Schema Drift SLA]` prefix satisfies the predicate; a truncated prefix (shorter substring) does not (P18, formalized this run).
- Escalation label set containing only one of `workflow`/`bug` is rejected by the overall validity gate (P17, formalized this run).
- Escalation template with empty waiver rationale but all other fields populated remains valid; the completeness result is identical whether or not waiver rationale is set (P19, formalized this run).

## Notes for Future Runs

- P1-P10 implemented in `pkg/workflow/awf_config_drift_formal_test.go` (verified present in repo).
- P11-P16 implemented in `pkg/workflow/awf_config_safeguards_formal_test.go` and related files (stub-based, no production code yet).
- **This run (2026-08-21) closed the P17-P19 gap** that was explicitly called out as unfinished in the 2026-08-16 notes: added `pkg/workflow/awf_config_escalation_template_formal_test.go` with 8 test functions covering the escalation label pair, title prefix, template field completeness, waiver-rationale optionality, and two dedicated edge cases (partial label set rejection, exact-boundary title prefix).
- All P1-P19 predicates now have dedicated formal test files backing stub implementations; none have real production code in `pkg/workflow/` yet — a future pass should audit whether escalation-issue construction logic has been added to `pkg/workflow/` (e.g., a real `EscalationIssue` type or drift-SLA-tracking workflow implementation) and, if so, replace the stubs with calls into that real code.
- Cross-spec note: `pkg/workflow/awf_config_conformance_registry_formal_test.go` already exists and covers T-DR test-ID registry mechanics (P1-P9 registry-specific, distinct series from the drift/safeguard predicates above) — do not duplicate.
- Next candidate specs in rotation (not yet touched or oldest-processed): `specs/compiler-threat-detection-compliance/README.md`, `specs/forecast-compliance-fixtures/README.md`, `specs/github-mcp-access-control-compliance/README.md`.
