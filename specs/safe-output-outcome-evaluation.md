---
title: Safe Output Outcome Evaluation Specification
version: 1.0.0
status: Working Draft
date: 2026-05-15
last_updated: 2026-05-16
---

# Safe Output Outcome Evaluation Specification

Every safe output type has a measurable outcome. This spec defines the exact evaluation logic for each type: what to check, how to classify the outcome, and what OTel attributes to emit.

## Principles

1. **Same as a repository observer would check.** If this action happened on GitHub, how would an observer decide it was good from visible repository state?
2. **Direct Outcome Only.** We check whether the action stuck, not whether it caused downstream effects.
3. **Bot-aware, not provenance-perfect.** Distinguish bot/app-visible closes and edits from non-bot ones, but do not assume non-bot actors are unassisted by AI.
4. **Time-bounded.** Check outcomes after a configurable delay (default: 48 hours).

## Norms

The key words **MUST**, **MUST NOT**, and **SHOULD** in this document are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

1. Outcome evaluation workers **MUST** treat GitHub API `404` responses as terminal for deleted or inaccessible objects and classify according to object semantics (for example, a deleted issue/PR should be `rejected`, while a transient target with no persistent evaluable object should be `ignored`).
2. Outcome evaluation workers **MUST** treat GitHub API `5xx` responses as transient infrastructure failures and return `pending` for that check cycle while recording retry metadata (`status_code`, `retry_after`, `attempt`).
3. Outcome evaluation workers **MUST** treat GitHub API rate-limit responses (`403` with limit exhaustion or `429`) as transient and **SHOULD** reschedule evaluation using the reset window before emitting final outcomes.
4. Outcome evaluation workers **MUST NOT** emit `accepted` or `rejected` when API failures prevent verification of the authoritative object state.

## Provenance Limits

Outcome evaluation is based on observable GitHub state and actor identity, not hidden authoring provenance.

1. Outcome evaluation workers **MUST** treat actions performed by visible bot or app identities as bot/app actions.
2. Outcome evaluation workers **MUST** treat actions performed by visible non-bot user identities as non-bot actions, even if those users may have used Copilot or another AI assistant.
3. Outcome evaluation workers **MUST NOT** infer hidden AI assistance when GitHub exposes only a normal user identity.
4. Metrics and fields that use `human_*` names are historical names. In this specification they mean actor-visible, non-bot activity unless explicit provenance metadata is available.
5. Implementations **SHOULD** prefer explicit provenance markers when available, such as bot identities, GitHub App identities, trace IDs, labels, commit trailers, or other durable metadata emitted by the workflow.

## Outcome Categories

Every evaluation produces one of these outcomes:

| Outcome | Meaning |
|---------|---------|
| `accepted` | The action was kept, merged, resolved, or engaged with |
| `rejected` | The action was undone, closed-as-not-planned, removed, or reverted |
| `ignored` | No observable non-bot interaction within the evaluation window |
| `pending` | The object has not reached a terminal state yet |
| `lifecycle` | Closed/removed by the workflow itself (e.g., `close-older-issues`) — not a rejection |
| `lifecycle_close` | Closed by lifecycle/noop bot policy and not reopened by a visible non-bot actor |

## Common OTel Attributes

Every outcome span carries these attributes:

| Attribute | Type | Description |
|-----------|------|-------------|
| `gh-aw.outcome.type` | string | Safe output type (e.g., `create_pull_request`) |
| `gh-aw.outcome.result` | string | One of: `accepted`, `rejected`, `ignored`, `pending`, `lifecycle`, `lifecycle_close` |
| `gh-aw.outcome.object_url` | string | GitHub URL of the affected object |
| `gh-aw.outcome.object_number` | int | Issue/PR/discussion number |
| `gh-aw.outcome.repo` | string | `owner/repo` |
| `gh-aw.outcome.source_run_id` | string | Workflow run that created this output |
| `gh-aw.outcome.source_trace_id` | string | Original OTLP trace ID |
| `gh-aw.outcome.created_at` | string | When the safe output was executed |
| `gh-aw.outcome.checked_at` | string | When this evaluation ran |
| `gh-aw.outcome.time_to_outcome_hours` | float | Hours from creation to terminal state |
| `gh-aw.outcome.human_comments` | int | Historical field name; means actor-visible non-bot comments on the object |
| `gh-aw.outcome.human_edits` | int | Historical field name; means actor-visible non-bot edits before acceptance |
| `gh-aw.outcome.zero_touch` | bool | Accepted with no actor-visible non-bot modifications |

## Implementation

The following implementation areas are responsible for evaluation data capture, outcome classification plumbing, and runtime event artifacts.

Status meanings:
- `implemented`: dedicated evaluator logic exists in both Go and JS.
- `partial`: dedicated evaluator exists in one runtime; the other relies on generic fallback logic.
- `not-started`: no dedicated evaluator exists yet; current behavior is generic/no-op only.

### Current Default Acceptance Map

This table summarizes the current runtime behavior in `pkg/cli/outcome_eval*.go`. It is intentionally about what the evaluator accepts today, not just the intended long-term spec semantics.

