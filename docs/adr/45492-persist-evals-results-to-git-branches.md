# ADR-45492: Persist Eval Results to Durable Git Branches with Branch-Aware CLI Fallback

**Date**: 2026-07-14
**Status**: Draft
**Deciders**: Unknown

---

### Context

GitHub Actions artifacts are ephemeral: they expire after a configurable retention period (default 90 days, capped at the organization maximum). Once an artifact expires, `audit --evals` and `logs --evals` commands can no longer retrieve eval results for historical workflow runs, making long-term eval trend analysis and post-hoc debugging impossible. The `experiments/*` storage pattern already solves this problem for experiment state by committing files to a durable git branch on every run. The eval pipeline produces a single `evals.jsonl` result file per run that is small enough to be stored the same way without meaningful repository bloat.

### Decision

We will add a built-in `push_evals_state` workflow job that, after every evals run, downloads the `evals.jsonl` artifact and commits it to a durable `evals/{sanitizedWorkflowID}` git branch using the existing `push_experiment_state.cjs` helper (generalized via `GH_AW_STATE_*` environment variables). The CLI `audit` and `logs` commands will transparently fall back to fetching `evals.jsonl` from this branch via the GitHub Contents API when the file is absent locally, so eval results remain accessible long after artifact expiration without requiring any new infrastructure or storage service.

### Alternatives Considered

#### Alternative 1: Extend artifact retention period

GitHub Actions allows retention to be configured up to the organization maximum (typically 400 days). Increasing retention delays the expiration problem without eliminating it, and the maximum is a hard policy ceiling outside our control. It also does not address the case where artifacts were deleted manually or were never uploaded (e.g., a job that failed before the upload step). This alternative was rejected because it does not provide permanent durability and would not unify the evals and experiments persistence patterns.

#### Alternative 2: Store eval results in an external storage service (e.g., S3, database)

An external service (object store, managed database) can provide indefinite retention and is a common pattern for long-lived CI data. However, this approach requires provisioning and managing infrastructure outside the repository, adds operational overhead (IAM policies, secrets rotation, service availability), and introduces a new external dependency into a tool that is currently self-contained within a GitHub repository. Because eval results are small JSONL files, the marginal cost of git storage is near zero, making a purpose-built service disproportionate. This alternative was rejected in favor of reusing the already-deployed git branch persistence mechanism.

#### Alternative 3: Keep evals artifact-only; warn on expiry instead of fetching from a branch

The simplest change: do nothing for persistence, but print a clearer warning when `--evals` is requested and the artifact has expired. This preserves the current behavior, avoids branch proliferation, and requires no new job. It was rejected because it does not solve the underlying problem: users routinely need to access eval results weeks or months after a run, and a warning cannot substitute for the data.

### Consequences

#### Positive
- Eval results survive artifact expiration and are permanently retrievable from the `evals/{id}` git branch, enabling long-term trend analysis.
- No new infrastructure is required: the generalized `push_experiment_state.cjs` helper and the GitHub Contents API are already in use for the `experiments/*` persistence path.
- The CLI fallback is transparent to users; `audit --evals` and `logs --evals` silently hydrate local files from the branch when needed, preserving existing UX.
- Reusing `WorkflowStateBranchName` and the shared env-var interface (`GH_AW_STATE_*`) enforces a consistent naming convention across all state-persistence jobs.

#### Negative
- Every evals-enabled workflow run creates or updates a branch commit in `evals/{id}`, causing long-lived branch proliferation in the repository; repositories with many workflows will accumulate many such branches.
- The `push_experiment_state.cjs` helper is now responsible for two distinct state types (experiments and evals); the `GH_AW_STATE_*` generalization adds complexity and two sets of env-var aliases that must be maintained for backward compatibility.
- The `push_evals_state` job runs after every qualifying evals run, adding one additional job to the workflow graph and consuming runner minutes even when the branch file would be identical to the previous commit (the script short-circuits on no-change, but the job still provisions a runner).

#### Neutral
- The `evals/*` branch namespace mirrors the `experiments/*` namespace; branch naming is now enforced centrally through `WorkflowStateBranchName`, which is a shared utility in the `workflow` package.
- The CLI fallback silently writes a fetched `evals.jsonl` into the local run directory; callers that inspect the directory after a `logs` or `audit` call will find the file even if it was not produced locally.
- Compiler tests were extended to cover `push_evals_state` creation and conclusion-job dependency wiring, matching the coverage pattern already in place for the experiments persistence path.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
