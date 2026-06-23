# ADR-41018: Reset `on:` Extension-Array Trackers at Event-Section Boundaries

**Date**: 2026-06-23
**Status**: Draft

## Context

`commentOutProcessedFieldsInOnSection` in `pkg/workflow/frontmatter_extraction_yaml.go` is a line-based state machine that walks the `on:` block and comments out compiler-processed extension fields (`bots`, `roles`, `labels`, `needs`, and their skip variants). It tracks "am I inside this array" with flat boolean flags (`inBotsArray`, `inRolesArray`, etc.). Because those flags were never reset when a new event section began, array state from one sibling key leaked into the next: when `bots:` or `roles:` appeared alongside `workflow_run:` in the same `on:` block, the stale array tracker caused the walker to comment out `workflow_run.workflows`/`types` list items. The stripped `workflows` then failed the non-empty-workflows compile-time validation, breaking an otherwise valid trigger composition.

## Decision

We will reset all top-level `on:` extension-array trackers at the start of every event section. A `resetOnArrayTrackers` closure clears `inSkipRolesArray`, `inSkipBotsArray`, `inRolesArray`, `inBotsArray`, `inLabelsArray`, and `inNeedsArray`, and it is invoked immediately before `activateEventSection` for each recognized section (`pull_request`, `issues`, `discussion`, `issue_comment`, `deployment_status`, `workflow_run`). This keeps the existing line-based scanner but makes its array state strictly section-local, so no sibling key can inherit a stale "inside array" flag.

## Alternatives Considered

### Alternative 1: Replace the line-based scanner with structured YAML scoping

We could parse the `on:` block into a node tree and decide field-by-field which keys to comment out, eliminating flat boolean state entirely. This is the more robust long-term fix, but it is a substantial rewrite of a security-relevant code path and risks changing comment-out behavior for the many already-covered cases. Given the bug is a narrow state-leak, the targeted reset was preferred over a structural rewrite under this PR's scope.

### Alternative 2: Reset only the bots/roles trackers, or only on entering `workflow_run`

We could clear just the two flags implicated in the reported failure, or reset only when the walker enters `workflow_run:`. This was rejected because the leak is a general property of the flat-flag design — any extension array preceding any event section can leak — so a partial reset would leave latent corruption for other key orderings (e.g. `labels`/`needs` before `issues`). Resetting all trackers at every section boundary fixes the class of bug rather than the single reported instance.

## Consequences

### Positive
- `workflow_run` triggers compose correctly with sibling `bots:`/`roles:` filters; `workflows`/`types` are no longer stripped, so the non-empty-workflows validation passes.
- The fix is minimal and localized, preserving the existing, well-tested line-based comment-out behavior for all other cases.
- Regression coverage is added at two levels: direct comment-out tests in `compiler_draft_test.go` (four key orderings) and strict-mode compile tests in `workflow_run_validation_test.go`.

### Negative
- The fragile flat-boolean state machine remains; the reset is a guard, not a structural fix, so similar state-leak bugs can recur if new trackers are added without being included in `resetOnArrayTrackers`.
- The reset list must be kept in sync with the set of extension-array trackers — a future tracker omitted from the closure would silently reintroduce the leak.

### Neutral
- Behavior is unchanged for `on:` blocks that do not mix extension arrays with event sections; the reset is a no-op when no array flag is set.
- The reset is tied to the fixed set of recognized event-section keys; adding a new event section requires adding the same `resetOnArrayTrackers()` call alongside its `activateEventSection` invocation.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