Rows marked `evalGenericSticky` fallback are generic existence checks, not type-specific acceptance logic.

| Output type | Current evaluator | `accepted` at a glance |
|-------------|-------------------|------------------------|
| `create_pull_request` | `evalCreatePullRequest` | merged |
| `create_issue` | `evalCreateIssue` | completed/closed |
| `add_comment` | `evalAddComment` | reacted to or replied to |
| `add_labels` | `evalAddLabels` | label retention |
| `add_reviewer` | `evalGenericSticky` fallback | review target exists |
| `update_issue` | `evalGenericSticky` fallback | issue still exists |
| `update_pull_request` | `evalGenericSticky` fallback | PR still exists |
| `close_issue` | `evalCloseSticky` | still closed |
| `close_pull_request` | `evalCloseSticky` | still closed |
| `close_discussion` | `evalCloseDiscussion` | none yet |
| `create_discussion` | `evalCreateDiscussion` | none yet |
| `update_discussion` | `evalGenericSticky` fallback | discussion target exists |
| `create_pull_request_review_comment` | `evalReviewComment` | none yet |
| `submit_pull_request_review` | `evalGenericSticky` fallback | PR still exists |
| `reply_to_pull_request_review_comment` | `evalGenericSticky` fallback | review target exists |
| `resolve_pull_request_review_thread` | `evalResolveThread` | none yet |
| `push_to_pull_request_branch` | `evalPushToPRBranch` | merged |
| `mark_pull_request_as_ready_for_review` | `evalMarkReady` | reviewed |
| `assign_to_agent` | `evalAssignToAgent` | merged or completed |
| `dispatch_workflow` | `evalGenericSticky` fallback | dispatch target exists |
| `autofix_code_scanning_alert` | `evalGenericSticky` fallback | alert target exists |
| `create_code_scanning_alert` | `evalGenericSticky` fallback | alert target exists |
| `link_sub_issue` | `evalGenericSticky` fallback | sub-issue link target exists |
| `hide_comment` | `evalHideComment` | none yet |
| `assign_milestone` | `evalAssignMilestone` | milestone still set |
| `update_project` | `evalGenericSticky` fallback | object still exists |
| `update_release` | `evalGenericSticky` fallback | object still exists |
| `noop` | explicit skip | skipped |
| `missing_tool` | explicit skip | skipped |

