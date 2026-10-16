# ADR-69369: Support Explicit `create-issue: false` Opt-Out for Safe Outputs

**Date**: 2026-10-15
**Status**: Draft (PR #69369 is not yet accepted)
**Deciders**: Not recorded; pending maintainer review of PR #69369. This draft does not establish approval.
**Author**: pelikhan (PR author), drafted by the Design Decision Gate

## Context

When a workflow declares no `safe-outputs:` section, or declares only system types (`noop`, `missing-tool`, `missing-data`), the compiler auto-injects `create-issue` with conservative defaults (`max: 1`, labels and title prefix derived from the workflow ID) and an accompanying prompt instruction telling the agent to report via an issue. The previous documented opt-out — "add an explicit `safe-outputs:` section with the outputs you want" — does not work for workflows that need system outputs but must never publish issues: those configurations still trip the auto-injection rule.

There was therefore no way to express "this workflow produces no issues" without also fabricating an unrelated non-system safe output. The gap affects both the generated `create_issue` tool and the generated reporting instruction, and it must survive import merging, where a top-level configuration takes precedence over imported fragments. The fix must not change the default behaviour for workflows that omit the key, pass `null`, or supply an object configuration (see `pkg/workflow/create_issue.go`, `safe_outputs_config_extraction.go`, `safe_outputs_state.go`, `imports.go`).

## Decision

The proposal in PR #69369 is to accept the boolean literal `false` as a value for `create-issue` and treat it as an explicit disablement signal:

```yaml
safe-outputs:
  create-issue: false
```

- The JSON schema (`main_workflow_schema.json`) gains a `const: false` branch for `create-issue`, so the value validates without loosening the existing object/null forms.
- Configuration extraction records the intent in a new internal field `SafeOutputsConfig.CreateIssuesDisabled` (`yaml:"-"`, not part of the user-facing surface) rather than relying on a nil config, which is indistinguishable from "not configured".
- `parseIssuesConfig` returns `nil` when the raw value is boolean `false`, so no `create_issue` tool is emitted.
- The prompt generator in `safe_outputs_state.go` returns early when `CreateIssuesDisabled` is set, so the automatic reporting instruction is suppressed alongside the tool.
- Import merging propagates disablement sticky-OR style: if any merged configuration disables issue creation, the result stays disabled, while top-level configuration precedence is preserved.
- System outputs (`noop`, `missing-tool`, `missing-data`) remain enabled, so a disabled workflow can still signal completion and report gaps.

The driver is a correctness constraint rather than a performance or cost concern: a workflow that must not publish issues currently has no faithful way to say so.

## Alternatives Considered

### Alternative 1: Document the existing workaround (declare a placeholder non-system output)

Authors could add any non-system safe output to suppress auto-injection. This requires no compiler change, but it forces workflows to request capabilities they do not want, grants unnecessary write intent, and leaves the intent implicit and easy to break during refactoring. It also fails for workflows whose only legitimate outputs are system types.

### Alternative 2: Represent disablement as a `nil`/absent `CreateIssues` config instead of a dedicated flag

Reusing the existing nil config avoids a new struct field. It was rejected because `nil` already means "not configured", which is exactly the state that triggers auto-injection; the two cases must be distinguished to suppress both the tool and the instruction, and to merge correctly across imports.

### Alternative 3: Introduce a general `safe-outputs: false` or a separate `disable:` list

A generic disablement mechanism would cover future outputs uniformly. It was rejected for this change as a larger surface-area decision: it would need precedence rules against every output type and a migration story, whereas only the auto-injected output currently lacks an opt-out.

## Consequences

### Positive

- Workflows that must not publish issues can state that intent declaratively, and the generated agent prompt no longer contradicts the policy.
- The opt-out is schema-validated, so typos such as `create-issue: no` or `create-issue: "false"` are rejected at compile time rather than silently ignored.
- Disablement is preserved through import merging, so a shared fragment cannot silently re-enable issue creation.

### Negative

- `SafeOutputsConfig` gains a second representation of "no issue output" (`CreateIssues == nil` vs `CreateIssuesDisabled == true`); future code touching issue configuration must check both, and missing the flag reintroduces the bug.
- `create-issue` now accepts a value shape (`false`) that no other safe output accepts, creating a temporary inconsistency until a general disablement mechanism exists.
- Sticky-OR merge semantics mean an imported fragment can disable issue creation for importers; this is intentional for safety but is not symmetric with other merge rules and may surprise authors.

### Neutral

- Default behaviour is unchanged for omitted, `null`, and object configurations; existing workflows and their lock files are unaffected.
- Documentation in `docs/src/content/docs/reference/safe-outputs.md` replaces the previous opt-out guidance with the supported configuration.
- Regression coverage is added in `pkg/workflow/safe_outputs_default_create_issue_test.go` and `pkg/parser/schema_safe_outputs_target_test.go`, covering tool suppression, prompt suppression, import merging, and schema acceptance/rejection.

---

*ADR drafted by the Design Decision Gate for PR #69369. Review and finalize before changing status from Draft to Accepted.*
