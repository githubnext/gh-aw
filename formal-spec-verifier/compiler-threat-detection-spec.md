# Formal Notes: compiler-threat-detection-spec.md

**Last formalized**: 2026-07-28-16-10-40
**Notation**: Z3 / SMT-LIB, F*, TLA+
**Issue**: (created via safe-output; number assigned post-processing)

## Predicates

| ID | Predicate | Description |
|---|---|---|
| P1 | `RuleModelComplete` | Every catalog rule has all six required fields (ID, threat class, detection condition, action, evidence, impl mapping) |
| P2 | `DeterministicResponse` | Same (rule, input) pair always yields the same diagnostic/action |
| P3 | `SecureOutcome` | Triggered rule either rejects compilation or applies a safe rewrite |
| P4 | `NoHyphenPrefixPassthrough` | CTR-013: package/image names starting with `-` are rejected before reaching `exec.Command` |
| P5 | `WorkflowsFieldMandatory` | CTR-021: missing/empty `workflow_run.workflows` is a hard error in both strict and non-strict modes |
| P6 | `BranchScopeModeSensitive` | CTR-021: missing `branches:` warns in non-strict mode, rejects in strict mode |
| P7 | `DeprecationRetainsEntry` | Deprecated rules keep their catalog row (never deleted) |
| P8 | `DeprecatedTestsNotRequired` | Tests mapped to a deprecated rule are excluded from required-for-conformance |
| P9 | `VersionSyncInvariant` | Every published spec version has a corresponding compatibility table row (§2) |
| P10 | `Conformance` | Conjunction of P1..P9; a single violated sub-predicate breaks overall conformance |

## Key Invariants

- All 21 rules (CTR-001..CTR-021) must maintain non-empty implementation + test mapping in §7.1 (audited as of commit d4872c2, 2026-07-26 — no gaps found).
- Spec-to-implementation sync table (§2) must be updated in the same PR as any lock-file compatibility change.
- Deprecation is append-only: rule rows are annotated `[Deprecated in vX.Y.Z]`, never removed (§5.3.1).
- Mode sensitivity (strict vs non-strict) is a recurring pattern across many rules (CTR-004, CTR-009, CTR-011, CTR-014, CTR-015, CTR-017, CTR-018, CTR-021) — worth a cross-cutting formalization in a future run.

## Edge Cases Identified

- Empty rule catalog is vacuously conformant under P1 (no rule to violate).
- `nil` package name slice must not panic in `rejectHyphenPrefixPackages`.
- Whitespace-only workflow name in `workflow_run.workflows` must still be treated as empty (hard error), per `hasNonEmptyWorkflowRunWorkflows`.
- Single incomplete rule entry (e.g. missing evidence) breaks the overall §3 Conformance predicate even if all other rules are complete.

## Notes for Future Runs

- Only CTR-013 and CTR-021 were formalized in concrete pre/post-condition detail this run (real function signatures inspected in `pkg/workflow/name_validation.go` and `pkg/workflow/agent_validation.go`). Remaining 19 rules were formalized only at the catalog-model level (P1-P3, P7-P10).
- Good candidates for deeper formalization next time: CTR-016 (Compile-Time Manifest Drift — has rich state in `safe_update_enforcement.go` with `collectSecretViolations`/`collectActionViolations`/`collectRedirectViolations`) and CTR-017 (Secret Leakage via Environment Variables — cross-cuts multiple validator functions).
- Consider a TLA+ state-machine model of the "Daily Optimizer Maintenance Protocol" (§6) itself in a future pass — it has a well-defined daily input/decision procedure that maps naturally to a state machine.
