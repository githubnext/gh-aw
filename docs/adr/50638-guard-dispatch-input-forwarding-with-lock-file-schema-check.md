# ADR-50638: Guard workflow_dispatch Input Forwarding with Lock File Schema Check

**Date**: 2026-08-05
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

`gh aw trial` supports mixed-trigger trials where an entity-triggered workflow (which declares an `issue_number` `workflow_dispatch` input) runs alongside a scheduled or global companion workflow (which does not). Prior to this change, the `triggerWorkflowRun` function appended `--field issue_number=…` unconditionally for every workflow in the trial whenever a `--trigger-context` was provided.

GitHub's `workflow_dispatch` API rejects requests that include undeclared inputs with `HTTP 422: Unexpected inputs provided`. The failure was silent: the companion workflow was installed but never dispatched, with no user-visible indication that the trial had partially failed. The compiled `.lock.yml` file — already written to disk before dispatch — contains the full `on.workflow_dispatch.inputs` schema and is the authoritative source of truth for what each workflow accepts.

### Decision

We will parse each workflow's compiled lock file at dispatch time to check whether it declares the requested `workflow_dispatch` input before forwarding it. A new `workflowDeclaresDispatchInput(lockFilePath, inputName)` function reads and parses the lock file's `on.workflow_dispatch.inputs` map; if the input is absent (or the file is missing or unreadable), the function returns `false` and the input is silently omitted rather than forwarded. This fail-safe direction ensures that no workflow ever receives an undeclared input.

### Alternatives Considered

#### Alternative 1: Reject mixed trigger types before installation

Detect at the start of a trial that the selected workflows have incompatible trigger types and abort with an error message before any installation occurs. This would provide the clearest user feedback but would break the documented use case of pairing an issue-triggered workflow with a scheduled companion, which is an intentional and supported pattern.

#### Alternative 2: Strip trigger-derived inputs at a higher level

Remove all trigger-derived input forwarding from `executeTrialRun` for workflows whose names are not matched by the entity-triggered pattern, without consulting the lock file. This avoids per-dispatch I/O but introduces a naming-convention dependency and would silently drop inputs for any future workflow type that legitimately accepts `issue_number` without matching the entity pattern.

### Consequences

#### Positive
- Mixed-trigger trials (entity-triggered workflow + scheduled companion) now dispatch correctly without HTTP 422 errors.
- The lock file is already present locally at dispatch time, so no additional network calls are required.
- Fail-safe semantics: parse or read errors return `false`, ensuring inputs are never forwarded to workflows that would reject them.
- Behavior is fully tested: declared, undeclared, no-inputs, schedule-only, and missing-file cases are all covered.

#### Negative
- Each workflow dispatch now incurs an additional file read and YAML parse of the lock file, adding minor I/O overhead at dispatch time.
- When the lock file is missing or malformed, the input is silently dropped with only a log entry; the operator may not notice that the trigger context was not forwarded.
- The approach is specific to `issue_number`; forwarding any future trigger-derived inputs will require the same guard to be extended or generalized.

#### Neutral
- The `triggerWorkflowRun` signature gains a `lockFilePath` parameter, which changes the internal API surface and requires callers to construct the path.
- Verbose mode now emits an informational message when an input is omitted rather than silently skipping it, improving debuggability.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
