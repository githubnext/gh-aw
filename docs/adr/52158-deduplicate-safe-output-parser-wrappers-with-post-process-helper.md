# ADR-52158: Deduplicate Safe-Output Parser Wrappers with a Post-Process Helper

**Date**: 2026-08-11
**Status**: Proposed — pending maintainer acceptance on merge
**Deciders**: pelikhan (via copilot-swe-agent, PR #52158)

---

### Context

The `pkg/workflow` package provides safe-output handlers for GitHub Actions workflow steps. Each handler calls the shared `parseConfigScaffold` helper to unmarshal its YAML config, but then repeats the same boilerplate: check whether the returned pointer is nil, apply default field values, and emit a post-parse debug log line. This pattern appeared identically across 13+ non-test files (`add_comment.go`, `add_labels.go`, `add_reviewer.go`, `assign_milestone.go`, `assign_to_agent.go`, `assign_to_user.go`, `close_entity_helpers.go`, `mark_pull_request_as_ready_for_review.go`, `remove_labels.go`, `replace_label.go`, `set_issue_field.go`, `set_issue_type.go`, `unassign_from_user.go`). Because each copy was hand-maintained, defaulting rules and log formatting could drift silently when one handler received a fix that peers did not.

### Decision

We will introduce `parseConfigScaffoldWithPostProcess[T any]` in `pkg/workflow/config_helpers.go`. This generic wrapper calls `parseConfigScaffold`, and if the result is non-nil and a `postProcess` callback is provided, invokes that callback before returning. All 13+ existing callers are refactored to use the new helper, moving their nil-check, default-value assignment, and post-parse logging into the `postProcess` closure. Handlers with genuinely unique pre-parse logic (e.g., `create_entity_helpers.go`) are left unchanged. No behavioral changes are introduced; existing defaults, error messages, and log formats are preserved exactly.

The callback contract is: `postProcess` runs for **every** non-nil result, including a non-nil fallback returned by `onError`, and is skipped when the result is nil (key absent, or `onError` returned nil to disable the handler). This keeps defaulting and logging consistent between successfully parsed configs and error fallbacks. The contract is pinned down by direct tests in `config_scaffold_helpers_test.go` covering valid config, non-nil error fallback, nil error fallback, absent key, and a nil callback.

### Alternatives Considered

#### Alternative 1: Extend `parseConfigScaffold` directly with an optional postProcess parameter

Add a `postProcess func(config *T)` parameter to the existing `parseConfigScaffold` signature so all callers can supply post-processing inline without a new function.

This was not chosen because changing `parseConfigScaffold`'s signature would require updating every existing call site (including those that pass `nil` or do not need post-processing), and would complicate the function's existing contract around nil handling and error fallback. A thin wrapper preserves backward compatibility and keeps `parseConfigScaffold` focused.

#### Alternative 2: Define a `PostProcessable` interface on each config type

Give each config struct a `PostProcess()` method implementing a shared interface, and have `parseConfigScaffold` detect and call it via a type assertion or a separate generic constraint.

This was not chosen because it would scatter post-processing logic across many struct definitions rather than co-locating it with the handler's parse function. It also increases coupling between the generic scaffold and the concrete config types, making the scaffold harder to reason about in isolation.

### Consequences

#### Positive
- The nil-check-and-callback wrapper is written once and reused across all handlers, eliminating 13+ nearly identical code blocks.
- Default-value policy and post-parse logging conventions can be changed in a single `postProcess` closure per handler rather than tracked across many files.
- New handlers automatically follow the pattern by passing a `postProcess` closure instead of copying boilerplate.
- Handler files are shorter and easier to read.

#### Negative
- `config_helpers.go` grows by one additional generic function, adding to the surface area readers must understand when onboarding.
- Handlers with unusual post-processing (e.g., extracting nested map keys as in `mark_pull_request_as_ready_for_review.go`) must still write non-trivial closures, so the helper provides less benefit for those cases.

#### Neutral
- `create_entity_helpers.go` is explicitly excluded from the refactor because it already owns a distinct `parseCreateEntityConfig` scaffold with pre- and post-hooks; the two scaffolds coexist without conflict.
- The refactor is purely internal to `pkg/workflow`; no public API or configuration schema changes.
