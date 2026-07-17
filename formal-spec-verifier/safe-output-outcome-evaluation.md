# Formal Notes: safe-output-outcome-evaluation.md

**Last formalized**: 2026-07-17-15-53-52
**Notation**: TLA+ / Z3 / F*
**Issue**: pending

## Predicates

| ID | Predicate | Description |
|---|---|---|
| P1 | `OutcomeExclusivity` | Each evaluation produces exactly one outcome from the defined set |
| P2 | `NoTerminalOnAPIFailure` | Evaluators MUST NOT emit accepted/rejected when API returns 5xx or rate-limit |
| P3 | `404IsTerminal` | 404 responses are terminal; deleted issue/PR → rejected, transient target → ignored |
| P4 | `PRMergedImpliesAccepted` | create_pull_request: merged==true ↔ accepted |
| P5 | `PRClosedNoMergeImpliesRejected` | create_pull_request: state==closed && merged==false ↔ rejected |
| P6 | `IssueCompletedImpliesAccepted` | create_issue: state==closed && state_reason==completed → accepted |
| P7 | `IssueBotCloseImpliesLifecycle` | create_issue: state==closed && state_reason==not_planned && bot_close → lifecycle |
| P8 | `LabelRetentionAllImpliesAccepted` | add_labels: all labels retained → accepted |
| P9 | `CloseStickiness` | close_issue/close_pull_request: still closed with lifecycle actor → lifecycle_close |
| P10 | `UpdateRetentionMatchAfterState` | update_issue/update_pr: fields match after_state snapshot → accepted |
| P11 | `DispatchWorkflowSuccessImpliesAccepted` | dispatch_workflow: run conclusion==success → accepted |
| P12 | `ZeroTouchConstraint` | zero_touch=true iff accepted AND human_edits==0 |
| P13 | `BotProvenanceObservable` | Bot actions classified by visible actor identity only |

## Key Invariants

- Outcomes are exclusive: exactly one outcome per evaluation
- API transient failures (5xx, rate-limit) block terminal outcomes
- Bot provenance is determined by visible GitHub actor identity, not hidden AI assistance
- `zero_touch` is derived from `accepted` and `human_edits==0`
- 404 on object → terminal classification (rejected or ignored depending on semantics)
- `pending` is the safe default when object state is unresolved

## Edge Cases Identified

- Comment deleted (404) → rejected, not ignored
- Comment minimized → rejected
- All labels removed → rejected (same code path as partial removal)
- Issue closed by bot with not_planned → lifecycle, not rejected
- PR merged but pushed commits were reverted → rejected
- Snapshots missing for update_issue/update_pr → unknown (not rejected or accepted)
- Rate-limit 403 vs auth 403: evaluator must distinguish by limit-exhaustion marker

## Notes for Future Runs

- The `close_issue` evaluation order is important: current state checked first (if open → rejected), then actor-based classification. This ordering is a key invariant.
- `dispatch_workflow` evaluation uses a time-window filter which introduces a lookup correctness property worth formalizing.
- `create_issue` bot detection via timeline API is an interesting correctness property.
- `update_issue` / `update_pr` use execution-time snapshots (before/after), making this a temporal consistency property.
