# Formal Notes: awf-config-sources-compliance/README.md

**Last formalized**: 2026-07-26-15-46-07
**Notation**: TLA+ / Z3 / F*
**Issue**: (pending)

## Predicates

| ID | Predicate | Description |
|---|---|---|
| P1 | `RequiredFieldsComplete` | All four required fields (property_path, drift_category, suggested_action, detected_at) must be non-zero |
| P2 | `DriftCategoryEnum` | drift_category ∈ {missing_in_ghaw, missing_in_schema, spec_mismatch}; no other value admitted |
| P3 | `DetectedAtISO8601` | detected_at must parse as a valid RFC 3339 / ISO 8601 UTC timestamp |
| P4 | `SuggestedActionNonEmpty` | suggested_action string length must be ≥ 1 |
| P5 | `NoAdditionalProperties` | DriftRecord keys are exactly the four required fields; extra properties rejected |
| P6 | `CorrectivePRTrigger` | missing_in_ghaw or spec_mismatch category → corrective PR must be opened (CR-05) |
| P7 | `SLAEscalationTrigger` | SLA exceeded AND actionable drift present → escalation issue opened/updated (CR-06) |
| P8 | `CorrectivePREmbedsFullList` | Corrective PR body must embed full DriftRecord list as JSON |
| P9 | `EmptyListNoAction` | Empty drift list must NOT trigger PR or escalation |
| P10 | `DriftProcedureOutputIsJSONArray` | Drift detection output must be a valid JSON array of DriftRecord objects |

## Key Invariants

- Schema closed: no properties beyond the four required fields are allowed
- Enum strict: drift_category restricted to three values; case-sensitive
- Timestamp format: ISO 8601 UTC; YYYY-MM-DDTHH:MM:SSZ canonical form
- PR trigger: missing_in_ghaw OR spec_mismatch (not missing_in_schema) triggers CR-05
- SLA escalation: requires both SLA breach AND at least one actionable record
- Empty list is valid: an empty list must silently pass with no side effects

## Edge Cases Identified

- missing_in_schema category: does NOT trigger corrective PR (only missing_in_ghaw and spec_mismatch do)
- Empty drift list with SLA exceeded: must still not trigger escalation (no actionable records)
- Mixed list (actionable + non-actionable): corrective PR opened; full list embedded
- Whitespace-only suggested_action: syntactically non-empty; spec may clarify
- detected_at with timezone offset (not Z): spec says UTC; implementation should validate or normalize

## Notes for Future Runs

- Test file target: pkg/workflow/awf_config_drift_formal_test.go
- Implementation is aspirational (DriftRecord validation not yet found in pkg/workflow/)
- Cross-spec dependency: parent spec awf-config-sources-spec.md §6.5 is the canonical source
- CR-06a (escalation owner assignment) is a rich sub-predicate worth formalizing separately
- SLA window computation (business days Mon-Fri UTC) is a non-trivial temporal property for TLA+
