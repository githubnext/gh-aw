# Formal Notes: security-architecture-spec.md

**Last formalized**: 2026-08-05-16-10-21
**Notation**: TLA+ (gate sequence) / Z3-style guard conjunction (config predicates)
**Issue**: pending (see workflow run)

## Predicates

| ID | Predicate | Description |
|---|---|---|
| CC01 | `ConcurrencyGroupAlwaysConfigured` | Every workflow (issue/PR/push/dispatch) emits a non-empty `concurrency:` block with a `group:` key (RS-16, RS-17) |
| CC02 | `ConcurrencyGroupIncludesWorkflowIdentity` | Distinct workflow names produce distinct concurrency groups (RS-18) |
| CC03 | `CommandTriggerNeverCancelsInProgress` | `isCommandTrigger=true` forces `cancel-in-progress` off regardless of `on:` shape |
| CC04 | `PullRequestWorkflowEnablesCancelInProgress` | PR-triggered, non-command workflows enable `cancel-in-progress: true` (RS-21) |
| CC05 | `NonPRNonCommandOmitsCancelInProgress` | Schedule/push/dispatch workflows omit `cancel-in-progress` (RS-22) |
| CC06 | `BotSelfCancelRiskDetection` | `issue_comment`-triggered workflows are flagged as bot-self-cancel risk; push is not |
| CC07 | `ConcurrencyGateIsFirstInSequence` | Concurrency gate precedes freshness/repo-trust/actor-auth/credential/network/output/termination gates (Section 11.9) |
| CC08 | `GeneratedConcurrencyYAMLShape` | Generated YAML is well-formed: `concurrency:` header, non-empty `group:` value, conditional `cancel-in-progress:` line |

## Key Invariants

- Concurrency control is mandatory for every workflow shape (never absent).
- `cancel-in-progress` is a three-way decision: forced-off for command triggers, forced-on for PR workflows, off-by-default otherwise.
- The runtime enforcement sequence (Section 11.9) is strictly ordered; concurrency gate setup (item 1) must run before freshness (item 2) and all downstream gates, terminating in the fail-closed audit gate (item 8).
- Bot self-cancel risk (`hasBotSelfCancelRisk`) is a documented mitigation for issue_comment-triggered workflows that post their own comments back, which could otherwise re-trigger and cancel the original run under a shared, cancel-enabled group.

## Edge Cases Identified

- Command-triggered workflow with a PR `on:` trigger — cancel-in-progress must still be disabled (command carve-out wins over PR default).
- Mixed/dispatch-only workflows still require a concurrency group (no "opt out" path).
- issue_comment trigger vs. push trigger for bot-self-cancel-risk detection (true vs false).

## Notes for Future Runs

- This spec (`specs/security-architecture-spec.md`) already has 3 companion formal test files covering P1-P10 (`security_architecture_formal_test.go`), PM10/AppG (`security_architecture_pm10_formal_test.go`), and SG01-07/CS/RS (`security_architecture_sg_formal_test.go`). This run added a 4th file covering the previously-uncovered Concurrency Control section (11.8) and Runtime Enforcement Operations Sequence (11.9).
- Remaining under-covered areas for future runs: Section 9 (Threat Detection Layer detection methods/output schema), Section 10.6-10.8 (Action Pinning, Deprecated Features, Compile-Time vs Runtime tradeoffs), and Section 7.6 (Role-Based Access Control) — none of these appear to have dedicated formal predicate coverage yet based on a scan of existing `*_formal_test.go` files.
- `replace-label-spec.md` was skipped in favor of this spec because it already has comprehensive formal coverage (`replace_label_formal_test.go` + `replace_label_transitions_formal_test.go`, 887 lines, P1-P15 plus transition gates) — low marginal value to duplicate.
