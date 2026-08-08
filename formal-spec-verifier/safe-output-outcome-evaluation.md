# Formal Notes: safe-output-outcome-evaluation.md

**Last formalized**: 2026-08-08-15-35-30
**Notation**: TLA+ / Z3 / F*
**Issue**: created (see run 31264760862)

## Predicates

| ID | Predicate | Description |
|---|---|---|
| P1 | `OutcomeExclusivity` | Each evaluation produces exactly one outcome from the defined set |
| P2 | `NoTerminalOnAPIFailure` | Evaluators MUST NOT emit accepted/rejected when API returns 5xx or rate-limit |
| P3 | `Http404IsTerminal` | 404 responses are terminal; deleted issue/PR → rejected, transient target → ignored |
| P4 | `PRMergedImpliesAccepted` | create_pull_request: merged==true ↔ accepted (grounded in `evalCreatePullRequest`) |
| P5 | `PRClosedNoMergeImpliesRejected` | create_pull_request: state==closed && merged==false ↔ rejected |
| P6 | `PROpenImpliesPending` | create_pull_request: neither merged nor closed → pending |
| P7 | `LabelRetentionAllImpliesAccepted` | add_labels: all labels retained → accepted; partial → rejected |
| P8 | `ZeroTouchDerivation` | zero_touch=true iff accepted AND human_comments==0 AND human_reviews==0 |
| P9 | `CloseStickiness` | close_issue/close_pull_request: still closed with lifecycle actor → lifecycle_close |
| P10 | `BotProvenanceObservable` | Bot actions classified by visible actor identity only (`isBotUser`) |
| P11 | `evalCreatePullRequestPost` | Combined pre/post contract across merged/closed/open PR states |
| P12 | `MissingIdentifiersImpliesError` | num==0 or repo=="" → OutcomeError |
| P13 | `TimeBoundedEvaluation` | Evaluation only permitted at/after the 48h default delay |
| P14 | `APIErrorNotTerminal` | Any `ghAPIGet`/`ghAPIGetArray` error → OutcomeError, never accepted/rejected |
| P15 | `ZeroTouchRequiresNoReviews` | zero_touch=true implies accepted AND zero human comments AND zero human reviews (grounded in `pkg/cli/outcome_eval_pr.go`) |

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
- 2026-08-08 run: read `pkg/cli/outcome_eval.go` and `pkg/cli/outcome_eval_pr.go` directly. Confirmed real type is `OutcomeResult` with 8 constants (including `OutcomeUnknown`, not in original spec table) and `OutcomeReport` struct with `HumanComments`, `HumanReviews`, `ZeroTouch`, `EvalError` fields.
- `evalCreatePullRequest` sets `ZeroTouch` only inside the `if report.Result == OutcomeAccepted` branch and only considers `HumanComments`/`HumanReviews`, not `HumanEdits` — the original P12 `ZeroTouchConstraint` (human_edits==0) was imprecise vs. the actual PR evaluator implementation; refined to P8/P15 using human_comments+human_reviews for the PR case.
- `evalCreatePullRequest` has a `findPRByTimestamp` fallback searching recent PRs by `github-actions[bot]` author when the manifest doesn't record a PR number — worth formalizing as a lookup-correctness property in a future run (similar to the dispatch_workflow time-window note above).
- Future run should read `pkg/cli/outcome_eval_issue.go`, `outcome_eval_label.go`, `outcome_eval_generic.go`, and `outcome_eval_workflow.go` directly to refine P6/P7/P9/P11 (create_issue, add_labels, close_sticky, dispatch_workflow) against real code rather than spec prose alone, since the PR evaluator review revealed spec/implementation nuances (e.g., ZeroTouch field composition) not obvious from the markdown spec.
