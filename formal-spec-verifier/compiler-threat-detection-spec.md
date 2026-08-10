# Formal Notes: compiler-threat-detection-spec.md

**Last formalized**: 2026-08-10-15-42-47
**Notation**: Z3 / SMT-LIB, F*
**Issue**: (created via safe-output; number assigned post-processing)

## Predicates

| ID | Predicate | Description |
|---|---|---|
| P1 | `SafeRefAccepted` | `ValidateGitRef` accepts iff non-empty, no leading `-`, no NUL byte, no `..` |
| P2 | `SafePathAccepted` | `ValidateGitPath` accepts iff non-empty, no leading `-`, not absolute, cleaned form has no `..` traversal |
| P3 | `HyphenPrefixRejected` | Any ref/path starting with `-` is rejected (CWE-88 argument injection guard) |
| P4 | `NulByteRejected` | Refs containing a NUL byte are rejected |
| P5 | `TraversalRejected` | Refs/paths containing `..` are rejected |
| P6 | `AbsolutePathRejected` | Absolute paths are rejected even without `..` |
| P7 | `EmptyValueRejected` | Empty ref or path is always rejected |
| P8 | `BashWildcardIsSafe` | `bash: ["*"]` / `[":*"]` is not an explicit restriction (CTR-023) |
| P9 | `BashAbsentIsSafe` | Absent/nil `tools.bash`, or `bash: true`, is not an explicit restriction |
| P10 | `BashFalseIsRestriction` | `bash: false` is an explicit restriction |
| P11 | `BashEmptyListIsRestriction` | `bash: []` is an explicit restriction |
| P12 | `BashNamedListIsRestriction` | Non-wildcard named command list is an explicit restriction |

## Key Invariants

- `ValidateGitRef`/`ValidateGitPath` (pkg/gitutil/gitutil.go) are hard security boundaries applied in all modes (compile-time, strict and non-strict) — not mode-sensitive like most other CTR rules.
- `HasBashExplicitRestriction` (pkg/workflow/agent_validation.go, exported) has an early-exit: any wildcard token (`*` or `:*`) anywhere in the bash command list makes the entire config "safe", even if other named commands are present.
- Carried forward from 2026-07-28 run: catalog-model predicates P1-P10 (RuleModelComplete, DeterministicResponse, SecureOutcome, VersionSyncInvariant, Conformance, etc.) remain valid — see prior notes below for full list, not re-derived this run to keep scope tight.

## Edge Cases Identified

- `nil` tools map must not panic in `HasBashExplicitRestriction` (returns false).
- `bash: nil` (present key, nil value) must be treated the same as absent (safe).
- Empty git ref/path string is a distinct rejection case from hyphen-prefix or traversal (must return the "must not be empty" message, not a generic error).
- Absolute path rejection (P6) is independent of traversal detection (P5) — `/etc/passwd` has no `..` yet must still be rejected.

## Notes for Future Runs

- This run drilled into CTR-022 and CTR-023 (both newest, v1.0.20) using real exported functions (`ValidateGitRef`, `ValidateGitPath`, `HasBashExplicitRestriction`) with full acceptance/rejection boundary coverage.
- CTR-023's `validateBashCommandAllowlistSupport` (unexported, on `*Compiler`) was intentionally NOT tested directly — it requires a full `CodingAgentEngine` + `Compiler` fixture. Only the exported, engine-independent `HasBashExplicitRestriction` predicate layer was formalized. A future run could add an engine-capability-parameterized test matrix (codex vs. copilot/claude/gemini) for the full `validateBashCommandAllowlistSupport` decision table (including `BashDisable` interaction via `hasBashFullyDisabled`).
- CTR-016 (Compile-Time Manifest Drift) and CTR-017 (Secret Leakage via Environment Variables) remain good candidates for deeper formalization per prior notes — not addressed this run.
- Still worth a future TLA+ state-machine model of the "Daily Optimizer Maintenance Protocol" (§6).

---
## Prior Run (2026-07-28) Notes — Catalog-Level Model

Predicates: P1 RuleModelComplete, P2 DeterministicResponse, P3 SecureOutcome, P4 NoHyphenPrefixPassthrough (CTR-013),
P5 WorkflowsFieldMandatory (CTR-021), P6 BranchScopeModeSensitive (CTR-021), P7 DeprecationRetainsEntry,
P8 DeprecatedTestsNotRequired, P9 VersionSyncInvariant, P10 Conformance (conjunction of P1-P9).
