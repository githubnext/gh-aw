# Formal Notes: awf-config-sources-compliance/README.md

**Last formalized**: 2026-07-23-16-03-14
**Notation**: TLA+ / Z3 / F*
**Issue**: pending

## Predicates

| ID | Predicate | Description |
|---|---|---|
| P1 | `DriftRecordRequiredFields` | Every DriftRecord must have all four required fields |
| P2 | `DriftCategoryEnum` | drift_category must be one of the three allowed enum values |
| P3 | `DetectedAtISO8601` | detected_at must be a valid ISO 8601 UTC timestamp |
| P4 | `SuggestedActionNonEmpty` | suggested_action must not be empty |
| P5 | `NoAdditionalProperties` | DriftRecord must not contain extra properties |
| P6 | `CorrectivePRTrigger` | Actionable drift categories trigger corrective PR |
| P7 | `SLAEscalationTrigger` | Exceeded SLA window triggers escalation issue |
| P8 | `CorrectivePREmbedsList` | PR description must embed full DriftRecord list as JSON |
| P9 | `EmptyListIsValid` | Empty drift list is valid, no PR or escalation triggered |
| P10 | `DriftOutputIsJSONArray` | Step 5 output must be a JSON array conforming to schema |

## Key Invariants

- A DriftRecord without all four required fields is invalid and must be rejected
- drift_category is a closed enum; no other values are permitted
- An empty drift list suppresses all corrective actions
- Corrective PR body must contain the serialized DriftRecord array

## Edge Cases Identified

- DriftRecord with all required fields but extra unknown property
- detected_at with non-UTC timezone offset (e.g., +05:00)
- suggested_action containing only whitespace (should this be rejected?)
- Mixed-category lists (some actionable, some not) — should trigger PR
- Empty property_path string

## Notes for Future Runs

- The spec's compliance README closely mirrors the main awf-config-sources-spec.md §6.5 section
- Cross-spec dependency: CR-05 and CR-06 are defined in the parent spec
- The stub interface pattern is needed since DriftRecord validation lives in automation, not yet in pkg/workflow
