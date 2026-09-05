# Formal Notes: awf-config-sources-compliance/README.md

**Last formalized**: 2026-09-05-15-38-00
**Notation**: TLA+ / Z3-style guard conjunction
**Issue**: created via safe-output (number resolved post-run)

## Predicates

| ID | Predicate | Description |
|---|---|---|
| G1 | `RequirementKeywordPresent` | every row's text contains MUST or SHOULD (formalized 2026-08-20) |
| G2 | `MustRowIsHardFail` | strongest keyword MUST -> enforcement = reject (formalized 2026-08-20) |
| G3 | `ShouldRowIsSoftDegrade` | strongest keyword SHOULD-only -> enforcement = warn_or_degrade (formalized 2026-08-20) |
| G4 | `EveryRowHasSectionAnchor` | every row cites a well-formed "§" section reference (formalized 2026-08-20) |
| G5 | `TableIDPrefixMatchesTable` | drift rows match T-DR-NNN, safeguard rows match T-DR-SAFE-NNN, never cross-validate (formalized 2026-08-20) |
| G6 | `EmptyDriftListNeverEscalates` | empty DriftRecord list never triggers corrective PR or escalation (formalized 2026-08-20) |
| G7 | `EscalationRequiresBothConditions` | escalation requires SLA exceeded AND actionable records present (formalized 2026-08-20) |
| G8 | `BothTablesNonEmptyAndDisjointIDs` | both conformance tables non-empty, ID sets fully disjoint (formalized 2026-08-20) |
| T1 | `formalTestIDWellFormed` | every Test ID matches T-DR-NNN or T-DR-SAFE-NNN (this run) |
| T2 | `formalTestIDsUnique` | no duplicate Test ID within a table (this run) |
| T3 | `formalImplementationFileDeclared` | every row names a non-empty implementation file under pkg/workflow/ (this run) |
| T4 | `formalGoFunctionNameEncodesTestID` | Go test function name embeds its Test ID token (this run) |
| T5 | `formalNextIDAlgorithmMonotonic` | documented next-ID lookup script yields strictly greater than max existing suffix (this run) |
| T6 | `formalSeriesNamespacesIndependent` | T-DR-SAFE-### numbering does not interfere with T-DR-### numbering (this run) |
| T7 | `formalRunCommandReferencesRealFuncPrefixes` | documented `-run` pattern matches actual Go test function prefixes (this run) |
| T8 | `formalDriftAndSafeguardTablesDisjoint` | DriftRecord table IDs and Safeguard table IDs share no common ID (this run; re-verifies G5/G8 at the traceability layer) |

## Key Invariants

- MUST takes precedence over SHOULD when a single row description contains both (e.g. T-DR-SAFE-001).
- The DriftRecord table (T-DR-001..011) and Safeguards table (T-DR-SAFE-001..004) use disjoint ID prefixes and must never collide.
- Escalation is a conjunction, not a disjunction: SLA breach alone or actionable records alone are each insufficient.
- Section anchors must literally start with the "§" sigil followed by a numeric path.
- Every Test ID must map 1:1 to a declared implementation file under `pkg/workflow/` and a Go test function whose name embeds the hyphen-stripped ID token.
- The documented next-ID lookup script (`grep | sed | sort -n | tail -1 | awk`) must be strictly monotonic per namespace and must never let the drift series and safeguard series interfere with each other.
- The README's `go test -run "TestDriftRecord|TestAWFConfigSafeguard"` command must actually match every Go test function's prefix, or the instructions silently skip tests.

## Edge Cases Identified

- A row with neither MUST nor SHOULD must resolve to `invalid` enforcement (from 2026-08-20 pass).
- T-DR-SAFE-001's mixed MUST+SHOULD description resolves to MUST (reject).
- Malformed section anchors must fail the anchor predicate.
- (This run) A Test ID with a non-numeric suffix (e.g. `T-DR-0AB`) must be rejected as malformed even though it starts with the correct prefix.
- (This run) An implementation file path escaping `pkg/workflow/` (e.g. relative traversal) must fail the declared-file predicate.
- (This run) The next-ID algorithm over an *empty* table must start numbering at `001`, not `000`, and must not panic.
- (This run) The next-ID algorithm must correctly ignore rows from the *other* namespace (e.g. skip `T-DR-SAFE-*` rows when computing the next `T-DR-*` ID) even when both appear in the same input slice — proving namespace isolation is enforced by the algorithm itself, not just by table separation.

## Notes for Future Runs

- This is the fourth formal pass over this README file. Run 1 (2026-07-26): DriftRecord schema-level validation. Run 2 (2026-08-19): conformance-registry meta-process (ID assignment, uniqueness, routing). Run 3 (2026-08-20): requirement-strength classification (MUST/SHOULD), escalation gating, cross-table disjointness (G1-G8). **Run 4 (this run, 2026-09-05)**: traceability integrity dimension — Test ID ↔ implementation-file ↔ Go-function-name binding, and the documented next-ID assignment algorithm (T1-T8).
- New test file: `pkg/workflow/awf_config_traceability_formal_test.go` (package `workflow_test`, black-box; literal fixture rows mirroring both README tables, no YAML/markdown parsing needed).
- T-DR-011 (added between run 3 and this run) is already reflected correctly in this run's fixtures and next-ID computation (next drift ID computed as `T-DR-012`).
- Potential future direction (carried from run 3, still open): if the README tables are ever externalized into structured YAML/JSON, both `awf_config_fixture_index_formal_test.go` and this run's `awf_config_traceability_formal_test.go` fixtures should be replaced with a shared loader reading the real file.
- Untouched dimensions for a possible future (5th) pass: the "Adding New Safeguard Conformance Tests" section's routing rule (drift-vs-safeguard file routing based on whether a safeguard "spans drift output and schema validation"), and validation that every safeguard bullet in `specs/awf-config-sources-spec.md` §8 actually cross-references its corresponding `T-DR-SAFE-###` ID (bidirectional spec↔README traceability, as opposed to this run's README↔Go traceability).
- Next candidate specs in rotation (oldest-touched / not yet processed): `specs/github-mcp-access-control-compliance/README.md`, `specs/awf-config-sources-spec.md` (the underlying spec itself is due for re-formalization given several README-only passes since P1-P19 were last touched 2026-08-21).