| Output type | Implementation status | Go implementation areas | JS/runtime implementation areas |
|-------------|------------------------|--------------------------|---------------------------------|
| `create_pull_request` | implemented | `pkg/workflow/safe_outputs_config.go`, `pkg/workflow/compiler_safe_outputs.go`, `pkg/cli/outcome_eval.go` (`evalCreatePullRequest`) | `actions/setup/js/safe_outputs_handlers.cjs`, `actions/setup/js/evaluate_outcomes.cjs` (PR-specific path) |
| `create_issue` | implemented | `pkg/workflow/safe_outputs_config.go`, `pkg/workflow/compiler_safe_outputs.go`, `pkg/cli/outcome_eval.go` (`evalCreateIssue`) | `actions/setup/js/safe_outputs_handlers.cjs`, `actions/setup/js/evaluate_outcomes.cjs` (issue-specific path) |
| `add_comment` | implemented | `pkg/workflow/safe_outputs_dispatch.go`, `pkg/cli/outcome_eval.go` (`evalAddComment`) | `actions/setup/js/safe_outputs_handlers.cjs`, `actions/setup/js/evaluate_outcomes.cjs` (issue-comment URL path) |
| `add_labels` | partial | `pkg/workflow/safe_outputs_allowed_labels_validation.go`, `pkg/cli/outcome_eval.go` (`evalAddLabels`) | `actions/setup/js/safe_outputs_handlers.cjs`, `actions/setup/js/evaluate_outcomes.cjs` (generic fallback) |
| `add_reviewer` | not-started | `pkg/workflow/add_reviewer.go`, `pkg/workflow/safe_outputs_config.go` | `actions/setup/js/add_reviewer.cjs`, `actions/setup/js/safe_outputs_handlers.cjs` |
| `update_issue` | not-started | `pkg/workflow/safe_outputs_config.go`, `pkg/cli/outcome_eval.go` (`evalGenericSticky` fallback) | `actions/setup/js/update_issue.cjs`, `actions/setup/js/evaluate_outcomes.cjs` (generic fallback) |
| `update_pull_request` | not-started | `pkg/workflow/safe_outputs_config.go`, `pkg/cli/outcome_eval.go` (`evalGenericSticky` fallback) | `actions/setup/js/update_pull_request.cjs`, `actions/setup/js/evaluate_outcomes.cjs` (generic fallback) |
| `close_issue` | partial | `pkg/workflow/safe_outputs_dispatch.go`, `pkg/cli/outcome_eval.go` (`evalCloseSticky`) | `actions/setup/js/close_issue.cjs`, `actions/setup/js/evaluate_outcomes.cjs` (generic fallback) |
| `close_pull_request` | partial | `pkg/workflow/safe_outputs_dispatch.go`, `pkg/cli/outcome_eval.go` (`evalCloseSticky`) | `actions/setup/js/close_pull_request.cjs`, `actions/setup/js/evaluate_outcomes.cjs` (generic fallback) |
| `close_discussion` | partial | `pkg/workflow/safe_outputs_dispatch.go`, `pkg/cli/outcome_eval.go` (`evalCloseDiscussion`) | `actions/setup/js/close_discussion.cjs`, `actions/setup/js/evaluate_outcomes.cjs` (generic fallback) |
| `create_discussion` | partial | `pkg/workflow/safe_outputs_dispatch.go`, `pkg/cli/outcome_eval.go` (`evalCreateDiscussion`) | `actions/setup/js/create_discussion.cjs`, `actions/setup/js/evaluate_outcomes.cjs` (generic fallback) |
| `update_discussion` | not-started | `pkg/workflow/safe_outputs_config.go`, `pkg/cli/outcome_eval.go` (`evalGenericSticky` fallback) | `actions/setup/js/update_discussion.cjs`, `actions/setup/js/evaluate_outcomes.cjs` (generic fallback) |
| `create_pull_request_review_comment` | partial | `pkg/workflow/safe_outputs_config.go`, `pkg/cli/outcome_eval.go` (`evalReviewComment`) | `actions/setup/js/create_pr_review_comment.cjs`, `actions/setup/js/evaluate_outcomes.cjs` (generic fallback) |
| `submit_pull_request_review` | not-started | `pkg/workflow/safe_outputs_config.go`, `pkg/cli/outcome_eval.go` (`evalGenericSticky` fallback) | `actions/setup/js/submit_pr_review.cjs`, `actions/setup/js/evaluate_outcomes.cjs` (generic fallback) |
| `reply_to_pull_request_review_comment` | not-started | `pkg/workflow/safe_outputs_config.go`, `pkg/cli/outcome_eval.go` (`evalGenericSticky` fallback) | `actions/setup/js/reply_to_pr_review_comment.cjs`, `actions/setup/js/evaluate_outcomes.cjs` (generic fallback) |
| `resolve_pull_request_review_thread` | partial | `pkg/workflow/safe_outputs_config.go`, `pkg/cli/outcome_eval.go` (`evalResolveThread`) | `actions/setup/js/resolve_pr_review_thread.cjs`, `actions/setup/js/evaluate_outcomes.cjs` (generic fallback) |
| `push_to_pull_request_branch` | partial | `pkg/workflow/push_to_pull_request_branch_validation.go`, `pkg/cli/outcome_eval.go` (`evalPushToPRBranch`) | `actions/setup/js/push_to_pull_request_branch.cjs`, `actions/setup/js/evaluate_outcomes.cjs` (generic fallback) |
| `mark_pull_request_as_ready_for_review` | partial | `pkg/workflow/safe_outputs_config.go`, `pkg/cli/outcome_eval.go` (`evalMarkReady`) | `actions/setup/js/mark_pull_request_as_ready_for_review.cjs`, `actions/setup/js/evaluate_outcomes.cjs` (generic fallback) |
| `assign_to_agent` | partial | `pkg/workflow/safe_outputs_dispatch.go`, `pkg/cli/outcome_eval.go` (`evalAssignToAgent`) | `actions/setup/js/assign_to_agent.cjs`, `actions/setup/js/evaluate_outcomes.cjs` (generic fallback) |
| `dispatch_workflow` | not-started | `pkg/workflow/dispatch_workflow.go`, `pkg/cli/outcome_eval.go` (`evalGenericSticky` fallback) | `actions/setup/js/dispatch_workflow.cjs`, `actions/setup/js/evaluate_outcomes.cjs` (generic fallback) |
| `autofix_code_scanning_alert` | not-started | `pkg/workflow/safe_outputs_config.go`, `pkg/cli/outcome_eval.go` (`evalGenericSticky` fallback) | `actions/setup/js/autofix_code_scanning_alert.cjs`, `actions/setup/js/evaluate_outcomes.cjs` (generic fallback) |
| `create_code_scanning_alert` | not-started | `pkg/workflow/create_code_scanning_alert.go`, `pkg/cli/outcome_eval.go` (`evalGenericSticky` fallback) | `actions/setup/js/create_code_scanning_alert.cjs`, `actions/setup/js/evaluate_outcomes.cjs` (generic fallback) |
| `link_sub_issue` | not-started | `pkg/workflow/link_sub_issue.go`, `pkg/cli/outcome_eval.go` (`evalGenericSticky` fallback) | `actions/setup/js/link_sub_issue.cjs`, `actions/setup/js/evaluate_outcomes.cjs` (generic fallback) |
| `hide_comment` | partial | `pkg/workflow/hide_comment.go`, `pkg/cli/outcome_eval.go` (`evalHideComment`) | `actions/setup/js/hide_comment.cjs`, `actions/setup/js/evaluate_outcomes.cjs` (generic fallback) |
| `assign_milestone` | partial | `pkg/workflow/assign_milestone.go`, `pkg/cli/outcome_eval.go` (`evalAssignMilestone`) | `actions/setup/js/assign_milestone.cjs`, `actions/setup/js/evaluate_outcomes.cjs` (generic fallback) |
| `update_project` | not-started | `pkg/workflow/update_project.go`, `pkg/cli/outcome_eval.go` (`evalGenericSticky` fallback) | `actions/setup/js/update_project.cjs`, `actions/setup/js/evaluate_outcomes.cjs` (generic fallback) |
| `update_release` | not-started | `pkg/workflow/safe_outputs_config.go`, `pkg/cli/outcome_eval.go` (`evalGenericSticky` fallback) | `actions/setup/js/update_release.cjs`, `actions/setup/js/evaluate_outcomes.cjs` (generic fallback) |
| `noop` | implemented | `pkg/cli/outcome_eval.go` (explicit skip in `EvaluateOutcomes`) | `actions/setup/js/evaluate_outcomes.cjs` (`NOOP_TYPES`) |
| `missing_tool` | implemented | `pkg/cli/outcome_eval.go` (explicit skip in `EvaluateOutcomes`) | `actions/setup/js/missing_tool.cjs`, `actions/setup/js/evaluate_outcomes.cjs` (`NOOP_TYPES`) |

