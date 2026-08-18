# Formal Notes: intent-attribution-agent-governance.md

**Last formalized**: 2026-08-18-15-43-32
**Notation**: Z3-style guard conjunction / propositional logic
**Issue**: (see repository issue tracker for this run's created issue)

## Predicates

| ID | Predicate | Description |
|---|---|---|
| P1 | `RiskResolutionDeterminism` | `ResolveRisk(intent)` is a pure function of (Risk, Domains, Priority) |
| P2 | `RiskExplicitOverride` | Explicit `intent.Risk` always wins over derived rules |
| P3 | `RiskSecurityCriticalHigh` | `domains ∋ security ∧ priority = critical ⇒ high` |
| P4 | `RiskProductionHigh` | `domains ∋ production ⇒ high` |
| P5 | `RiskInfrastructureMedium` | `domains ∋ infrastructure ⇒ medium` |
| P6 | `RiskDocumentationLow` | `domains ∋ documentation ⇒ low` |
| P7 | `RiskUnknownDefault` | No matching rule ⇒ `unknown` |
| P8 | `RiskPrecedenceOrder` | Security+critical rule evaluated before production/infrastructure/documentation |
| P9 | `AuthorizeToolDeniedWins` | Deny list takes precedence over allow list |
| P10 | `AuthorizeToolAllowlistGate` | Non-nil AllowedTools rejects tools not listed |
| P11 | `AuthorizeToolUnrestricted` | `nil` AllowedTools means unrestricted |
| P12 | `AuthorizeToolEmptyDenyAll` | Non-nil empty AllowedTools means deny-all |
| P13 | `SafestDefaultFailClosed` | `unlinked`/`ambiguous` status forces safest policy (already implemented and verified against `pkg/intent/policy.go`) |

## Key Invariants

- Unknown or ambiguous intent must never grant elevated authority (fail-closed default).
- Policy merging must preserve stricter, higher-precedence constraints when multiple rules match.
- `ResolveRisk` and `Authorizer.AuthorizeTool` are specified but **not yet implemented** in `pkg/intent` — the spec's own "Authorizer.AuthorizeTool Implementation Audit" section documents this gap; all `ExecutionPolicy` enforcement fields except none are currently wired to runtime.
- `pkg/intent/policy.go` already implements `PolicyCompiler.Compile` fail-closed behavior for `Unlinked`/`Ambiguous` — this predicate (P13) was verified directly against real code, not a stub.

## Edge Cases Identified

- Fully empty intent record (no Risk, Domains, Priority) must resolve to `unknown`, not panic or empty string.
- `AuthorizeTool` must not panic when both `AllowedTools` and `DeniedTools` are nil (fully unrestricted policy).
- Multiple matching policy rules must merge with stricter-wins semantics; a later lenient rule must not silently override an earlier strict one.

## Notes for Future Runs

- `pkg/intent` already has `policy.go`, `resolver.go`, `slices.go` plus existing formal test files (`intent_formal_test.go`, `compliance_fixtures_formal_test.go`, `spec_test.go`, `resolver_test.go`) — future formalizations of this spec should check those files first to avoid duplicating predicate coverage (e.g. resolution-order predicates for `Resolver.Resolve`/`ResolvePullRequest` are likely already covered there).
- Not yet formalized in this run: OpenTelemetry/metrics section, CLI section (`explain policy`, `report outcomes`), Evidence/Outcome record schemas, and the multi-root/fractional-attribution future-policy section. Good candidates for a follow-up pass.
- `ResolveRisk` and `Authorizer.AuthorizeTool` remain unimplemented in the codebase as of this run — once implemented, replace the stubs in the generated test file with real calls.
