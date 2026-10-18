# ADR-69868: Report Sub-agent Failures Separately from the Parent Workflow Conclusion

**Date**: 2026-10-18
**Status**: Draft (PR #69868 is not yet accepted)
**Deciders**: Not recorded; pending maintainer review of PR #69868.

---

### Context

`gh aw audit` reported a successful parent workflow as "Completed in N turns with no errors" even when delegated sub-agent invocations failed. The Contribution Check run 39098581712 showed this: the Actions conclusion was `success` with `error_count: 0`, yet two delegated invocations failed. Failure counts already existed in `ProcessedRun.TokenUsage.SubagentModelRequests[].FailedCount` and were surfaced in `subagent_failed` findings. However, no aggregate metric showed them, and the success narrative implied that delegated work had completed. Parent success alone establishes neither completion of delegated work nor recovery from its failures. Audit results are cached with a versioned schema (`auditSchemaVersion`), so unchanged caches would retain the old wording. The daily CLI tools tester had a similar truthfulness problem: it reported expected coverage exclusions as incomplete work and gave a blanket all-tools-success message.

### Decision

Report observed delegated failures as a separate metric, `metrics.subagent_failure_count`, summed from `SubagentModelRequests[].FailedCount`. This counts failed invocations, not distinct agents or the mirrored `DeclaredSubagentModels` entries. A later successful retry does not remove an observed failure. Preserve the existing workflow-error meaning of `error_count` and the actual parent `conclusion`.

When this count is non-zero and a `workflow_succeeded` finding is emitted, its title is "Parent Workflow Completed Successfully" and its description states that sub-agent invocations failed. If reported workflow errors and delegated failures coexist, both the description and impact must retain both signals; neither qualification replaces the other. A high-priority recommendation tells users to verify recovery from session evidence instead of inferring it from parent success. Add the field to `audit.schema.json` and `logs-jsonl.schema.json`, show `subagent_failures=` in console output when non-zero, and raise `auditSchemaVersion` from 7 to 8 so stale cached narratives are rebuilt on next use.

Similarly, the CLI tester marks expected exclusions as "not tested (expected exclusion)". They do not count as passes or alone trigger `report_incomplete`, but blocked or unfinished required checks still trigger it. Neither a final `noop` nor a findings issue overrides incomplete required coverage or establishes full coverage.

### Alternatives Considered

#### Alternative 1: Count sub-agent failures in `error_count`

Adding delegated failures to `error_count` would surface them without a schema change. This is not chosen because it would change the workflow-error metric's meaning: the observed parent run reported zero workflow errors and concluded `success`. Consumers would lose the distinction between workflow errors and delegated failures. Separate metrics preserve that distinction without asserting whether a retry or parent fallback recovered the delegated work.

#### Alternative 2: Rely only on the existing `subagent_failed` findings

The individual findings were already emitted, so retaining them alone would avoid a new metric. This is not chosen because they do not qualify the misleading success narrative or provide an aggregate failed-invocation count for JSON/JSONL consumers and console readers. The findings remain useful detail alongside the new metric.

#### Alternative 3: Mark the overall audit as failed when any sub-agent fails

Treating every observed delegated failure as an overall failure would provide a simple failure signal. This is not chosen because overriding the conclusion would misrepresent the actual GitHub Actions result, while a separate audit failure policy would conflate an observed invocation failure with unresolved task failure. The available count establishes neither recovery nor the absence of recovery; the qualified narrative and recommendation expose the uncertainty instead.

### Consequences

#### Positive
- Parent outcome, workflow errors and delegated failures remain distinct, machine-readable signals (`conclusion`, `error_count`, `subagent_failure_count`).
- The success narrative qualifies both workflow errors and delegated failures when present, and the new recommendation directs users to recovery evidence.
- Raising the cache schema version means existing cached audits are regenerated with the corrected wording.

#### Negative
- Raising `auditSchemaVersion` invalidates older cached audits, so the first subsequent use of each requires regeneration.
- JSON and JSONL output gain a new field (`omitempty`, so it is absent when zero), and consumers must learn its meaning.
- The count reflects observed `FailedCount` only, not a count of unresolved tasks. Missing telemetry is not proof of success, and recovery still requires separate evidence even when a later retry completes.

#### Neutral
- The metric relies on `SubagentModelRequests` and deliberately does not count `DeclaredSubagentModels` again, because that field mirrors the requests.
- The CLI tester coverage contract is enforced by a string-matching Go test (`pkg/workflowcontract`), which links the prompt wording to test assertions.

---

*Draft completed against the PR implementation; acceptance remains pending maintainer review.*
