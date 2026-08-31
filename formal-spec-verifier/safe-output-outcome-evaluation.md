# Formal Notes: safe-output-outcome-evaluation.md

**Last formalized**: 2026-08-31-15-31-26
**Notation**: TLA+ / Z3 / F*
**Issue**: created (see run 33408692217)

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
- 2026-08-31 run: current code (`pkg/cli/outcome_eval.go`, `pkg/cli/outcome_evaluation.go`) actually uses `OutcomeStatus`/`EvidenceStrength` types (not the older `OutcomeResult` naming noted above) with constants `OutcomeStatusAccepted/Rejected/Pending/Ignored/Skipped/Unknown/Lifecycle/LifecycleClose/Error`; `OutcomeReport` embeds `OutcomeEvaluation{OutcomeStatus, EvidenceStrength, Signal}`. This run produced a fresh, tighter 10-predicate formalization (P1-P10) focused on Norms #1-4 (API failure handling) and Provenance Limits #1/#3 (bot detection), plus per-type mappings for `create_pull_request`, `create_issue`, `add_comment`, and the `zero_touch` derivation — complementary to but not a full replacement of the prior 15-predicate set above (P1-P15), which still holds useful detail on `close_issue` ordering, `dispatch_workflow` time windows, and `findPRByTimestamp` fallback lookups not re-verified this run.
- Still open for a future run: verify whether `ZeroTouch` in the current `OutcomeReport` struct is computed from `HumanComments`+`HumanReviews`+`HumanEdits` combined or a subset — this run's P10 assumed `humanEdits==0` only (matching the spec's "Common OTel Attributes" table prose), which may need reconciling against the 2026-08-08 finding that the real `evalCreatePullRequest` only checks `HumanComments`/`HumanReviews`.
