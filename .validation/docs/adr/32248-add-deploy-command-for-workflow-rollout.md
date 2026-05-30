# ADR-32248: Add `gh aw deploy` Command for Cross-Repo Workflow Rollout

**Date**: 2026-05-15
**Status**: Draft
**Deciders**: [TODO: verify]

---

## Part 1 — Narrative (Human-Friendly)

### Context

Rolling out agentic workflows to a target repository previously required operators to chain several separate `gh aw` commands manually: clone the target repo, run `update` to refresh sourced workflows, run `add` to install requested workflows, run `compile --purge` to regenerate lock files and remove stale outputs, then open a pull request. This sequence is repetitive, easy to get wrong, and produces inconsistent results across rollouts. The flags users already know from `gh aw add` (engine, name, append, dir, stop-after, security, cooldown) need to apply uniformly across all phases of the rollout. The rollout must be presented to the target repo as a single reviewable PR so changes are auditable before merge.

### Decision

We will introduce a new top-level `deploy <workflow>...` command in `pkg/cli/deploy_command.go` that orchestrates the full clone → update → add → compile (with purge) → create-PR sequence in one invocation. The command reuses existing primitives (`shallowCloneTargetRepo`, `RunUpdateWorkflows`, `AddWorkflows`, `CompileWorkflows`, `CreatePRWithChanges`) rather than duplicating their logic, and accepts the same flag surface as `gh aw add` (plus a required `--repo` target and a `--cool-down` default of `7d`). Existing workflows that already carry a `source:` frontmatter field are detected and excluded from the add phase to avoid duplicate-add errors after the update pass already refreshed them.

### Alternatives Considered

#### Alternative 1: Document a multi-command recipe

We could publish a documented bash script or runbook that chains the existing commands (`clone`, `update`, `add`, `compile --purge`, `pr`). No new CLI surface, but operators still own all error handling, retries, and ordering. Rejected because rollouts are a frequent, multi-step operation where small ordering or flag mistakes cause partial deployments, and the chain has subtle behavior (e.g., skip-add for sourced workflows) that is hard to encode in a runbook.

#### Alternative 2: Add a `--target-repo` flag to `gh aw add`

We could extend `gh aw add` with cross-repo deployment semantics behind a flag. Rejected because it overloads `add` with checkout, update, compile, and PR creation responsibilities, blurring the boundary between "install a workflow locally" and "roll out workflows to a remote repo via PR." The flag-driven mode would also be harder to discover than a dedicated `deploy` command in the `setup` group.

#### Alternative 3: External shell script outside the CLI

We could ship a standalone shell script (or composite GitHub Action) that calls `gh aw` subcommands. Rejected because it loses the CLI's integrated help, flag completion, engine validation, and consistent error reporting, and because it would not have access to internal helpers like `excludeExistingSourcedWorkflows`.

### Consequences

#### Positive
- Single command replaces a 5-step manual sequence, reducing operator error during rollouts.
- Reuses existing primitives (`AddWorkflows`, `RunUpdateWorkflows`, `CompileWorkflows`, `CreatePRWithChanges`) rather than forking logic, so future improvements to those primitives benefit `deploy` automatically.
- Sourced-workflow detection (`existingWorkflowHasSource`) prevents duplicate-add failures and keeps update/add phases idempotent on re-runs.
- `compile --purge` is enforced in the deploy path so stale `.lock.yml` artifacts are removed in every rollout, keeping the target repo clean.

#### Negative
- Adds ~250 lines of new CLI orchestration code plus a parallel flag-wiring block that duplicates the flag surface already declared on `add`. If those flag sets drift, deploy and add could behave differently for the same flag name.
- The command requires running inside a git repository (uses `gitutil.FindGitRoot()` to compute a per-target checkout directory under the source repo's working tree). Operators cannot run `deploy` from an arbitrary directory.
- PR title and body are fixed (`chore: deploy agentic workflows` / templated body) with no customization flag, limiting how downstream PR-routing automation can tag or categorize deploy PRs.

#### Neutral
- The command is registered in the existing `setup` command group, surfacing alongside `add`, `update`, `remove`, and `upgrade` rather than as a standalone top-level concept.
- A new test file `pkg/cli/deploy_command_test.go` covers the command shape, flag registration, repo-requirement enforcement, PR metadata formatting, and the sourced-workflow exclusion helper.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Command Surface

1. The CLI **MUST** expose a `deploy <workflow>...` subcommand under the root command.
2. The `deploy` command **MUST** require at least one workflow specification argument and **MUST** return a clear "missing workflow specification" error when invoked with none.
3. The `deploy` command **MUST** require a non-empty `--repo` flag in `owner/repo` form and **MUST** fail with a descriptive error when it is missing.
4. The `deploy` command **MUST** be assigned to the `setup` command group.
5. The `deploy` command **MUST** register the following flags with semantics matching `gh aw add`: `name`, `engine`, `force`, `append`, `no-gitattributes`, `dir`, `no-stop-after`, `stop-after`, `disable-security-scanner`.
6. The `deploy` command **MUST** register a `--cool-down` flag whose default value is `7d`.
7. The `deploy` command **MUST NOT** accept `--name` together with more than one workflow argument.

### Orchestration Flow

1. The command **MUST** execute the rollout phases in this order: shallow clone of the target repo, `RunUpdateWorkflows`, exclusion of existing sourced workflows, `AddWorkflows` for the remaining specs, `CompileWorkflows` with `Purge: true`, and `CreatePRWithChanges`.
2. The command **MUST** invoke `CompileWorkflows` with `Purge` enabled so stale `.lock.yml` artifacts are removed during rollout.
3. The command **MUST** skip the add phase for any workflow whose existing file in the target repo contains a non-empty `source:` frontmatter field, treating it as already handled by the update phase.
4. The command **MUST** log skipped workflows so operators can see which specs were not re-added.
5. The command **MUST** restore the original working directory before returning, even on error paths.
6. The command **SHOULD** treat malformed frontmatter in an existing workflow file as "not sourced" and proceed with the add phase rather than aborting the rollout.

### PR Metadata

1. The created pull request **MUST** use the commit message `chore: deploy agentic workflows` as its title.
2. The pull request body **MUST** name the target repo and **MUST** indicate that the PR was created by `gh aw deploy` after running update, add, and compile with purge.
3. When more than one workflow is deployed, the PR body **MUST** describe the rollout as "N workflows" rather than listing each spec.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the Design Decision Gate workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
