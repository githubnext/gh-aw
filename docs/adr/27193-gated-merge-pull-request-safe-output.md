# ADR-27193: Gated `merge-pull-request` Safe-Output with Policy-Driven Merge Enforcement

**Date**: 2026-04-19
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The gh-aw agentic workflow platform already supports a safe-output model in which agents can perform real side-effects (creating issues, posting comments, etc.) only through a compiler-validated, runtime-gated execution path. Until this change, there was no way for an agent to merge a pull request through the same safety layer. Merging is a high-consequence, irreversible action that must be gated on repository policy (CI status, review approval, label constraints, and branch restrictions) before it can be executed safely. The existing safe-output infrastructure — a Go compiler that validates frontmatter configuration and a Node.js runtime handler layer — already provides the extension point needed to add merge support without inventing a separate execution path.

### Decision

We will add `merge-pull-request` as a new safe-output type that integrates with the existing compiler and runtime handler model rather than introducing a standalone merge action. The merge handler evaluates a sequenced set of policy gates — CI checks, review decision, unresolved threads, required/allowed labels, source-branch allow-list, default-branch protection, draft state, mergeability, and conflict state — and only proceeds when all gates pass. Configuration is expressed in workflow YAML frontmatter under `safe-outputs.merge-pull-request` using the same typed-config pattern already used by other safe-output types.

### Alternatives Considered

#### Alternative 1: Standalone Merge Action Outside the Safe-Output Model

A dedicated GitHub Actions action or a separate Go command could have been written to perform gated merges independently of the safe-output layer. This would have been simpler to prototype but would have forked the security model: safe-outputs validate permissions at compile time, enforce `max` call budgets, and provide a single auditable execution path. A standalone action would duplicate that plumbing or omit it entirely, leaving merge calls outside the auditable boundary.

#### Alternative 2: Thin Merge Wrapper With No Policy Gates

The handler could have simply called the merge API and relied on external branch-protection rules configured in GitHub repository settings. This reduces code but shifts policy configuration to GitHub UI settings, making it invisible to code reviewers and hard to version-control. Policy gates expressed in workflow frontmatter are auditable, diffable, and scoped to the specific workflow rather than globally to the repo.

#### Alternative 3: Separate Runtime Execution Path for High-Risk Operations

Merge could have been treated as a distinct risk tier requiring its own runtime pipeline separate from lower-risk safe-output types. This would allow future independent evolution of merge-specific policy but introduces architectural fragmentation immediately without a concrete need. The existing model already supports configuration-driven per-type gates, so a separate pipeline is premature.

### Consequences

#### Positive
- Merge operations are now auditable through the same compiler + runtime path as all other safe-output types.
- Policy gates (labels, branches, CI, reviews, files) are version-controlled in workflow YAML frontmatter rather than scattered across GitHub repository UI settings.
- Shared `check_runs_helpers.cjs` eliminates logic duplication between merge gating and the existing `check_skip_if_check_failing` safe-output.
- `withRetry` wrapping of mergeability and GraphQL review-summary calls handles eventual-consistency delays from the GitHub API without requiring callers to manage retry logic.
- Idempotency: if the PR is already merged the handler returns success, making the operation safe to re-run.

#### Negative
- The gate evaluation logic is complex (10+ sequential checks) and lives entirely in a single `.cjs` handler file; future contributors extending the gate list must understand the full sequencing.
- Retry-backed mergeability polling adds latency on every merge attempt, even when mergeability is immediately available.
- Adding a new safe-output type increases schema surface area in `main_workflow_schema.json` and both `safe_outputs_tools.json` catalogs, which must be kept in sync manually.

#### Neutral
- The `contents:write` + `pull-requests:write` permission pair must be present in any workflow that uses `merge-pull-request`; this is enforced at compile time but requires authors to explicitly declare permissions.
- The W3C-style Safe Outputs specification (`docs/src/content/docs/reference/safe-outputs-specification.md`) was updated to include a formal `merge_pull_request` section, continuing the precedent of spec-first documentation for safe-output types.
- A Go spec-enforcement test (`safe_outputs_specification_merge_pull_request_test.go`) was added to prevent spec drift; this test must be updated if the type name or required policy statements change.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Safe-Output Model Integration

1. The `merge-pull-request` capability **MUST** be implemented as a safe-output type within the existing compiler-plus-runtime-handler model and **MUST NOT** introduce a separate merge execution path outside that model.
2. Configuration for `merge-pull-request` **MUST** be expressed in workflow YAML frontmatter under the `safe-outputs.merge-pull-request` key, using the same typed-config parsing pattern used by other safe-output types.
3. The compiler **MUST** validate `merge-pull-request` configuration at compile time, including configured branch and label constraints (`allowed-branches`, `allowed-labels`, `required-labels`).
4. The runtime handler **MUST** be registered in the safe-output handler manager alongside all other safe-output handlers.

### Policy Gate Evaluation

1. Before invoking the merge API, the runtime handler **MUST** evaluate all of the following gates in order, and **MUST** abort with a descriptive error if any gate fails:
   a. Draft state — the PR **MUST NOT** be a draft.
   b. Mergeability — the PR **MUST** be in a mergeable state (not conflicting, not blocked).
   c. CI checks — all required check runs **MUST** be passing; the handler **MUST** exclude deployment-environment check runs from this evaluation.
   d. Review decision — the PR's review decision **MUST NOT** be `CHANGES_REQUESTED` or `REVIEW_REQUIRED` when those states are present.
   e. Unresolved review threads — the PR **MUST** have zero unresolved review threads.
   f. Required labels — every label in `required-labels` **MUST** be present on the PR.
   g. Allowed labels — when `allowed-labels` is configured, at least one PR label **MUST** exactly match a configured label name.
   h. Allowed branches — when `allowed-branches` is configured, the PR source branch **MUST** match at least one configured glob pattern.
   i. Default-branch protection — the PR target branch **MUST NOT** be the repository default branch.
2. Gate evaluation **MUST** be idempotent: if the PR is already merged the handler **MUST** return a success response without attempting another merge.
3. Mergeability retrieval **MUST** use retry logic to handle GitHub API eventual-consistency delays; implementations **SHOULD** retry at least 3 times with exponential back-off before reporting failure.

### Shared Infrastructure

1. Check-run filtering and deduplication logic **MUST** be implemented in a shared helper module (`check_runs_helpers.cjs`) and **MUST NOT** be duplicated in individual safe-output handlers.
2. GraphQL calls used to retrieve review summary data **SHOULD** be wrapped with retry logic to tolerate transient API failures.

### Schema and Permissions

1. The `merge-pull-request` type **MUST** be declared in `main_workflow_schema.json` and in all `safe_outputs_tools.json` catalogs used by compiler and runtime.
2. Any workflow using `merge-pull-request` **MUST** declare `contents: write` and `pull-requests: write` permissions; the compiler **MUST** enforce this at compile time.
3. The W3C-style Safe Outputs specification **MUST** include a formal section documenting the `merge_pull_request` type, its policy gates, and its required permissions.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24632957089) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
