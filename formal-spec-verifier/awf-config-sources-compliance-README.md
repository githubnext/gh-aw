# Formal Notes: awf-config-sources-compliance/README.md

**Last formalized**: 2026-08-19-15-44-16
**Notation**: TLA+ / Z3-style guard conjunction
**Issue**: (created via safe-output; number resolved post-run)

## Predicates

| ID | Predicate | Description |
|---|---|---|
| P1 | `TestIDMonotonicity` | next T-DR-xxx ID strictly greater than max existing ID |
| P2 | `TestIDNoDuplicates` | no two conformance rows share a T-DR-xxx ID |
| P3 | `TestIDFormatWellFormed` | every assigned ID matches ^T-DR-(SAFE-)?\d{3,}$ |
| P4 | `PlaceholderIDRejectedAsFinal` | "T-DR-NNN" draft placeholder MUST NOT be treated as assigned/final |
| P5 | `RowHasRequirementReference` | every conformance row cites a "§" section reference |
| P6 | `RowHasImplementationFile` | every conformance row maps to a concrete Go test file path under pkg/workflow/ |
| P7 | `SafeguardRowRoutingDecision` | safeguard rows spanning schema+drift route to drift test file; pure safeguard rows route to safeguards test file |
| P8 | `SpecCrossReferenceRequired` | new row MUST be cross-referenced from specs/awf-config-sources-spec.md |
| P9 | `DriftSeriesVsSafeguardSeriesDisjoint` | T-DR-001..010 and T-DR-SAFE-001..004 numbering series never collide |
| P10 | `EmptyRegistryNextIDIsFirst` | empty registry's next-ID computation starts at T-DR-001 |

(Prior run's P1-P10, covering the DriftRecord *schema* itself, remain fully implemented in
`pkg/workflow/awf_config_drift_test.go`; this run's P1-P10 are a distinct set covering the
*conformance-registry meta-process* — ID assignment, uniqueness, routing, cross-referencing.)

## Key Invariants

- ID series are disjoint: T-DR-xxx (DriftRecord, 001-010) vs T-DR-SAFE-xxx (safeguards, 001-004).
- Next-ID assignment is strictly monotonic per series and ignores the other series.
- The README's own documented placeholder "T-DR-NNN" must never validate as a final ID.
- Every row requires both a "§"-referenced requirement AND a concrete pkg/workflow/ test file.
- Routing rule for safeguard behavior: pure safeguard -> safeguards_formal_test.go; schema+drift span -> drift_test.go.

## Edge Cases Identified

- Four-digit ID rollover beyond 999 (T-DR-1000) must remain well-formed, no truncation.
- A registry containing only the safeguard series must not perturb the plain-series next-ID computation.
- A row with an empty TestFile fails the implementation-file requirement (incomplete registry entry).

## Notes for Future Runs

- This formalization is complementary to the 2026-07-26 run (DriftRecord schema-level P1-P10);
  it targets the *meta-process* of adding new conformance rows/IDs described in the README's
  "Adding New Conformance Tests" and "Adding New Safeguard Conformance Tests" sections instead.
- The registry today is a markdown table, not structured data — the conformanceRow type and
  nextConformanceID/routeSafeguardRow functions in the new test file are stubs; if the registry
  is ever materialized as JSON/YAML, wire these functions to the real data source.
- Cross-spec dependency: parent spec `specs/awf-config-sources-spec.md` (formalized 2026-08-16)
  remains the canonical DriftRecord schema source; this README is the fixture index layer.
- Next candidate for rotation: any spec not processed in the last 14 days — check rotation.json.