---

## 1. `create_pull_request`

**Question:** Was the PR merged?

**API:** `GET /repos/{owner}/{repo}/pulls/{number}`

**Evaluation:**

| Condition | Outcome |
|-----------|---------|
| `merged == true` | `accepted` |
| `state == "closed"` and `merged == false` | `rejected` |
| `state == "open"` | `pending` |

**Extra signals:**
- `human_edits`: historical field name; count actor-visible non-bot commits pushed by users other than the PR author after creation
- `human_comments`: historical field name; count actor-visible non-bot comments on the PR
- `zero_touch`: `accepted` and `human_edits == 0`, meaning no actor-visible non-bot follow-up
- `time_to_outcome_hours`: `merged_at - created_at` or `closed_at - created_at`

**Additional OTel attributes:**

| Attribute | Type | Description |
|-----------|------|-------------|
| `ghaw.outcome.pr.merged` | bool | Whether PR was merged |
| `ghaw.outcome.pr.review_count` | int | Number of reviews submitted |
| `ghaw.outcome.pr.additions` | int | Lines added |
| `ghaw.outcome.pr.deletions` | int | Lines deleted |

---

## 2. `create_issue`

**Question:** Was the issue resolved or dismissed?

**API:** `GET /repos/{owner}/{repo}/issues/{number}`

**Evaluation:**

| Condition | Outcome |
|-----------|---------|
| `state == "closed"` and `state_reason == "completed"` | `accepted` |
| `state == "closed"` and `state_reason == "not_planned"` and closed by bot | `lifecycle` |
| `state == "closed"` and `state_reason == "not_planned"` and closed by a visible non-bot actor | `rejected` |
| `state == "open"` and has non-bot comments | `pending` (engaged) |
| `state == "open"` and no non-bot comments | `ignored` |

**Bot detection:** check the close event in `GET /repos/{owner}/{repo}/issues/{number}/timeline` — if the actor is `github-actions[bot]`, classify as `lifecycle` not `rejected`.

**Extra signals:**
- `human_comments`: historical field name; non-bot comments
- Reactions on the issue body

**Additional OTel attributes:**

| Attribute | Type | Description |
|-----------|------|-------------|
| `ghaw.outcome.issue.state_reason` | string | `completed` or `not_planned` |
| `ghaw.outcome.issue.closed_by` | string | Username that closed the issue |
| `ghaw.outcome.issue.closed_by_bot` | bool | Whether a bot closed it |

---

## 3. `add_comment`

**Question:** Did anyone respond or react?

**API:** `GET /repos/{owner}/{repo}/issues/comments/{comment_id}`

**Evaluation:**

| Condition | Outcome |
|-----------|---------|
| Comment has replies (subsequent comments referencing it) or reactions > 0 | `accepted` |
| Comment exists, no replies, no reactions | `ignored` |
| Comment was deleted (404) | `rejected` |
| Comment was minimized | `rejected` |

**Extra signals:**
- Reaction count and types
- Reply count (comments posted after this one on the same issue/PR)

**Additional OTel attributes:**

| Attribute | Type | Description |
|-----------|------|-------------|
| `ghaw.outcome.comment.reactions` | int | Total reaction count |
| `ghaw.outcome.comment.replies` | int | Subsequent comments on same thread |
| `ghaw.outcome.comment.minimized` | bool | Whether the comment was hidden |

---

## 4. `add_labels`

**Question:** Did the labels stick?

**API:** `GET /repos/{owner}/{repo}/issues/{number}/labels`

**Evaluation:**

| Condition | Outcome |
|-----------|---------|
| All added labels still present | `accepted` |
| Some labels removed | `rejected` (partial) |
| All labels removed | `rejected` |

**Additional OTel attributes:**

| Attribute | Type | Description |
|-----------|------|-------------|
| `ghaw.outcome.labels.added` | int | Labels the workflow added |
| `ghaw.outcome.labels.retained` | int | Labels still present |
| `ghaw.outcome.labels.removed` | int | Labels that were removed |

**API failure safeguards (`add_labels`):**

