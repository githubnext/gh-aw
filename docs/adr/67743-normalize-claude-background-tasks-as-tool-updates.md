# ADR-67743: Normalize Claude `local_bash` Background Tasks as Tool Execution Updates

**Date**: 2026-10-11
**Status**: Draft
**Deciders**: pelikhan (PR author), gh-aw maintainers

---

### Context

The Claude session adapter (`actions/setup/js/claude_session.cjs`) translates Claude-native stdio records into the engine-neutral agent-session event schema. Claude emits `system` records with `subtype: "task_started"` / `task_notification`, which the adapter previously routed through `DYNAMIC_WORKFLOW_EVENT_TYPES`. Backgrounded shell tasks (`task_type: "local_bash"`) were therefore reported as dynamic workflow lifecycle events rather than as observations belonging to a tool call. A notification can omit `task_type`; identifying it requires correlation with a start record in the same session/child scope, without reclassifying unrelated workflow or subagent activity.

Existing downloaded `agent-stdio.log` artifacts from runs [36812703027](https://github.com/github/gh-aw/actions/runs/36812703027) and [36762044297](https://github.com/github/gh-aw/actions/runs/36762044297), both using Claude 2.1.285, contain eight and four Bash lifecycle records respectively. Their starts identify `local_bash`; their notifications report `status: "completed"` without `task_type`. These observations do not establish a tool result, exit code or accounting data. Run [38098153730](https://github.com/github/gh-aw/actions/runs/38098153730), using Claude 2.1.288, provides saved-session compatibility evidence but contains no Bash lifecycle records. No fresh agent execution or workflow run is required for this decision, and no newer-version live behavior is claimed.

Saved sessions may already contain `dynamicWorkflows.*` Bash wrappers. Migration must retain provenance without replaying conversation or tool results. Separately, previously discarded unknown native types need an opaque representation; their names must not impersonate canonical events.

### Decision

Map Claude `local_bash` task starts and notifications to canonical `tool.execution_update` observations. `collectClaudeBashTasks` indexes starts by `bashTaskKey(source, task_id)`, combining session/child scope and task identity, so a notification without `task_type` can use a matching start even when record order differs. An explicit non-Bash type is not overridden, and a notification without either explicit Bash identity or a matching start stays on the existing mapping path.

Updates carry `taskId`, `taskType: "local_bash"`, an available `toolCallId`, `partial: true`, the observed subtype/status/error fields and string `summary` as `output`. Available tool metadata is retained; missing metadata is not invented. A reported task status, including `completed`, is not synthesized into a canonical tool completion, success flag, exit code or usage. `partial: true` marks an observational event, not a claim that the background process is still running.

Migrate saved `dynamicWorkflows.*` Bash wrappers in `normalizeClaudeObservations`, preserving provenance and avoiding duplicate tool results. Preserve unknown native record types as opaque `claude.*` extension events, including dotted names such as `session.result` and `tool.execution_complete`; only names already prefixed with `claude.` are left unchanged. The primary driver is fidelity: preserve what Claude observed without assigning unsupported terminal or workflow semantics.

### Alternatives Considered

#### Alternative 1: Keep `local_bash` tasks on the dynamic-workflow event path

Continue emitting `dynamicWorkflows.task_started` / `task_notification` for Bash tasks and let downstream consumers special-case `task_type`. This preserves existing saved-session shapes and avoids migration, but overloads workflow vocabulary with shell activity and leaves correlation and classification to consumers. Adapter-level normalization provides a single interpretation while retaining the original observed fields.

#### Alternative 2: Synthesize paired `tool.execution_start` / `tool.execution_complete` events

Model each background task as a full tool lifecycle, inferring completion and success from the `task_notification` status. This would give consumers a start/end pair, but the inspected lifecycle records do not establish a tool result or exit code. Synthesizing completion could also double-count executions already represented by real tool events under the same `tool_use_id`. An update can retain the reported summary without inventing a result.

#### Alternative 3: Drop `local_bash` lifecycle records entirely

Filter them out as noise, relying on the real tool call events. This reduces payload size but loses task identity, status, summaries and diagnostics present in the trace. Preserving observations is preferable to discarding potentially useful audit evidence.

### Consequences

#### Positive

- Background shell activity can be attributed to an existing tool call via `toolCallId`, rather than represented as a workflow.
- Observational semantics (`partial: true`, no synthesized completion or accounting) distinguish task reports from canonical tool results.
- Unknown Claude record types remain inspectable as namespaced extension events without acquiring canonical semantics from their names.
- Saved sessions containing legacy `dynamicWorkflows.*` Bash wrappers are migrated with provenance, so historical artifacts render under the new model.

#### Negative

- Consumers that previously matched on `dynamicWorkflows.task_started` for Bash tasks must accommodate the new event type.
- Correlation depends on matching scope and task identity. Untyped orphan notifications retain their existing mapping; explicitly typed Bash records can become updates even without a correlated tool call. A reused task id within the same scope is not independently disambiguated.
- The adapter grows an additional pre-pass (`collectClaudeBashTasks`) over all records plus a migration branch, increasing the surface area of a file that is already dense with engine-specific special cases.
- Preserving unknown native records increases session payload size for traces that contain many opaque wrappers.

#### Neutral

- Relative to the PR base `pelikhan-unified-session-parsers`, implementation changes are confined to the Claude adapter and its two test files, plus this ADR. Work-queue staging, dry-run compiler metadata and security-review policy changes are inherited base work, not decisions made by this PR.
- Existing workflow/subagent mappings, untyped orphan notifications and explicit non-Bash task types remain on their existing paths. Conversation, tool-result and usage compatibility remain covered by the Claude tests.
- Shared unified-session support for `taskId`/`taskType`/`error` is a prerequisite supplied by the base; this decision does not change the shared schema or runtime version pin.
- Failed/stopped/future statuses, saved-wrapper migration, scope conflicts and unknown-type behavior are synthetic regression cases, not claims about fresh live traces. Bash lifecycle fixtures in `claude_session_audit.test.cjs` have documented provenance from the existing 2.1.285 run.

---

*Status remains Draft pending maintainer acceptance; completing this record does not imply that the proposal has been accepted.*
