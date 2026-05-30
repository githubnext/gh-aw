# ADR-29996: Experiment State Storage — Git Branch as Default, Cache as Fallback

**Date**: 2026-05-03
**Status**: Draft
**Deciders**: pelikhan (PR author)

---

## Part 1 — Narrative (Human-Friendly)

### Context

The A/B experiment system in gh-aw persists per-experiment run counters so that variant selection remains balanced across workflow runs. Previously, this state was stored exclusively in GitHub Actions cache. GitHub Actions cache has a 7-day inactivity eviction policy: if no workflow runs occur for a week (e.g., over a holiday period), the cache entry is silently deleted and the run counter resets to zero. Because A/B experiments may need weeks of accumulated run data to reach statistically meaningful conclusions, this silent data loss undermines the purpose of the feature. Additionally, there was no single storage backend available to all callers — the existing cache approach assumed cache availability, making it impossible to opt out without forking the generated workflow.

### Decision

We will add a `storage` key to the `experiments:` frontmatter that controls where experiment state is persisted, defaulting to `repo` (git branch) and allowing `cache` (legacy behaviour) as an explicit opt-in. The `repo` mode commits `state.json` and `assignments.json` to a dedicated orphan git branch named `experiments/{workflowID}` after each run, using the GitHub GraphQL `createCommitOnBranch` mutation for signed commits with a plain `git push` fallback. Because git branches are permanent (not subject to eviction), `repo` is the safer default for protecting long-running experiment data.

### Alternatives Considered

#### Alternative 1: Keep GitHub Actions Cache as the Only Backend

The existing cache approach required no extra permissions and no new workflow jobs. It was rejected because the 7-day inactivity eviction policy is outside the operator's control and silently resets experiment state, which is a correctness problem for long-running A/B tests that span weekends or holidays.

#### Alternative 2: External Persistent Storage (Database / Object Store)

Storing state in an external database (e.g., a managed key-value store or blob storage) would provide strong durability guarantees without adding git history noise. This option was not chosen because it would require every workflow operator to provision and credential-manage an external service — a significant onboarding burden that contradicts gh-aw's goal of zero-infrastructure experiments.

#### Alternative 3: Workflow Artifact as the Sole Storage

Workflow artifacts are already uploaded per run (retained 30 days). Using artifacts alone as state storage was considered but rejected because artifact retention is also time-bounded, artifact names must be unique per run, and reading the most-recent artifact would require a non-trivial search across run history rather than a single known ref.

### Consequences

#### Positive
- Experiment run counters survive cache evictions, including multi-week inactivity gaps (holidays, pauses).
- The git branch provides a transparent audit trail: every state transition is a commit visible in the repository.
- `cache` mode remains available as an explicit opt-in for workflows that cannot or should not be granted `contents: write`.

#### Negative
- `repo` mode requires `contents: write` permission on the workflow, which is a broader permission than the cache approach needed.
- Every workflow using `repo` storage gains an additional `push_experiments_state` job, increasing per-run job count and slightly increasing workflow cost and latency.
- The experiments git branch grows monotonically over time; operators must manually prune it if storage size becomes a concern.

#### Neutral
- Generated lock files for all workflows with experiments are regenerated to switch from `actions/cache/restore` + `actions/cache/save` steps to the new `load_experiment_state_from_repo.cjs` / `push_experiment_state.cjs` scripts.
- The `storage` key is a reserved word in the experiments frontmatter and is excluded from the experiment config map to avoid being treated as an experiment name.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Experiment State Storage Configuration

1. The `experiments:` frontmatter **MUST** support a `storage` key whose value is one of the enum `["repo", "cache"]`.
2. When `storage` is absent, implementations **MUST** behave as if `storage: repo` was specified.
3. Implementations **MUST NOT** treat the `storage` key as an experiment name; it **MUST** be excluded from experiment config extraction.
4. The JSON schema for the experiments frontmatter **MUST** declare `storage` as an optional property with an enum of `["repo", "cache"]` and a default of `"repo"`.

### `repo` Storage Mode

1. When `storage: repo` is active, the activation job **MUST** load experiment state by fetching `state.json` from the git branch named `experiments/{sanitizedWorkflowID}` via the GitHub API.
2. On a first run (404 response), implementations **MUST** fall back to an empty state object rather than failing.
3. After the activation job completes, a dedicated `push_experiments_state` job **MUST** be generated that downloads the experiment artifact and commits the updated state to the experiments git branch.
4. The `push_experiments_state` job **MUST** declare `permissions: contents: write`.
5. The commit **SHOULD** be made via the GitHub GraphQL `createCommitOnBranch` mutation (producing a verified, signed commit); a plain `git push` **MAY** be used as a fallback when the GraphQL mutation is unavailable.
6. The `push_experiments_state` job **MUST** be listed as a dependency of the conclusion job so that state is persisted before the workflow terminates.
7. Retry logic with exponential backoff **SHOULD** be implemented for the push step to handle transient API failures.

### `cache` Storage Mode

1. When `storage: cache` is explicitly set, the activation job **MUST** restore experiment state from GitHub Actions cache using the workflow-specific cache key.
2. When `storage: cache` is explicitly set, the activation job **MUST** save experiment state back to cache after variant selection.
3. When `storage: cache` is active, no `push_experiments_state` job **SHALL** be generated.
4. Implementations **MUST NOT** require `contents: write` permission when `storage: cache` is configured.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Specifically: (a) `repo` is the default storage mode, (b) `repo` mode generates the `push_experiments_state` job with `contents: write`, (c) `cache` mode preserves legacy cache-only behaviour with no additional job or permission, and (d) the `storage` key is not treated as an experiment name. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25288924793) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
