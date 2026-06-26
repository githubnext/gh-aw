# ADR-41617: Workflow-Name Filtering in `--org` Mode for `gh aw update`

**Date**: 2026-06-26
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The `gh aw update --org` command previously treated org mode as all-or-nothing: it scanned every source-managed workflow across all repositories in the organization, regardless of which specific workflows the caller requested. Positional workflow name arguments passed to `update` were silently ignored when `--org` was active. For large organizations this meant unnecessary GitHub code-search queries and expensive shallow-checkout work on repositories that did not contain the requested workflows, increasing latency and API consumption even when operators needed to update only one or two specific workflows.

### Decision

We will thread `UpdateWorkflowsOptions.WorkflowNames` through both the org-wide repository discovery phase (via a richer GitHub code-search query) and the per-repository preview phase (via `findWorkflowsWithSource`). A new helper, `buildOrgWorkflowSearchQuery`, normalizes workflow IDs, deduplicates them, and appends deterministic `filename:` predicates to the base code-search query so that only repositories containing at least one requested workflow are returned as candidates. When no workflow names are specified, the query degrades to the original broad search, preserving backwards compatibility.

### Alternatives Considered

#### Alternative 1: Post-filter (scan all repos, discard non-matching results)

Keep the existing broad discovery unchanged and filter out non-matching workflows only during the per-repo preview stage. This is simpler to implement—no query construction logic is needed—but wastes GitHub code-search quota and per-repo shallow-checkout work on repositories that will ultimately be discarded. For organizations with hundreds of repositories this is prohibitively expensive.

#### Alternative 2: Separate subcommand for targeted org updates

Introduce a dedicated `gh aw update --org --only <workflow>` subcommand path rather than enriching the existing `--org` mode. This avoids changing the `searchOrgWorkflowRepos` function signature and keeps the original code path untouched. However, it duplicates orchestration logic, splits the mental model for operators, and provides no improvement to the common case where a user passes workflow names alongside `--org` expecting targeted behavior.

### Consequences

#### Positive
- Reduced API consumption: GitHub code-search results are narrowed before expensive per-repo shallow checkouts occur, cutting unnecessary network requests in proportion to the number of unrelated repos in the org.
- Correct semantics: workflow name arguments now take effect consistently in both single-repo and org modes, eliminating silent arg-ignore behavior.
- Deterministic queries: `buildOrgWorkflowSearchQuery` normalizes IDs and sorts filename predicates, making search queries reproducible and testable.

#### Negative
- Breaking internal API: the `searchOrgWorkflowRepos` function signature adds a `workflowNames []string` parameter, requiring updates to all call sites and mock stubs in tests.
- Additional query-construction complexity: `buildOrgWorkflowSearchQuery` introduces a normalization and deduplication step that must be kept in sync with `normalizeWorkflowID`; bugs here silently widen or narrow the candidate repo set.

#### Neutral
- The `searchOrgWorkflowReposFn` function variable type changes accordingly, so any external callers injecting a custom search function (e.g., in integration tests) must update their signatures.
- The optimization applies only when at least one workflow name is specified; zero-argument invocations produce the same query as before.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
