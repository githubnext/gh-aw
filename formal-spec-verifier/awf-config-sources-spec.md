# Formal Notes: awf-config-sources-spec.md

**Last formalized**: 2026-09-06-15-28-00
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
| P20 | `formalAutomationTriggerCondition` | §7.1 trigger conditions: schema/spec-touching PR, scheduled cron, or explicit agent request |
| P21 | `formalScheduledRunFailsOnMissingInGhaw` | §7.4 bullet 2: non-zero exit required when missing_in_ghaw drift is found |
| P22 | `formalPRSummaryCommentRequired` | §7.4 bullet 3: PR-triggered runs SHOULD post a summary comment with the drift report |
| P23 | `formalScheduledTrackingIssueOnDrift` | §7.4 bullet 4: scheduled run MUST create a tracking issue when drift is detected |
| P24 | `formalAutomationExitCodeMonotone` | Derived: any missing_in_ghaw record forces failing exit code regardless of other benign-category records present |

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
- §7.1: three valid trigger conditions — schema/spec-touching PR, scheduled cron, explicit agent request.
- §7.4 bullet 2: the check MUST fail (non-zero exit) when any missing_in_ghaw drift is found; presence of a single such record forces failure even amid otherwise-benign categories.
- §7.4 bullet 3: PR-triggered runs SHOULD post a summary comment with the drift report, even when the drift list is empty.
- §7.4 bullet 4: scheduled runs MUST create a tracking issue when drift is detected; this obligation does not apply to PR or ad hoc triggers.

## Edge Cases Identified

- Snapshot age exactly at 168h boundary (not yet expired) vs. 168h+1min (expired).
- Snapshot degraded (>168h) but not yet eligible for deletion (<336h).
- Escalation assignment on a Friday requiring skip-to-Monday business-day math for 1-day ack window.
- `LastMaintainerKnown=true` but maintainer string empty — must still fall back to on-call (CR-06a).
- Partial CLI mapping coverage surfacing exactly the uncovered top-level property (CR-04).
- Escalation title exactly equal to the required `[Schema Drift SLA]` prefix satisfies the predicate; a truncated prefix (shorter substring) does not (P18, formalized this run).
- Escalation label set containing only one of `workflow`/`bug` is rejected by the overall validity gate (P17, formalized this run).
- Escalation template with empty waiver rationale but all other fields populated remains valid; the completeness result is identical whether or not waiver rationale is set (P19, formalized this run).
- PR-triggered run with zero drift records still must post a summary comment; omitting it is non-conformant (P22 edge case, this run).
- PR run with only `spec_mismatch`-category drift is not tied to a failing exit code by §7.4 bullet 2 text scope, unlike `missing_in_ghaw` (P21/P24 edge case, this run).
- Ad hoc/manual invocation (§7.1 item 3) is a valid trigger but is exempt from both the PR-summary-comment and scheduled-tracking-issue obligations even when drift is present (P22/P23 edge case, this run).

## Notes for Future Runs

- P1-P10 implemented in `pkg/workflow/awf_config_drift_formal_test.go` (verified present in repo).
- P11-P16 implemented in `pkg/workflow/awf_config_safeguards_formal_test.go` and related files (stub-based, no production code yet).
- The 2026-08-21 run closed the P17-P19 gap (escalation label pair, title prefix, template field completeness) in `pkg/workflow/awf_config_escalation_template_formal_test.go`.
- **This run (2026-09-06) closed the §7.1/§7.4 automation-trigger gap**: added `pkg/workflow/awf_config_automation_trigger_formal_test.go` with predicates P20-P24 and 8 test functions covering trigger-condition validity, scheduled-run fail-on-drift exit-code semantics, PR summary-comment obligations, scheduled tracking-issue creation, exit-code monotonicity across mixed drift categories, and an end-to-end conformance matrix including the ad hoc-exemption edge case.
- All P1-P24 predicates now have dedicated formal test files backing stub implementations; none have real production code in `pkg/workflow/` yet — a future pass should audit whether escalation-issue construction logic or a real schema-consistency-checker automation type has been added to `pkg/workflow/` and, if so, replace the stubs with calls into that real code.
- Cross-spec note: `pkg/workflow/awf_config_conformance_registry_formal_test.go` already exists and covers T-DR test-ID registry mechanics (P1-P9 registry-specific, distinct series from the drift/safeguard predicates above) — do not duplicate.
- Remaining unformalized area identified this run: §7.3 "Example Drift Check (CLI)" bash snippet is illustrative tooling only (not a MUST/SHOULD normative obligation) — low priority for formalization. §5 "Known drift example (apiProxy)" is a worked example already covered indirectly by P5/P6 test fixtures — no new predicate needed.
- Next candidate specs in rotation (not yet touched or oldest-processed): `specs/compiler-threat-detection-compliance/README.md`, `specs/forecast-compliance-fixtures/README.md`, `specs/github-mcp-access-control-compliance/README.md`.