1. If `GET /repos/{owner}/{repo}/issues/{number}/labels` returns `404`, the evaluator **MUST** classify as `rejected` because the authoritative labeling target is no longer reachable.
2. If the API returns `5xx`, timeout, or transport failure, the evaluator **MUST** classify as `pending`, record retry metadata, and retry without emitting a terminal outcome.
3. If the API returns rate-limit responses (`403` exhaustion or `429`), the evaluator **MUST** classify as `pending` and reschedule evaluation using the reset window.
4. While any transient API failure condition exists, evaluators **MUST NOT** emit `accepted` or `rejected` for label stickiness.

---

## 5. `add_reviewer`

**Question:** Did the reviewer actually review?

**API:** `GET /repos/{owner}/{repo}/pulls/{number}/reviews`

**Evaluation:**

| Condition | Outcome |
|-----------|---------|
| At least one review submitted by the assigned reviewer | `accepted` |
| No review from assigned reviewer but PR still open | `pending` |
| No review and PR closed/merged | `ignored` |

**Additional OTel attributes:**

| Attribute | Type | Description |
|-----------|------|-------------|
| `ghaw.outcome.reviewer.reviewed` | bool | Whether the assigned reviewer submitted a review |
| `ghaw.outcome.reviewer.review_state` | string | `approved`, `changes_requested`, `commented` |

---

## 6. `update_issue`

**Question:** Did the edit stick?

**API:** `GET /repos/{owner}/{repo}/issues/{number}`

**Evaluation:**

| Condition | Outcome |
|-----------|---------|
| Title/body unchanged since workflow edit (or only bot edits after) | `accepted` |
| Title/body changed by a visible non-bot actor after workflow edit | `rejected` |

**Detection:** compare `updated_at` with the workflow's edit timestamp. If `updated_at` is close to the workflow timestamp and no visible non-bot events follow, the edit stuck.

---

## 7. `update_pull_request`

**Question:** Did the edit stick?

Same logic as `update_issue` but on a PR object.

**API:** `GET /repos/{owner}/{repo}/pulls/{number}`

---

## 8. `close_issue`

**Question:** Did it stay closed?

**API:** `GET /repos/{owner}/{repo}/issues/{number}`

**Evaluation order:** first check the current issue state. If the issue is currently open (meaning it was reopened after a prior close), classify as `rejected` regardless of prior close actor. Only currently closed issues use actor-based classification below.

**Evaluation:**

| Condition | Outcome |
|-----------|---------|
| Issue still closed and close actor is `github-actions[bot]` or configured lifecycle bot | `lifecycle_close` |
| Issue still closed and close actor is a non-lifecycle actor (visible non-bot user or non-lifecycle GitHub App/integration) | `rejected` |
| Issue reopened | `rejected` |

---

## 9. `close_pull_request`

**Question:** Did it stay closed?

**API:** `GET /repos/{owner}/{repo}/pulls/{number}`

**Evaluation order:** first check the current PR state. If the PR is currently open (meaning it was reopened after a prior close), classify as `rejected` regardless of prior close actor. Only currently closed PRs use actor-based classification below.

**Evaluation:**

| Condition | Outcome |
|-----------|---------|
| PR still closed and close actor is `github-actions[bot]` or configured lifecycle bot | `lifecycle_close` |
| PR still closed and close actor is a non-lifecycle actor (visible non-bot user or non-lifecycle GitHub App/integration) | `rejected` |
| PR reopened | `rejected` |

---

## 10. `close_discussion`

**Question:** Did it stay closed?

**API:** GraphQL `repository.discussion(number:)` → `closed`

**Evaluation:**

| Condition | Outcome |
|-----------|---------|
| Discussion still closed | `accepted` |
| Discussion reopened | `rejected` |

---

## 11. `create_discussion`

**Question:** Did anyone engage?

**API:** GraphQL `repository.discussion(number:)` → `comments.totalCount`, `answer`

**Evaluation:**

| Condition | Outcome |
|-----------|---------|
| Has replies or is marked as answered | `accepted` |
| No replies, not answered | `ignored` |

**Additional OTel attributes:**

| Attribute | Type | Description |
|-----------|------|-------------|
| `ghaw.outcome.discussion.replies` | int | Reply count |
| `ghaw.outcome.discussion.answered` | bool | Whether marked as answered |

---

## 12. `update_discussion`

**Question:** Did the edit stick?

Same logic as `update_issue` but via GraphQL on the discussion body.

---

## 13. `create_pull_request_review_comment`

**Question:** Was the thread resolved or engaged?

**API:** GraphQL `pullRequest.reviewThreads` filtered to the comment's thread

**Evaluation:**

| Condition | Outcome |
|-----------|---------|
| Thread resolved | `accepted` |
| Thread has replies | `accepted` |
| Thread not resolved, no replies | `ignored` |

---

## 14. `submit_pull_request_review`

**Question:** Was the feedback addressed?

**API:** `GET /repos/{owner}/{repo}/pulls/{number}/commits` (check for commits after review)

**Evaluation:**

| Condition | Outcome |
|-----------|---------|
| New commits pushed after review submission | `accepted` |
| Review dismissed | `rejected` |
| No new commits and PR still open | `pending` |
| PR merged (regardless of follow-up commits) | `accepted` |

