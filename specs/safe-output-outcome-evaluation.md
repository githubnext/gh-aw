---
title: Safe Output Outcome Evaluation Specification
version: 1.0.0
status: Working Draft
date: 2026-05-15
last_updated: 2026-05-15
---

# Safe Output Outcome Evaluation Specification

Every safe output type has a measurable outcome. This spec defines the exact evaluation logic for each type: what to check, how to classify the outcome, and what OTel attributes to emit.

## Principles

1. **Same as a human would check.** If a human did this action, how would you know it was good?
2. **Level 2 only.** We check whether the action stuck, not whether it caused downstream effects.
3. **Bot-aware.** Distinguish bot-initiated closes/edits from human ones.
4. **Time-bounded.** Check outcomes after a configurable delay (default: 48 hours).

## Norms

The key words **MUST**, **MUST NOT**, and **SHOULD** in this document are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

1. Outcome evaluation workers **MUST** treat GitHub API `404` responses as terminal for deleted or inaccessible objects and classify according to object semantics (for example, a deleted issue/PR should be `rejected`, while a transient target with no persistent evaluable object should be `ignored`).
2. Outcome evaluation workers **MUST** treat GitHub API `5xx` responses as transient infrastructure failures and return `pending` for that check cycle while recording retry metadata (`status_code`, `retry_after`, `attempt`).
3. Outcome evaluation workers **MUST** treat GitHub API rate-limit responses (`403` with limit exhaustion or `429`) as transient and **SHOULD** reschedule evaluation using the reset window before emitting final outcomes.
4. Outcome evaluation workers **MUST NOT** emit `accepted` or `rejected` when API failures prevent verification of the authoritative object state.

## Outcome Categories

Every evaluation produces one of these outcomes:

| Outcome | Meaning |
|---------|---------|
| `accepted` | The action was kept, merged, resolved, or engaged with |
| `rejected` | The action was undone, closed-as-not-planned, removed, or reverted |
| `ignored` | No human interaction within the evaluation window |
| `pending` | The object has not reached a terminal state yet |
| `lifecycle` | Closed/removed by the workflow itself (e.g., `close-older-issues`) — not a rejection |
| `lifecycle_close` | Closed by lifecycle/noop bot policy and not reopened by humans |

## Common OTel Attributes

Every outcome span carries these attributes:

| Attribute | Type | Description |
|-----------|------|-------------|
| `ghaw.outcome.type` | string | Safe output type (e.g., `create_pull_request`) |
| `ghaw.outcome.result` | string | One of: `accepted`, `rejected`, `ignored`, `pending`, `lifecycle`, `lifecycle_close` |
| `ghaw.outcome.object_url` | string | GitHub URL of the affected object |
| `ghaw.outcome.object_number` | int | Issue/PR/discussion number |
| `ghaw.outcome.repo` | string | `owner/repo` |
| `ghaw.outcome.source_run_id` | string | Workflow run that created this output |
| `ghaw.outcome.source_trace_id` | string | Original OTLP trace ID |
| `ghaw.outcome.created_at` | string | When the safe output was executed |
| `ghaw.outcome.checked_at` | string | When this evaluation ran |
| `ghaw.outcome.time_to_outcome_hours` | float | Hours from creation to terminal state |
| `ghaw.outcome.human_comments` | int | Human (non-bot) comments on the object |
| `ghaw.outcome.human_edits` | int | Human edits before acceptance (0 = zero-touch) |
| `ghaw.outcome.zero_touch` | bool | Accepted with no human modifications |

## Implementation

The following implementation areas are responsible for evaluation data capture, outcome classification plumbing, and runtime event artifacts:

| Output type | Go implementation areas | JS/runtime implementation areas |
|-------------|--------------------------|---------------------------------|
| `create_pull_request` | `pkg/workflow/safe_outputs_config.go`, `pkg/workflow/safe_outputs_parser.go`, `pkg/workflow/compiler_safe_outputs.go` | `actions/setup/js/safe_outputs_config.cjs`, `actions/setup/js/safe_outputs_handlers.cjs` |
| `create_issue` | `pkg/workflow/safe_outputs_config.go`, `pkg/workflow/safe_outputs_parser.go`, `pkg/workflow/compiler_safe_outputs.go` | `actions/setup/js/safe_outputs_config.cjs`, `actions/setup/js/safe_outputs_handlers.cjs` |
| `add_comment` | `pkg/workflow/safe_outputs_config.go`, `pkg/workflow/safe_outputs_dispatch.go`, `pkg/workflow/compiler_safe_outputs.go` | `actions/setup/js/safe_outputs_handlers.cjs`, `actions/setup/js/safe_outputs_action_outputs.cjs` |
| `add_labels` | `pkg/workflow/safe_outputs_allowed_labels_validation.go`, `pkg/workflow/safe_outputs_config.go` | `actions/setup/js/safe_outputs_handlers.cjs`, `actions/setup/js/safe_outputs_config.cjs` |
| `assign_to_agent` | `pkg/workflow/safe_outputs_config.go`, `pkg/workflow/safe_outputs_dispatch.go`, `pkg/workflow/compiler_safe_outputs.go` | `actions/setup/js/safe_outputs_handlers.cjs`, `actions/setup/js/safe_outputs_bootstrap.cjs` |
| `close_issue` / `close_pull_request` | `pkg/workflow/safe_outputs_config.go`, `pkg/workflow/safe_outputs_dispatch.go`, `pkg/workflow/compiler_safe_outputs.go` | `actions/setup/js/safe_outputs_handlers.cjs`, `actions/setup/js/safe_outputs_action_outputs.cjs` |

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
- `human_edits`: count commits pushed by users other than the PR author after creation
- `human_comments`: count non-bot comments on the PR
- `zero_touch`: `accepted` and `human_edits == 0`
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
| `state == "closed"` and `state_reason == "not_planned"` and closed by human | `rejected` |
| `state == "open"` and has human comments | `pending` (engaged) |
| `state == "open"` and no human comments | `ignored` |

**Bot detection:** check the close event in `GET /repos/{owner}/{repo}/issues/{number}/timeline` — if the actor is `github-actions[bot]`, classify as `lifecycle` not `rejected`.

**Extra signals:**
- `human_comments`: non-bot comments
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
| Title/body changed by a human after workflow edit | `rejected` |

**Detection:** compare `updated_at` with the workflow's edit timestamp. If `updated_at` is close to the workflow timestamp and no human events follow, the edit stuck.

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
| Issue still closed and close actor is a non-lifecycle actor (human or non-lifecycle GitHub App/integration) | `rejected` |
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
| PR still closed and close actor is a non-lifecycle actor (human or non-lifecycle GitHub App/integration) | `rejected` |
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

| Metric | Formula | Description |
|--------|---------|-------------|
| `acceptance_rate` | accepted / (accepted + rejected) | How often actions are kept |
| `waste_rate` | rejected / total | How often actions are undone |
| `ignore_rate` | ignored / total | How often actions get no response |
| `zero_touch_rate` | zero_touch / accepted | How often accepted actions need no human edits |
| `time_to_outcome` | median(time_to_outcome_hours) | How fast outcomes resolve |
| `cost_per_accepted_outcome` | total_run_cost / accepted_count | Efficiency metric |

## Implementation Priority

Start with the 5 highest-value, lowest-effort types:

1. `create_pull_request` — cleanest signal, most valuable
2. `create_issue` — common, needs bot-aware close detection
3. `add_comment` — very common, engagement signal
4. `add_labels` — simple binary check
5. `assign_to_agent` — important for delegation workflows

These cover the majority of safe output usage. Add the rest incrementally.
