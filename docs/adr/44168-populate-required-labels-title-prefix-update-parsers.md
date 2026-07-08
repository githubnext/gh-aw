# ADR-44168: Populate Required-Labels and Required-Title-Prefix in Update-Pull-Request and Update-Issue Parsers

**Date**: 2026-07-08
**Status**: Draft
**Deciders**: pelikhan, Copilot SWE Agent

---

### Context

The `update-pull-request` and `update-issue` safe-output workflow types support `required-labels` and `required-title-prefix` filter fields in YAML configuration. These fields are embedded in the shared `SafeOutputFilterConfig` struct and were expected to constrain which PRs and issues are eligible for update. However, the parsers (`parseUpdatePullRequestsConfig` and `parseUpdateIssuesConfig`) never called `ParseFilterConfig(configMap)`, leaving both fields zero-valued at compile time. As a result, the runtime received empty constraints and passed every item unconditionally — silently ignoring any `required-labels` or `required-title-prefix` values specified by the workflow author.

All other safe-output workflow types (`add-comment`, `merge-pull-request`, `push-to-pull-request-branch`) already invoke `ParseFilterConfig` correctly. This was an oversight limited to the two update parsers introduced in a separate PR.

### Decision

We will add `ParseFilterConfig(configMap)` calls to both `parseUpdatePullRequestsConfig` and `parseUpdateIssuesConfig` so that `required-labels` and `required-title-prefix` filters are parsed from the config map and propagated to the runtime, consistent with every other safe-output workflow type. Tests are added alongside the fix to guard against regression.

### Alternatives Considered

#### Alternative 1: Remove the filter fields from the update-* structs

Formally acknowledge that `update-pull-request` and `update-issue` do not support `required-labels` or `required-title-prefix`, remove the struct fields, and document the limitation. This would eliminate the silent-ignore behavior by making the limitation explicit at the type level.

This was rejected because the infrastructure for these filters was already designed to be shared across all output types. Removing the fields from two types would create an inconsistent user-facing API and require documentation updates and conditional logic in the emitter — more complexity than a two-line fix.

#### Alternative 2: Fail compilation when unsupported filter fields are specified

Add validation at parse time that returns an error if `required-labels` or `required-title-prefix` are specified for output types that do not call `ParseFilterConfig`. This would surface the gap immediately instead of silently ignoring the config.

This was rejected as too heavy for a bug fix: the existing parser already handles unknown keys gracefully, and the correct path is to support the fields, not forbid them. Validation would need to be maintained separately as new output types are added.

### Consequences

#### Positive
- `required-labels` and `required-title-prefix` now work for `update-pull-request` and `update-issue`, making filter behavior consistent across all safe-output workflow types.
- Workflow authors who already specified these fields expecting them to work will have their intent honored without any configuration changes on their end.

#### Negative
- Workflows that relied on the (unintentional) pass-through behavior — where every PR or issue was updated regardless of labels or title — will now be filtered. This is a silent behavioral change for any workflow that specified `required-labels` or `required-title-prefix` and expected them to be ignored.
- The bug existed since the `update-pull-request`/`update-issue` types were introduced, so any in-production workflow depending on the broken behavior must be reviewed.

#### Neutral
- The fix is implemented with `ParseFilterConfig` — the same shared function used by other output types — so no new parser logic is introduced.
- The deprecated `title-prefix` field in `update-issue` continues to be extracted separately from `required-title-prefix`, preserving backward compatibility.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