---

## 15. `reply_to_pull_request_review_comment`

**Question:** Was the conversation advanced?

**API:** GraphQL `pullRequest.reviewThreads` → check thread state

**Evaluation:**

| Condition | Outcome |
|-----------|---------|
| Thread resolved after reply | `accepted` |
| Further replies from others | `accepted` |
| No further activity | `ignored` |

---

## 16. `resolve_pull_request_review_thread`

**Question:** Did it stay resolved?

**API:** GraphQL `pullRequest.reviewThreads` → `isResolved`

**Evaluation:**

| Condition | Outcome |
|-----------|---------|
| Thread still resolved | `accepted` |
| Thread unresolved | `rejected` |

---

## 17. `push_to_pull_request_branch`

**Question:** Was the code accepted?

**API:** `GET /repos/{owner}/{repo}/pulls/{number}`

**Evaluation:**

| Condition | Outcome |
|-----------|---------|
| PR merged | `accepted` |
| PR closed without merge | `rejected` |
| PR still open | `pending` |
| PR merged but pushed commits were reverted | `rejected` |

---

## 18. `mark_pull_request_as_ready_for_review`

**Question:** Did someone review it?

**API:** `GET /repos/{owner}/{repo}/pulls/{number}/reviews`

**Evaluation:**

| Condition | Outcome |
|-----------|---------|
| At least one review submitted after marking | `accepted` |
| No reviews but PR still open | `pending` |
| PR merged or closed with no reviews | `ignored` |

---

## 19. `assign_to_agent`

**Question:** Did the agent produce a result?

**API:**
1. `GET /repos/{owner}/{repo}/issues/{number}` → check state
2. Search for PRs from agent: `GET /repos/{owner}/{repo}/issues/{number}/timeline` → find linked PRs from `copilot-swe-agent`

**Evaluation:**

| Condition | Outcome |
|-----------|---------|
| Agent PR created and merged | `accepted` |
| Agent PR created but closed without merge | `rejected` |
| Agent PR created and still open | `pending` |
| No agent PR created | `ignored` |
| Issue resolved without agent PR | `accepted` (resolved by other means) |

**Additional OTel attributes:**

| Attribute | Type | Description |
|-----------|------|-------------|
| `ghaw.outcome.agent.pr_number` | int | PR number created by agent |
| `ghaw.outcome.agent.pr_merged` | bool | Whether agent's PR was merged |

---

## 20. `dispatch_workflow`

**Question:** Did the dispatched workflow succeed?

**API:** `GET /repos/{owner}/{repo}/actions/runs` filtered by workflow and time window

**Evaluation:**

| Condition | Outcome |
|-----------|---------|
| Dispatched run completed with `conclusion == "success"` | `accepted` |
| Dispatched run completed with `conclusion == "failure"` | `rejected` |
| Dispatched run not found or still running | `pending` |

---

## 21. `autofix_code_scanning_alert`

**Question:** Was the fix accepted?

**API:**
1. Check alert state: `GET /repos/{owner}/{repo}/code-scanning/alerts/{alert_number}`
2. Check linked PR if any

**Evaluation:**

| Condition | Outcome |
|-----------|---------|
| Alert state changed to `fixed` | `accepted` |
| Alert dismissed | `rejected` |
| Linked PR merged | `accepted` |
| Linked PR closed | `rejected` |
| Alert still open | `pending` |

---

## 22. `create_code_scanning_alert`

**Question:** Was the alert triaged?

**API:** `GET /repos/{owner}/{repo}/code-scanning/alerts/{alert_number}`

**Evaluation:**

| Condition | Outcome |
|-----------|---------|
| Alert state `fixed` | `accepted` |
| Alert state `dismissed` with reason | `accepted` (triaged) |
| Alert still `open` | `pending` |

---

## 23. `link_sub_issue`

**Question:** Did the link stick?

**API:** GraphQL `issue.subIssues`

**Evaluation:**

| Condition | Outcome |
|-----------|---------|
| Sub-issue link still present | `accepted` |
| Link removed | `rejected` |

---

## 24. `hide_comment`

**Question:** Did it stay hidden?

**API:** GraphQL `node(id:)` → `isMinimized`

**Evaluation:**

| Condition | Outcome |
|-----------|---------|
| Comment still minimized | `accepted` |
| Comment un-minimized | `rejected` |

---

## 25. `assign_milestone`

**Question:** Did the milestone stick?

**API:** `GET /repos/{owner}/{repo}/issues/{number}`

**Evaluation:**

| Condition | Outcome |
|-----------|---------|
| Milestone still assigned | `accepted` |
| Milestone removed or changed | `rejected` |

---

## 26. `update_project`

**Question:** Did the field value stick?

**API:** GraphQL `projectV2` → field value query

**Evaluation:**

| Condition | Outcome |
|-----------|---------|
| Field value unchanged | `accepted` |
| Field value changed by someone else | `rejected` |

---

## 27. `update_release`

**Question:** Did the edit stick?

**API:** `GET /repos/{owner}/{repo}/releases/{release_id}`

**Evaluation:**

