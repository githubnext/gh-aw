# ADR-26382: Auto-Generate Side-Repo Maintenance Workflows for SideRepoOps Pattern

**Date**: 2026-04-15
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

gh-aw supports a SideRepoOps pattern where a workflow hosted in one repository (`current: true` checkout) operates against a separate target repository. Previously, the target repository's maintenance workflow — responsible for replaying safe outputs, creating labels, and closing expired entities — had to be created manually and re-synchronized by hand on every `gh aw upgrade` cycle. This manual process was error-prone, required detailed knowledge of the gh-aw internals, and was frequently left out of sync with the hosting repository's generated workflows. With the SideRepoOps pattern seeing broader adoption, a sustainable automated solution became necessary.

### Decision

We will automatically detect SideRepoOps targets at compile time by scanning all `WorkflowData` entries for checkout configurations with `current: true` and a static (non-expression) `repository` field, then generate a per-target `agentics-maintenance-<owner-repo>.yml` workflow alongside the standard `agentics-maintenance.yml`. The generated side-repo maintenance workflow is pre-wired with the checkout config's custom token (falling back to `${{ secrets.GH_AW_GITHUB_TOKEN }}`) and sets `GH_AW_TARGET_REPO_SLUG` on every cross-repo job, enabling all maintenance operations to act against the target repository without any manual configuration. Expression-based repository values (e.g. `${{ inputs.target_repo }}`) are excluded since no static filename can be derived.

### Alternatives Considered

#### Alternative 1: Manual Workflow Authoring (Status Quo)

The existing approach required repository owners to author and maintain a custom side-repo maintenance workflow by hand. While this gave maximum flexibility, it imposed ongoing maintenance burden and required users to track internal implementation changes across upgrades. It was rejected because it directly contradicts gh-aw's stated goal of eliminating boilerplate agentic workflow management.

#### Alternative 2: On-Demand CLI Command

A dedicated CLI command (e.g. `gh aw generate-side-repo-maintenance --repo owner/repo`) could generate the file when explicitly invoked. This was rejected because it requires users to know about the feature, remember to run it after changes, and re-run it after every upgrade — preserving the core problem of manual synchronization. Auto-detection at compile time guarantees the files stay in sync without user intervention.

#### Alternative 3: Dynamic Workflow Invocation via `workflow_call` Parameters

Instead of a per-target file, a single parameterized `agentics-maintenance-side-repo.yml` could accept the target slug as an input. This was rejected because: (a) it requires callers to supply the slug explicitly, (b) cross-repo token selection cannot be resolved statically without a per-target file, and (c) GitHub Actions does not support dynamically choosing secrets by name at runtime.

### Consequences

#### Positive
- Side-repo maintenance workflows are always in sync with the hosting repo's compile output; no manual re-synchronization required after upgrades.
- Correct token selection is handled automatically: the generated workflow uses the same GitHub token declared in the checkout config.
- `GH_AW_TARGET_REPO_SLUG` is injected on every cross-repo job, enabling all maintenance JavaScript actions to operate against the right repository without code changes.
- The side-repo workflow is generated even when no `expires` configuration exists, ensuring `safe_outputs` and `create_labels` operations are always available.

#### Negative
- Only workflows with a static (literal string) `repository` field in their checkout config generate a side-repo maintenance workflow; expression-based targets (e.g. `${{ inputs.target_repo }}`) are silently skipped with a log message.
- Each unique static target produces a separate workflow file, which may grow the `.github/workflows/` directory noticeably in repositories with many side-repo targets.
- The `GH_AW_TARGET_REPO_SLUG` environment variable is now load-bearing for cross-repo operations; misconfiguration or accidental override in calling workflows could misdirect operations.

#### Neutral
- The `GenerateMaintenanceWorkflow` function's control flow is modified: the early-return path for the no-`expires` case now calls `generateAllSideRepoMaintenanceWorkflows` before returning, which is a behavioural change for consumers relying on "no file output when no expires."
- Four JavaScript maintenance scripts (`close_expired_discussions.cjs`, `close_expired_issues.cjs`, `close_expired_pull_requests.cjs`, `create_labels.cjs`) now branch on `GH_AW_TARGET_REPO_SLUG` to resolve `owner`/`repo`, affecting all execution contexts — including non-SideRepoOps runs where the variable is absent (falls back to `context.repo` as before).

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Side-Repo Target Detection

1. Implementations **MUST** treat a checkout configuration as a SideRepoOps target if and only if `current` is `true` and `repository` is a non-empty string that does not contain `${{`.
2. Implementations **MUST NOT** generate a side-repo maintenance workflow for checkout configurations where `repository` contains a GitHub Actions expression (i.e. the string `${{`).
3. Implementations **MUST** deduplicate targets by `repository` slug so that at most one maintenance workflow file is generated per unique target repository, regardless of how many compiled workflows reference it.
4. Implementations **SHOULD** emit a log message for each skipped expression-based repository to aid debugging.

### Workflow File Naming

1. Implementations **MUST** derive the side-repo maintenance workflow filename as `agentics-maintenance-<sanitized-slug>.yml`, where `<sanitized-slug>` is the `owner/repo` string with `/` replaced by `-` and any character outside `[a-zA-Z0-9\-_.]` replaced by `-`.
2. Implementations **MUST** write the generated file to the same directory as the standard `agentics-maintenance.yml` (the `workflowDir` parameter).
3. Implementations **MUST NOT** overwrite an existing `agentics-maintenance.yml` (the standard hosting-repo workflow) when generating side-repo maintenance files.

### Generated Workflow Content

1. Implementations **MUST** include `workflow_call` and `workflow_dispatch` triggers in every generated side-repo maintenance workflow.
2. Implementations **MUST** set the `GH_AW_TARGET_REPO_SLUG` environment variable to the static `owner/repo` slug on every job step that performs operations against the target repository.
3. Implementations **MUST** use the `github-token` value from the checkout configuration as the GitHub token for cross-repo job steps; when no token is configured, implementations **MUST** fall back to `${{ secrets.GH_AW_GITHUB_TOKEN }}`.
4. Implementations **MUST** include `apply_safe_outputs`, `create_labels`, and `validate_workflows` jobs in every generated side-repo maintenance workflow, regardless of whether `expires` is configured.
5. Implementations **MUST** include the `close-expired-entities` job only when `hasExpires` is `true` for the workflow set.
6. Implementations **SHOULD** include a human-readable comment in the generated workflow identifying the target repository and the fact that the file is auto-generated.

### Cross-Repo JavaScript Action Resolution

1. JavaScript maintenance scripts that operate on GitHub resources **MUST** resolve `owner` and `repo` from `GH_AW_TARGET_REPO_SLUG` when that environment variable is set and matches the pattern `/^[^/]+\/[^/]+$/`; otherwise they **MUST** fall back to `context.repo.owner` and `context.repo.repo`.
2. Implementations **MUST NOT** split `GH_AW_TARGET_REPO_SLUG` on more than one `/` (i.e. `split("/", 2)` semantics are required).
3. Implementations **SHOULD** emit an informational log message when `GH_AW_TARGET_REPO_SLUG` is used, identifying the resolved `owner/repo`.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24455822765) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
