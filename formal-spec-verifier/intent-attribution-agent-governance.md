# Formal Notes: intent-attribution-agent-governance.md

**Last formalized**: 2026-07-30-16-13-07
**Notation**: TLA+ / Z3 / F*
**Issue**: (created via safe-output; number resolved post-run)

## Predicates

| ID | Predicate | Description |
|---|---|---|
| P1 | `ExplicitIntentPrecedence` | Explicit workflow intent always resolves first, overriding otherwise-ambiguous candidates |
| P2 | `SingleClosingIssueMapped` | Exactly one closing issue -> mapped/closing_issue |
| P3 | `AmbiguousOnMultipleRoots` | 2+ closing issues -> ambiguous, order-independent, never arbitrary pick |
| P4 | `ArtifactLabelFallback` | Zero closing issues + labels present -> artifact_labels fallback |
| P5 | `UnlinkedWhenNoSource` | Zero closing issues + no labels -> unlinked |
| P6 | `AmbiguousNotMapped` | Ambiguous status MUST NOT be treated as mapped for reporting/authorization |
| P7 | `FailClosedPolicy` | unlinked/ambiguous/suggested/unmapped -> safest policy (propose_only, write_scope=none, human_approval_required, no auto-merge, max_attempts=1) |
| P8 | `PolicyDeterminism` | Identical attribution inputs -> identical policy output |
| P9 | `RiskClassificationOrder` | Explicit risk wins; else derived by domain/priority rules (security+critical=high, production=high, infrastructure=medium, documentation=low, else unknown) |
| P10 | `PrecedenceOrdering` | organization > repository > intent > workflow > agent_request |
| P11 | `NoElevatedAuthorityOnAbsentAttribution` | Unresolved attribution never grants elevated autonomy/auto-merge/max_attempts |

## Key Invariants

- Resolution order is strictly deterministic: explicit intent > closing issues > labels > unlinked.
- Ambiguity (2+ closing issues) must never be resolved by arbitrary selection (first/last/random).
- Fail-closed behavior is the safety backstop: any indeterminate status collapses to the safest policy regardless of what an elevated/requested policy would have been.
- Policy precedence is strictly layered; lower layers cannot weaken higher layers.

## Edge Cases Identified

- Zero closing issues AND zero labels → unlinked (fully indeterminate).
- Exactly one closing issue → deterministic mapped (no ambiguity).
- 2+ closing issues in different orderings must all resolve identically to ambiguous (order independence tested with 3 permutations).
- Explicit risk field present should short-circuit all derived-risk domain rules.
- Unknown/unrecognized domain (e.g. "marketing") must resolve to "unknown" risk, not error.

## Notes for Future Runs

- The spec is still "Partially Implemented" (Phase 1 of 7 per the Implementation Phases section). No concrete Go resolver package exists yet in `pkg/` — the test file uses a full stub reproduction of `Resolver.Resolve`, `ResolveRisk`, and `ExecutionPolicy` derivation logic taken directly from the spec's embedded Go code blocks.
- Future runs formalizing this spec further could add: OpenTelemetry span/metric predicates (spec §"OpenTelemetry"/"Metrics"), CLI "Explain policy before execution" predicate, and Evidence record / provenance predicates (§"Decision provenance", §"Evidence record").
- Cross-spec dependency: `.github/intent-policy.json` schema (§"Intent configuration") could be jointly formalized with `specs/replace-label-spec.md`'s gate-check mechanisms (required-labels/required-title-prefix) since both implement a "skip, don't fail" gating pattern.
- `specs/replace-label-spec.md` remains unprocessed and is a good next candidate (large, well-structured RL-0xx numbered requirements, Go-style processing pipeline).