| Condition | Outcome |
|-----------|---------|
| Release body/name unchanged since workflow edit | `accepted` |
| Release body/name changed by someone else | `rejected` |

---

## 28. `noop`

No outcome to evaluate. Skip.

## 29. `missing_tool`

No outcome to evaluate. Skip.

---

## Derived Metrics

From the outcome evaluations above, compute:

| Metric | Formula | Description | Go aggregation owner | OTel emission owner |
|--------|---------|-------------|----------------------|---------------------|
| `acceptance_rate` | accepted / (accepted + rejected) | How often actions are kept | `pkg/cli/outcome_eval.go` (`ComputeOutcomeSummary`) | `actions/setup/js/emit_outcome_spans.cjs` (`buildSummaryAttributes`) |
| `waste_rate` | rejected / total | How often actions are undone | `pkg/cli/outcome_eval.go` (`ComputeOutcomeSummary`) | `actions/setup/js/emit_outcome_spans.cjs` (`buildSummaryAttributes`) |
| `ignore_rate` | ignored / total | How often actions get no response | `pkg/cli/outcome_eval.go` (`ComputeOutcomeSummary`) | `actions/setup/js/emit_outcome_spans.cjs` (`buildSummaryAttributes`) |
| `zero_touch_rate` | zero_touch / accepted | How often accepted actions need no actor-visible non-bot edits | `pkg/cli/outcome_eval.go` (`ComputeOutcomeSummary`) | `actions/setup/js/emit_outcome_spans.cjs` (`buildSummaryAttributes`) |
| `time_to_outcome` | median(time_to_outcome_hours) | How fast outcomes resolve | `pkg/cli/outcome_eval.go` (`ComputeOutcomeSummary`) | `actions/setup/js/emit_outcome_spans.cjs` (`buildSummaryAttributes`) |
| `cost_per_accepted_outcome` | total_run_cost / accepted_count | Efficiency metric | `pkg/cli/outcome_eval.go` (`ComputeOutcomeSummary`) | `actions/setup/js/emit_outcome_spans.cjs` (`buildSummaryAttributes`) |

## Implementation Priority

Start with the 5 highest-value, lowest-effort types:

1. `create_pull_request` — cleanest signal, most valuable
2. `create_issue` — common, needs bot-aware close detection
3. `add_comment` — very common, engagement signal
4. `add_labels` — simple binary check
5. `assign_to_agent` — important for delegation workflows

These cover the majority of safe output usage. Add the rest incrementally.

---

## Conformance

### Conformance Test Table

The table below specifies one conformance test row per safe-output type. Each row defines the expected OTel attribute value emitted by a correct evaluator, the pass condition (what must be true for `accepted`), and the fail condition (what signals `rejected`). Implementations **MUST** satisfy the pass condition and **MUST** not emit `accepted` when the fail condition is observed.

