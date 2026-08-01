# ADR-49412: Extend `auto-merge` to Support Explicit Merge Methods

**Date**: 2026-08-01
**Status**: Draft
**Deciders**: Unknown

---

### Context

The `safe-outputs.create-pull-request` handler supports an `auto-merge` boolean field that enables GitHub's native auto-merge on the created PR. When set to `true`, the handler calls `enablePullRequestAutoMerge` via GraphQL without specifying a `mergeMethod`, so the PR uses the repository's default merge strategy. Scheduled and automated workflows that produce structured, single-purpose PRs (e.g. docs regeneration, dependency bumps) need to enforce a specific merge method — squash, merge, or rebase — rather than relying on the repository default, which varies across repos and cannot be guaranteed. A workaround using the undocumented `created_pr_number` output and a custom safe-output job existed but depended on internal implementation details.

### Decision

We will extend the `auto-merge` field to accept, in addition to the existing `true`/`false` boolean, one of the string literals `"squash"`, `"merge"`, or `"rebase"`. When an explicit strategy string is provided, the handler will pass the corresponding `PullRequestMergeMethod` enum value (`SQUASH`, `MERGE`, `REBASE`) to `enablePullRequestAutoMerge`. When `true` is specified without an explicit method, it defaults to `SQUASH`. The value `false` disables auto-merge. Schema validation is extended with a `oneOf` to cover both the boolean and string-enum shapes.

### Alternatives Considered

#### Alternative 1: Add a Separate `auto-merge-method` Field

Introduce a new `auto-merge-method: squash | merge | rebase` field alongside the existing boolean `auto-merge`. This keeps the boolean field semantically clean (enabled/disabled) and separates the method concern. It was not chosen because it doubles the config surface for a tightly coupled concern, and requiring both fields to be consistent adds cognitive overhead and a new class of misconfiguration (method set but auto-merge disabled).

#### Alternative 2: Keep `auto-merge: true` with Repository Default Only

Accept that auto-merge always uses the repository default merge strategy and document the workaround using `gh pr merge --auto`. This was not chosen because the workaround depends on undocumented safe-outputs internals (`created_pr_number`) that may change across releases, creating a fragile coupling between user workflows and implementation details.

### Consequences

#### Positive
- Workflow authors can enforce a specific merge strategy for auto-merged PRs without relying on the repository default, making agentic workflows predictable across repos with varying defaults.
- `auto-merge: true` now defaults to `SQUASH`, providing a sensible and consistent default. Explicit method strings (`squash`, `merge`, `rebase`) and `auto-merge: false` continue to work as before.

#### Negative
- The `auto-merge` field now has a polymorphic type (boolean or string enum), which increases schema complexity. The `oneOf` with three variants (boolean, string enum, and GitHub Actions expression pattern) is more difficult to document clearly than a simple boolean.
- The new `parseAutoMergeConfig` function in the runtime handler introduces an additional parsing layer that must stay in sync with the compile-time validation logic in `parseCreatePullRequestsConfig`.

#### Neutral
- The `auto-merge` field is removed from the `BoolFields` list in `parseCreatePullRequestsConfig` and is now handled by a dedicated validation block, making the compile-time parsing path slightly more explicit.
- GitHub Actions template expressions (`${{ ... }}`) resolving to any of the accepted values are supported at the schema level, maintaining consistency with other polymorphic fields in the safe-outputs config.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
