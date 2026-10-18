# ADR-69868: Report Sub-agent Failures Separately from the Parent Workflow Conclusion

**Date**: 2026-10-18
**Status**: Draft
**Deciders**: pelikhan, Unknown

---

### Context

`gh aw audit` reported a successful parent workflow as "Completed in N turns with no errors" even when delegated sub-agent invocations failed. The Contribution Check run 39098581712 showed this: the Actions conclusion was `success` with `error_count: 0`, yet two delegated invocations failed. The per-invocation failure data already existed in `ProcessedRun.TokenUsage.SubagentModelRequests[].FailedCount` and as individual `subagent_failed` findings. However, no aggregate metric showed it, and the success narrative implied that delegated work had completed. Audit results are cached with a versioned schema (`auditSchemaVersion`), so cached reports keep the old wording until the version changes. The daily CLI tools tester had a similar truthfulness problem: it reported expected coverage exclusions as incomplete work and gave a blanket all-tools-success message.

### Decision

We will report observed delegated failures as a separate metric, `metrics.subagent_failure_count`, summed from `SubagentModelRequests[].FailedCount`. We will not fold them into `error_count` or change the parent `conclusion`. When this count is non-zero, the `workflow_succeeded` finding will say "Parent Workflow Completed Successfully" and state that sub-agent invocations failed. A high-priority recommendation will tell users to verify recovery instead of inferring it from the parent's success. We will add the field to `audit.schema.json` and `logs-jsonl.schema.json`, show `subagent_failures=` in console output, and raise `auditSchemaVersion` from 7 to 8 so stale cached narratives are rebuilt. Similarly, the CLI tester will mark expected exclusions as "not tested". They will not count as passes or trigger `report_incomplete`, but blocked or unfinished required checks will still trigger it.

### Alternatives Considered

#### Alternative 1: Count sub-agent failures in `error_count`

Adding delegated failures to `error_count` would surface them without a schema change. We rejected this because it would misstate the parent run, which reported zero workflow errors and concluded `success`. It would also mix two different signals, so consumers could no longer tell parent errors apart from delegated failures. Delegated failures may also have been recovered by retries or a parent fallback, so counting them as workflow errors would overstate the problem.

#### Alternative 2: Rely only on the existing per-invocation `subagent_failed` findings

The individual findings were already emitted, so the PR could have kept the status quo. We rejected this because the success finding still said "no errors", which contradicted those findings. There was also no aggregate number for JSON/JSONL consumers or console readers to filter on.

#### Alternative 3: Mark the overall audit as failed when any sub-agent fails

Overriding the conclusion would make failures impossible to miss. We rejected this because the audit must preserve the actual GitHub Actions conclusion. Delegated failures can be benign or recovered, so turning them into a failure verdict would presume an outcome that the evidence does not establish.

### Consequences

#### Positive
- Parent success, parent errors and delegated failures are now distinct, machine-readable signals (`conclusion`, `error_count`, `subagent_failure_count`).
- The success narrative no longer claims "no errors" when delegated work failed, and the new recommendation sends users to the right evidence.
- Raising the cache schema version means existing cached audits are regenerated with the corrected wording.

#### Negative
- Raising `auditSchemaVersion` invalidates every cached audit, so the first audit of each run after upgrading is reprocessed.
- JSON and JSONL output gain a new field (`omitempty`, so it is absent when zero), and consumers must learn its meaning.
- The count reflects observed `FailedCount` only. It cannot show whether failures were later recovered, so manual verification is still needed.

#### Neutral
- The metric relies on `SubagentModelRequests` and deliberately does not count `DeclaredSubagentModels` again, because that field mirrors the requests.
- The CLI tester coverage contract is enforced by a string-matching Go test (`pkg/workflowcontract`), which links the prompt wording to test assertions.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
