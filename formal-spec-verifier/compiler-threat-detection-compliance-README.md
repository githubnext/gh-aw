# Formal Notes: compiler-threat-detection-compliance/README.md

**Last formalized**: 2026-08-15-15-35-13
**Notation**: Z3-style guard conjunction / set-theoretic bijection
**Issue**: (pending — created via create_issue safe output this run)

## Predicates

| ID | Predicate | Description |
|---|---|---|
| P1 | `RuleTestIDBijection` | Every active CTR-* rule maps to exactly one T-CTR-* test ID and vice versa |
| P2 | `TestIDFormatWellFormed` | Rule IDs match `^CTR-\d{3}$`; test IDs match `^T-CTR-\d{3}$` |
| P3 | `NoOrphanTestID` | No T-CTR-* test ID exists without a corresponding CTR-* rule |
| P4 | `DeterministicDiagnostic` | Same malicious input always yields the same diagnostic ID |
| P5 | `StableDiagnosticIDPresence` | Compiler error output contains the rule's stable diagnostic ID |
| P6 | `DeprecatedRuleExclusion` | Deprecated rules marked [DEPRECATED] and excluded from required gate |
| P7 | `NewRuleRequiresTestID` | New rule in §5.1 requires new test ID in §8.1 in same change set |
| P8 | `ActiveRuleCoverageComplete` | Count of active rules equals count of mapped test IDs |

## Key Invariants

- 1:1 bijection between the 23 active CTR-* rules and T-CTR-* test IDs (currently CTR-001..CTR-023)
- All rule/test IDs are strictly formatted (`CTR-\d{3}`, `T-CTR-\d{3}`)
- Determinism: identical malicious input must always produce identical diagnostic across repeated compiler runs
- Stable diagnostic IDs (e.g. `CTR-006`) must appear verbatim in compiler error text for CI mechanical verification
- Deprecated rules are excluded from the required compliance gate but retain historical test ID with `[DEPRECATED]` marker

## Edge Cases Identified

- Empty/no malicious input presented to a detector — must error explicitly rather than silently pass or false-positive
- Unknown/nonexistent rule ID lookup — must report not-found, never silently match another rule
- Duplicate test ID accidentally assigned to two different rules — violates the bijection invariant and must be detectable
- New rule added to §5.1 without a corresponding new test ID in §8.1 in the same change set — violates P7

## Notes for Future Runs

- Test file target: `pkg/workflow/compiler_threat_detection_compliance_formal_test.go`
- This README is a thin cross-reference table; the richer normative detail (detection triggers, expected actions,
  diagnostic IDs) lives in the parent spec `specs/compiler-threat-detection-spec.md` §8.1. Future runs on that
  parent spec should formalize each CTR-xxx rule's specific detection trigger/action pair individually — this run
  only formalized the catalog-level bijection/coverage properties, not per-rule semantics.
- Only one concrete implementation hook was found in the codebase during this run: CTR-015 in
  `pkg/workflow/safe_outputs_allowed_labels_validation.go`. Other rules (CTR-001..CTR-023) likely have validators
  scattered across `pkg/workflow/` and `pkg/parser/` (e.g. CTR-022 in `pkg/gitutil/gitutil.go` per spec changelog) —
  worth a follow-up run that maps each T-CTR-* ID to its concrete `_test.go` file for a stronger coverage predicate.
- Deprecation semantics (§5.3.1) were not read this run; a future pass could formalize the deprecation lifecycle
  transition (`active -> deprecated -> removed`) as a small state machine.