| Output type | Expected `ghaw.outcome.type` OTel attribute | Pass condition | Fail condition |
|---|---|---|---|
| `create_pull_request` | `create_pull_request` | PR exists in open or merged state; was not closed-as-not-planned or reverted within the evaluation window | PR closed-as-not-planned, reverted, or deleted within the evaluation window |
| `create_issue` | `create_issue` | Issue exists in open state, or was closed by a visible non-bot action (not bot policy) within the evaluation window | Issue closed-as-not-planned by a visible non-bot actor within the evaluation window, or deleted |
| `add_comment` | `add_comment` | Comment exists on the target object at evaluation time | Comment was deleted or hidden by a visible non-bot actor within the evaluation window |
| `add_labels` | `add_labels` | At least one of the bot-applied labels is still present on the target object at evaluation time | All bot-applied labels were removed by a visible non-bot actor within the evaluation window |
| `add_reviewer` | `add_reviewer` | Requested reviewer is still listed as a requested reviewer, or has already submitted a review | Reviewer request was removed by a visible non-bot actor before any review was submitted |
| `update_issue` | `update_issue` | Updated field(s) (title, body, assignee) match the values the bot submitted at evaluation time | Updated field(s) were reverted to pre-bot values by a visible non-bot actor within the evaluation window |
| `update_pull_request` | `update_pull_request` | Updated field(s) (title, body, base branch) match the values the bot submitted at evaluation time | Updated field(s) were reverted to pre-bot values by a visible non-bot actor within the evaluation window |
| `close_issue` | `close_issue` | Issue remains closed at evaluation time | Issue was reopened by a visible non-bot actor within the evaluation window |
| `close_pull_request` | `close_pull_request` | PR remains closed (not merged) at evaluation time | PR was reopened or merged after the bot closed it within the evaluation window |
| `close_discussion` | `close_discussion` | Discussion remains closed at evaluation time | Discussion was reopened by a visible non-bot actor within the evaluation window |
| `create_discussion` | `create_discussion` | Discussion exists and has not been deleted or locked within the evaluation window | Discussion was deleted or permanently locked (preventing any responses) within the evaluation window |
| `update_discussion` | `update_discussion` | Updated field(s) (title, body, category) match the values the bot submitted at evaluation time | Updated field(s) were reverted to pre-bot values by a visible non-bot actor within the evaluation window |
| `create_pull_request_review_comment` | `create_pull_request_review_comment` | Review comment exists on the PR diff at evaluation time | Review comment was deleted by a visible non-bot actor within the evaluation window |
| `submit_pull_request_review` | `submit_pull_request_review` | PR review record exists with the submitted state (APPROVED, CHANGES_REQUESTED, COMMENT) at evaluation time | Review was dismissed by a visible non-bot actor within the evaluation window |
| `reply_to_pull_request_review_comment` | `reply_to_pull_request_review_comment` | Reply comment exists in the review thread at evaluation time | Reply comment was deleted by a visible non-bot actor within the evaluation window |
| `resolve_pull_request_review_thread` | `resolve_pull_request_review_thread` | Review thread remains resolved at evaluation time | Thread was re-opened (un-resolved) by a visible non-bot actor within the evaluation window |
| `push_to_pull_request_branch` | `push_to_pull_request_branch` | The pushed commit SHA is still present in the PR branch history at evaluation time | The commit was force-pushed out of the branch history by a visible non-bot actor within the evaluation window |
| `mark_pull_request_as_ready_for_review` | `mark_pull_request_as_ready_for_review` | PR is no longer in draft state at evaluation time | PR was converted back to draft by a visible non-bot actor within the evaluation window |
| `assign_to_agent` | `assign_to_agent` | Assignment record exists on the target issue/PR at evaluation time | Assignment was removed by a visible non-bot actor before the assigned agent acted on it |
| `dispatch_workflow` | `dispatch_workflow` | The dispatched workflow run exists and reached a terminal state (success or failure) within the evaluation window | The dispatched workflow run was cancelled before reaching a terminal state; or no corresponding run record is found |
| `autofix_code_scanning_alert` | `autofix_code_scanning_alert` | Code scanning alert is in a fixed or dismissed state at evaluation time | Alert was re-opened or the fix commit was reverted within the evaluation window |
| `create_code_scanning_alert` | `create_code_scanning_alert` | Alert record exists in the repository's code scanning results at evaluation time | Alert was immediately dismissed (within the evaluation window) with no investigation action |
| `link_sub_issue` | `link_sub_issue` | Sub-issue link exists on the parent issue at evaluation time | Sub-issue link was removed by a visible non-bot actor within the evaluation window |
| `hide_comment` | `hide_comment` | Comment is minimized (hidden) at evaluation time | Comment was un-hidden by a visible non-bot actor within the evaluation window |
| `assign_milestone` | `assign_milestone` | Milestone assignment is present on the target issue/PR at evaluation time | Milestone assignment was removed by a visible non-bot actor within the evaluation window |
| `update_project` | `update_project` | Project item field(s) match the values the bot submitted at evaluation time | Project item field(s) were reverted to pre-bot values by a visible non-bot actor within the evaluation window |
| `update_release` | `update_release` | Release field(s) (name, body, tag, draft status) match the values the bot submitted at evaluation time | Release field(s) were reverted by a visible non-bot actor, or the release was deleted within the evaluation window |
| `noop` | `noop` | Evaluation is skipped; no outcome is computed | N/A — `noop` always results in `ignored` |
| `missing_tool` | `missing_tool` | Evaluation is skipped; no outcome is computed | N/A — `missing_tool` always results in `ignored` |

### OTel Backend Unavailability

When the OTLP exporter is unavailable (e.g., endpoint unreachable, network timeout, authentication failure) during outcome evaluation, the following safeguards **MUST** apply:

1. **Graceful degradation**: Outcome evaluation workers **MUST** complete their classification logic (determining `accepted`, `rejected`, `ignored`, etc.) regardless of OTLP exporter availability. The computed outcome **MUST** be persisted to a local audit fallback log (e.g., a NDJSON file at a known path such as `/tmp/gh-aw/outcome-audit.ndjson`) before any attempt to export to OTLP. If the OTLP export fails, the local audit log entry **MUST** still be written and **MUST NOT** be discarded. This ensures the outcome is recoverable even when the telemetry backend is down.

2. **Audit fallback and retry**: When OTLP export fails, the evaluation worker **SHOULD** schedule a retry using an exponential back-off strategy (initial delay: 5 seconds; maximum delay: 5 minutes; maximum attempts: 5). If all retries are exhausted without a successful export, the worker **MUST** record the export failure in the local audit log with a `export_failed: true` flag and the final error reason. A downstream reconciliation process **SHOULD** periodically sweep the local audit log and re-attempt export for any entries marked `export_failed: true`.

### Conformance Safeguard Coverage Requirements

Conformance suites **MUST** include explicit safeguard coverage classes in addition to happy-path outcome checks:

1. **Class A (state success/failure):** validates standard accepted/rejected/ignored/pending transitions from authoritative object state.
2. **Class B (human override/lifecycle):** validates human edits, deletions, reopen events, and lifecycle outcomes where applicable.
3. **Class C (API degradation):** validates `404`, `5xx`, timeout, and rate-limit behaviors, including retry metadata and non-terminal handling.

Every safe-output type **MUST** have at least one Class A test. Types that query GitHub APIs for evaluation **MUST** also include at least one Class C test case.
