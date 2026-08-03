# Formal Notes: specs/replace-label-compliance/README.md

**Last formalized**: 2026-08-03-16-14-02
**Notation**: TLA+ / Z3-style guard conjunction
**Issue**: (created via safeoutputs create_issue; number resolved post-run)

## Predicates

| ID | Predicate | Description |
|---|---|---|
| Q1 | `TransitionSetEmpty` | Empty/absent allowed-transitions ⇒ any (from,to) pair allowed |
| Q2 | `TransitionExactMatch` | Pair must match at least one listed (from,to) entry exactly |
| Q3 | `TransitionLayeredAfterAllowlist` | Blocklist/allowlist checks run before transition validation |
| Q4 | `TransitionRejectedYieldsSoftSkip` | Disallowed transition -> success=false, not a workflow failure |
| Q5 | `PostSetLabelsAddPresent` (RL-057a) | label_to_add must appear in setLabels response |
| Q6 | `PostSetLabelsRemoveAbsent` (RL-057b) | label_to_remove must not appear in setLabels response |
| Q7 | `PartialSuccessRejected` (RL-058) | Violating Q5/Q6 -> rejected outcome, success=false |
| Q8 | `PartialSuccessNoNewErrorCode` (RL-059) | Rejected outcome reuses SETLABELS_FAILED, no new error code |
| Q9 | `TransitionConfigShape` | LabelTransition{From,To} round-trips YAML from/to keys |

## Key Invariants

- Blocklist and allowlist evaluation (RL-001–RL-003) are security boundaries and MUST run before allowed-transitions checks.
- allowed-transitions is an additive state-machine constraint, not a replacement for allowlist/blocklist.
- The pre-existing formal suite (`pkg/workflow/replace_label_formal_test.go`, P1–P15 + 3 edges) already fully covers glob semantics, allowlist/blocklist ordering, schema, count gate, label-set computation, staged mode, and required-labels/title-prefix gates — this run intentionally targeted the two remaining gaps (transitions, post-setLabels verification).

## Edge Cases Identified

- Empty `setLabels` response array (all labels unexpectedly removed) must fail the add-presence check.
- Self-transition (`from == to`) is not implicitly allowed unless explicitly listed.
- Duplicate entries in `allowed-transitions` must not change the allow/deny decision (idempotent set semantics).

## Notes for Future Runs

- **Implementation gap found**: `actions/setup/js/replace_label.cjs` does NOT implement RL-057/RL-058/RL-059 post-`setLabels` response verification — it trusts a 200 response unconditionally. The new test suite formalizes the spec requirement against a stub (`formalVerifyPostSetLabels`), not the production handler. A follow-up could file a separate implementation-gap issue and/or wire this stub into the real JS handler once implemented, then extend `replace_label.test.cjs` accordingly.
- `allowed-transitions` config plumbing already exists end-to-end (Go struct → handler config builder → JS handler enforcement) — only the formal test coverage was missing.
- Next specs due for rotation: aw-harness.md, awf-config-sources-compliance/README.md, awf-config-sources-spec.md, etc. (processed list resets after 14 days per rotation.json cache).
