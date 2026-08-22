# Formal Notes: intent-attribution-compliance/README.md

**Last formalized**: 2026-08-22-15-35-35
**Notation**: TLA+ / Z3-style guard conjunction
**Issue**: (created via safe-output; number resolved post-run)

## Predicates

| ID | Predicate | Description |
|---|---|---|
| F1 | `ExplicitIntentWins` | Explicit intent metadata is sole attribution source, overrides conflicting closing-issue/label signals |
| F2 | `AmbiguousRootStatus` | 2+ distinct closing issues, no explicit intent -> ambiguous / closing_issue / no intent key |
| F3 | `UnlinkedFailsClosed` | No explicit intent, no closing issues, no labels -> unlinked / none |
| F4 | `SafestPolicyOnIndeterminate` | ambiguous/unlinked both compile to propose_only / write_scope=none / approval required / no auto-merge / max_attempts=1 |
| F5 | `MappedStatusPermitsRelaxedPolicy` | mapped/explicit status is not universally forced fail-closed; a matching rule may grant relaxed policy |
| F6 | `PolicyDeterminism` | identical attribution inputs always produce identical compiled policy |
| F7 | `SingleSourcePerRecord` | every fixture resolves to exactly one attribution source, never a blend |
| F8 | `FailClosedIgnoresRuleConfig` | ambiguous/unlinked status ignores even a matching permissive wildcard rule (short-circuit happens before rule matching) |
| F9 | `ExplicitLabelsDoNotOverrideMappedSource` | explicit intent stays the sole source even when present labels would independently resolve via artifact labels |

## Key Invariants

- The three compliance fixtures (explicit-intent-wins, ambiguous-root-closing-issues, unlinked-pr-fail-closed) already exist as YAML files in `specs/intent-attribution-compliance/` (not just described as "future" in the README table).
- Fail-closed applies uniformly to both `ambiguous` and `unlinked` statuses via the same safest-policy tuple.
- Fail-closed is conditional on status, not universal — mapped/explicit status can receive a relaxed policy from a matching rule (proven by F5, the converse of F4).
- Order independence for ambiguous resolution: reordering closing issues must not change the outcome.

## Edge Cases Identified

- Explicitly empty (non-nil) labels slice must still resolve to unlinked, not accidentally trigger artifact-label fallback.
- Explicit intent overrides even a single (non-ambiguous) closing issue, proving precedence is unconditional, not merely an ambiguity tiebreaker.
- Reordered closing issues (issue #121 first vs #120 first) must produce identical ambiguous status/source.

## Notes for Future Runs

- 2026-08-22 run: extended the compliance suite with F8 (fail-closed short-circuit ignores a matching permissive wildcard rule, proving the ambiguous/unlinked branch never reaches rule matching) and F9 (explicit intent source is preserved even when present labels would independently resolve via artifact-label attribution — precedence is absolute, not signal-strength-based). New tests live in a separate file (`intent_compliance_extended_formal_test.go`) to avoid touching the existing fixture-anchored suite.
- This formalization targets the fixture-level compliance README specifically and is complementary to the broader parent spec (`specs/intent-attribution-agent-governance.md`, formalized 2026-07-30) — predicates here are scoped to the 3 required fixture scenarios rather than the full resolver/policy surface.
- Existing test files in `pkg/intent/` (`intent_formal_test.go`, `resolver_test.go`, `spec_test.go`) already cover general resolver/policy behavior; the new `compliance_fixtures_formal_test.go` uses distinct fixture-anchored function names (`TestFormalFixture_*`) to avoid collisions and to trace directly back to the 3 YAML fixture files by name and field values.
- Future work: extend fixtures directory per the README's guidance (fixture files should record input artifact shape, expected attribution source/status, and expected compiled execution policy) — could add machine-readable YAML-driven test loading instead of hardcoded Go fixture builders.
- Cross-spec dependency: `specs/replace-label-spec.md` already processed (2026-08-07); consider `specs/otel-observability-spec.md` (last processed 2026-08-02, oldest by date) as a strong future candidate given rotation age.
