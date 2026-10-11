# ADR-67743: Normalize Claude `local_bash` Background Tasks as Tool Execution Updates

**Date**: 2026-10-11
**Status**: Draft
**Deciders**: pelikhan (PR author), gh-aw maintainers

---

### Context

The Claude session adapter (`actions/setup/js/claude_session.cjs`) translates Claude-native stdio records into the engine-neutral agent-session event schema. Claude emits `system` records with `subtype: "task_started"` / `task_notification`, which the adapter previously routed through `DYNAMIC_WORKFLOW_EVENT_TYPES`, so backgrounded shell tasks (`task_type: "local_bash"`) were reported as dynamic workflow lifecycle events rather than as activity belonging to an existing tool call. Those records carry only observational facts — a task id, an optional `tool_use_id`, a status and an optional `summary` — and never a tool result, exit code or usage block, so any mapping that implies completion would fabricate data. Saved sessions from earlier runs already contain `dynamicWorkflows.*` Bash wrappers that must keep working, and unknown native record types were previously discarded outright, losing forward compatibility with newer Claude CLI versions. Live evidence from three existing `agent` artifacts (Claude 2.1.285 and 2.1.288) confirms both the mis-typed Bash lifecycle records and the volume of discarded wrappers.

### Decision

We will map Claude `local_bash` task starts and notifications to the canonical `tool.execution_update` event, correlating task identity within a session/child scope via `bashTaskKey(source, task_id)` so a `task_notification` without `task_type` can be resolved back to its originating `task_started`. Emitted updates carry `taskId`, `taskType: "local_bash"`, the resolved `toolCallId`, `partial: true`, the observed `sourceType`/status, errors and `summary` output, and deliberately omit success flags, exit codes and usage. Saved `dynamicWorkflows.*` Bash wrappers are migrated in `normalizeClaudeObservations` with provenance intact, and previously discarded unknown native record types are now preserved as opaque Claude extension events. The primary driver is fidelity: a background shell task is an in-progress observation of a tool call, and the schema must not invent terminal state the engine never observed.

### Alternatives Considered

#### Alternative 1: Keep `local_bash` tasks on the dynamic-workflow event path

Continue emitting `dynamicWorkflows.task_started` / `task_notification` for Bash tasks and let downstream consumers special-case `task_type`. This was a close call because it requires no adapter change and preserves existing saved-session shapes verbatim. It was rejected because it overloads the dynamic-workflow vocabulary with something that is not a workflow, forcing every consumer (log rendering, audit, metrics) to re-implement the same discrimination, and it leaves the task disconnected from the tool call it actually belongs to.

#### Alternative 2: Synthesize paired `tool.execution_start` / `tool.execution_end` events

Model each background task as a full tool lifecycle, inferring completion and success from the `task_notification` status. This would give consumers a tidy start/end pair. It was rejected because Claude never reports a result, exit code or usage for these records, so success, duration and output would be fabricated; it would also double-count tool executions that already have real start/end events under the same `tool_use_id`.

#### Alternative 3: Drop `local_bash` lifecycle records entirely

Filter them out as noise, relying on the real tool call events. Rejected because the `summary` and error fields are the only visibility into long-running backgrounded shells, and silently discarding them repeats the forward-compatibility failure this PR is also fixing for unknown native record types.

### Consequences

#### Positive

- Background shell activity is attributed to the correct tool call via `toolCallId`, so consumers see progress on the existing execution rather than a phantom workflow.
- Observed-only semantics (`partial: true`, no exit code/usage) keep the session schema honest; no downstream consumer can mistake an observation for a completion.
- Unknown Claude record types are preserved as opaque extension events, so newer CLI releases degrade gracefully instead of losing data.
- Saved sessions containing legacy `dynamicWorkflows.*` Bash wrappers are migrated with provenance, so historical artifacts render under the new model.

#### Negative

- Consumers that previously matched on `dynamicWorkflows.task_started` for Bash tasks must be updated; the change is a behavioural break in the emitted event stream.
- Scope-based correlation (`sessionScopeKey` + `task_id`) is heuristic: a `task_notification` whose `task_started` was never observed, or whose scope conflicts, falls back to an un-correlated update with no `toolCallId`.
- The adapter grows an additional pre-pass (`collectClaudeBashTasks`) over all records plus a migration branch, increasing the surface area of a file that is already dense with engine-specific special cases.
- Preserving unknown native records increases session payload size for traces that contain many opaque wrappers.

#### Neutral

- Changes are confined to the Claude adapter and its tests; workflow/subagent mappings, orphan notifications, explicit non-Bash task types, conversation, tool results and usage compatibility are unchanged.
- Depends on a shared commit (`36774f94cb`, cherry-picked as `a5198048f7`) that retains `taskId`/`taskType`/`error` on unified tool updates; shared schema, spec and version-constant work remains coordinator-owned.
- Coverage for failed/stopped/future statuses, saved-wrapper migration and scope conflicts is synthetic; only the Bash lifecycle normalization itself is backed by live artifact evidence.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
