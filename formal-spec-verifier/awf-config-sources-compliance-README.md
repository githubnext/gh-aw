# Formal Notes: awf-config-sources-compliance/README.md

**Last formalized**: 2026-08-20-15-39-59
**Notation**: TLA+ / Z3-style guard conjunction
**Issue**: (created via safe-output; number resolved post-run)

## Predicates

| ID | Predicate | Description |
|---|---|---|
| G1 | `RequirementKeywordPresent` | every row's text contains MUST or SHOULD |
| G2 | `MustRowIsHardFail` | strongest keyword MUST -> enforcement = reject |
| G3 | `ShouldRowIsSoftDegrade` | strongest keyword SHOULD-only -> enforcement = warn_or_degrade |
| G4 | `EveryRowHasSectionAnchor` | every row cites a well-formed "§" section reference |
| G5 | `TableIDPrefixMatchesTable` | drift rows match T-DR-NNN, safeguard rows match T-DR-SAFE-NNN, never cross-validate |
| G6 | `EmptyDriftListNeverEscalates` | empty DriftRecord list never triggers corrective PR or escalation |
| G7 | `EscalationRequiresBothConditions` | escalation requires SLA exceeded AND actionable records present |
| G8 | `BothTablesNonEmptyAndDisjointIDs` | both conformance tables non-empty, ID sets fully disjoint |

## Key Invariants

- MUST takes precedence over SHOULD when a single row description contains both (e.g. T-DR-SAFE-001).
- The DriftRecord table (T-DR-001..010) and Safeguards table (T-DR-SAFE-001..004) use disjoint ID prefixes and must never collide.
- Escalation is a conjunction, not a disjunction: SLA breach alone or actionable records alone are each insufficient.
- Section anchors must literally start with the "§" sigil followed by a numeric path; text like "section 7.5.1" or a bare "§" both fail.

## Edge Cases Identified

- A row with neither MUST nor SHOULD (hypothetical, not present in the current README) must resolve to `invalid` enforcement, not silently default to reject or warn.
- T-DR-SAFE-001's mixed MUST+SHOULD description must resolve its overall enforcement to MUST (reject), not the weaker SHOULD.
- Malformed section anchors (missing "§", bare "§" with no number, or wrong prefix word) must all fail the anchor predicate.

## Notes for Future Runs

- This is the third formal pass over this same README file. Run 1 (2026-07-26) covered the DriftRecord *schema* itself (property/type-level validation, delegated to `pkg/workflow/awf_config_drift_test.go`). Run 2 (2026-08-19) covered the conformance-*registry meta-process* (ID assignment, uniqueness, routing rules). This run (2026-08-20) covers *requirement strength classification* (MUST vs SHOULD enforcement), escalation gating logic, and cross-table ID disjointness — a third, non-overlapping dimension.
- New test file: `pkg/workflow/awf_config_fixture_index_formal_test.go` (package `workflow`, in-file fixtures mirroring both README tables; no YAML loading, no stubs needed since it operates on literal table data).
- All three specs rotation-processed on 2026-08-20 had already been visited within the trailing 14 days, so `processed` was reset and `last_index` restarted at 0 per the rotation reset rule. Future runs should re-check `rotation.json` for the next unprocessed spec.
- Potential future direction: if the README tables are ever externalized into structured YAML/JSON (as hinted in run 2's notes), this file's `formalDriftTableRows`/`formalSafeguardTableRows` fixtures should be replaced with a loader reading the real file, keeping the same predicate functions.
