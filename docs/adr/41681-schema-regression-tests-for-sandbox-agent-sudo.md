# ADR-41681: Schema Regression Tests for sandbox.agent.sudo

**Date**: 2026-06-26
**Status**: Draft
**Deciders**: Unknown

---

### Context

The `sandbox.agent.sudo` boolean field was introduced as a replacement for the deprecated `network-isolation` field in the workflow frontmatter parser (see #41679). The JSON schema at `pkg/parser/schemas/main_workflow_schema.json` already contained the `sudo` property, but there were no schema-level unit tests verifying that the validator accepted it. This created a silent drift risk: future changes to the Go struct YAML tags or the JSON schema could break the contract without any CI signal until an integration or end-to-end test caught it — much later in the pipeline. PR #41679 demonstrated this risk concretely by shipping a renamed field without a regression guard.

### Decision

We decided to add schema-level regression tests for `sandbox.agent.sudo` in `pkg/parser/schema_test.go`, covering: `sudo: false` accepted (with and without `id`), `sudo: true` accepted, the deprecated `network-isolation` field rejected as an unknown property, and a non-boolean `sudo` value rejected by the type constraint. This follows the established pattern of `TestMainWorkflowSchema_*` table-driven sub-tests already in the file and runs in CI on every pull request.

### Alternatives Considered

#### Alternative 1: Rely solely on integration and end-to-end tests

Integration tests exercise the full workflow execution path, which does include schema validation. The trade-off is that failure feedback is slow (integration tests are slower to run) and it is harder to pinpoint a schema-specific regression among broader test failures. This approach was rejected because it does not provide a fast, targeted signal for schema contract violations.

#### Alternative 2: Auto-generate schema validation tests from the JSON schema definition

Code generation from the JSON schema would keep test expectations automatically in sync with the schema file. However, no such generator exists in this codebase today and introducing one is significant new infrastructure. Given that the existing hand-authored test pattern already covers this need and is well-understood by the team, the overhead of building a generator was not justified by the scope of this change.

### Consequences

#### Positive
- Schema contract for `sandbox.agent.sudo` is explicitly enforced in CI, catching drift early.
- Rejection of the deprecated `network-isolation` field is now a tested invariant, not just an implicit behaviour.

#### Negative
- Tests must be manually updated whenever the `sandbox.agent` schema evolves (e.g., new fields, type changes); there is no automated synchronisation.
- Adding test code to `pkg/` inflates business-logic line counts used by automated gates, which may trigger false-positive ADR enforcement for future pure-test PRs.

#### Neutral
- The new test function follows the established `TestMainWorkflowSchema_*` pattern and does not introduce new test infrastructure.
- The tests are co-located with other schema tests in `pkg/parser/schema_test.go`, consistent with the existing file layout.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
