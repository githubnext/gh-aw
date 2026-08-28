# Formal Notes: intent-attribution-agent-governance.md

**Last formalized**: 2026-08-28-19-32-00
**Notation**: TLA+-style state predicates + Z3-style schema/enum constraints
**Issue**: (created via safe-output create_issue; number assigned post-processing)

## Predicates

| ID | Predicate | Description |
|---|---|---|
| P1 | `AttributionPrecedenceOrder` | Explicit intent > single closing issue > artifact-label fallback |
| P2 | `SingleSourcePerRecord` | Each attribution record has exactly one source |
| P3 | `AmbiguousRootDetection` | 2+ distinct closing issues, no explicit override => ambiguous, source=closing_issue |
| P4 | `AmbiguousNeverArbitrary` | No first/last/random resolution of ambiguity |
| P5 | `FailClosedSafestPolicy` | unlinked/ambiguous => safest policy regardless of matching rules |
| P6 | `PolicyDeterminism` | Same inputs => same compiled policy, always |
| P7 | `IntentPolicyConfigLabelDimensionEnum` | dimension in {priority, domain, risk, initiative} |
| P8 | `IntentPolicyConfigScoringStrategyEnum` | strategy in {max, sum} |
| P9 | `IntentPolicyConfigMultipleRootsDefault` | multiple_roots defaults to "ambiguous"; only {ambiguous, first} valid |
| P10 | `RulesPrecedenceOrdering` | Every rule requires a stable non-empty id; rules ordered highest->lowest precedence |

## Key Invariants

- Intent determines authority; execution produces evidence (central spec principle).
- Fail-closed policy: `autonomy=propose_only`, `write_scope=none`, `human_approval_required=true`, `auto_merge_allowed=false`, `max_attempts=1`.
- A pull request is an execution artifact, not an objective — multiple PRs on one issue must not multiply completed-objective counts.

## Edge Cases Identified

- Empty `labels` map in `.github/intent-policy.json` (required field) must be rejected.
- Unsupported schema `version` (anything other than `1`) must be rejected.
- No attribution sources present (no explicit intent, no closing issues, no labels) must resolve to `unlinked`, not crash or silently map to a default.

## Notes for Future Runs

- `pkg/intent` already has strong formal test coverage for `Resolver` and `PolicyCompiler` (see `intent_formal_test.go`, `governance_formal_test.go`, `compliance_fixtures_formal_test.go`, `resolver_test.go`, `spec_test.go`). This run's new test file focuses specifically on the **not-yet-implemented** `.github/intent-policy.json` schema validation (`IntentPolicyConfig` stub), since no concrete parser exists yet in the codebase.
- Future runs on this spec should watch for: (1) landing of a real `.github/intent-policy.json` parser in `pkg/intent` — once it exists, replace the stub types in the generated test file with real imports; (2) the "Escalation norm" (3+ consecutive CI runs with unaddressed sync warnings => compliance failure) is not yet covered by any Go test — a good candidate for the next formalization pass; (3) the non-normative maintenance split proposal (splitting spec into companion files) may change section anchors — re-verify predicate source citations if that split happens.
- Cross-spec dependency: `specs/intent-attribution-compliance/README.md` (already processed) contains the YAML compliance fixtures referenced by this spec's "Compliance Fixtures" section — those fixtures are already exercised by `compliance_fixtures_formal_test.go`.
