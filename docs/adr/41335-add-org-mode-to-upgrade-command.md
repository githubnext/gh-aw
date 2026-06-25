# ADR-41335: Add organization-wide mode to the `upgrade` command

**Date**: 2026-06-25
**Status**: Draft
**Deciders**: Unknown (auto-generated from PR #41335)

---

### Context

The `gh aw update` command already supports an organization-wide mode that discovers every repository in an org containing source-managed agentic workflows and operates on each one. The `gh aw upgrade` command — which applies codemods, bumps GitHub Actions versions, and recompiles workflows — had no equivalent: it could only run against the current repository. Org administrators who wanted to roll upgrades out across many repositories had to clone and run the command manually, one repository at a time. The org-iteration machinery already exists in `update` (`shallowCloneTargetRepo`, `ensureUpdateTargetRepoGitignore`, `sanitizeRepoPath`, the rate-limit wait, and the code-search pagination loop), so the problem is how to bring that capability to `upgrade` without duplicating that logic.

### Decision

We will add `--org`, `--repos`, and `--create-issue` flags to `upgrade` and implement org mode in a new `pkg/cli/upgrade_org.go` by **reusing the existing `update` org infrastructure** rather than duplicating it. To enable sharing, the pagination loop inside `searchOrgWorkflowRepos` is extracted into `searchOrgReposByQuery(ctx, query, verbose)`; `upgrade` calls it through a new `searchOrgAnyWorkflowRepos` that drops the `"source:"` qualifier (upgrade applies to all agentic workflows, not only source-managed ones). Org mode exposes three behaviors driven by flags: a default dry-run preview listing the repos that would be touched, per-repo PR creation (`--create-pull-request`), and per-repo issue creation (`--create-issue`). Per-repo work reuses `shallowCloneTargetRepo`, `ensureUpdateTargetRepoGitignore`, and `sanitizeRepoPath`, runs `runUpgradeCommand` with `skipExtensionUpgrade: true`, and performs the preflight check per repository (mirroring the `update` pattern) rather than once at the command level.

### Alternatives Considered

#### Alternative 1: Duplicate the `update` org logic inside `upgrade`

Copy the org search, clone, gitignore, and rate-limit code from the update path into the upgrade path. This avoids touching shared code (lower blast radius on `update`) but creates two near-identical copies of fiddly pagination and checkout logic that would drift over time. Rejected: the small refactor to extract `searchOrgReposByQuery` keeps a single source of truth and is covered by the existing update tests plus the new upgrade tests.

#### Alternative 2: A generic org-iteration framework shared by both commands

Introduce a single abstraction (e.g. a `forEachOrgRepo` higher-order function taking a per-repo callback) that both `update` and `upgrade` plug into. This maximizes reuse but over-generalizes before a third caller exists, and the two commands differ in meaningful ways (search qualifier, issue-vs-PR output, preflight placement). Rejected as premature abstraction; extracting just the search helper captures the duplication that actually hurts today.

#### Alternative 3: Issue-only org mode (no in-tool PR creation)

Have org mode only open tracking issues asking maintainers to run `gh aw upgrade` locally, never cloning or opening PRs itself. This is simpler and lower-risk but fails the primary use case of bulk-applying upgrades. Rejected as the default; the issue path is retained as the opt-in `--create-issue` mode for orgs that prefer maintainer-driven rollout.

### Consequences

#### Positive

- Brings org-wide rollout to `upgrade`, matching `update` and giving administrators dry-run, bulk-PR, and bulk-issue rollout strategies from one command.
- Avoids code duplication: the shared `searchOrgReposByQuery` is the single pagination/dedup implementation for both commands, and per-repo checkout reuses the existing update helpers.
- The new code is structured around package-level function variables (`runUpgradeForTargetRepoFn`, `searchOrgAnyWorkflowReposFn`, `createIssueForUpgradeOrgRepoFn`), enabling the 9 unit tests in `upgrade_org_test.go` to exercise dry-run, PR, issue, filtering, and validation paths without network access.

#### Negative

- Widens the `upgrade` command's flag surface (`--org`, `--repos`, `--create-issue`) and adds flag-combination rules (`--create-issue` requires `--org`; `--create-issue` and `--create-pull-request` are mutually exclusive) that users must learn and that must stay consistent with `update`.
- Per-repo PR creation clones and recompiles each matched repository sequentially, which is slow and rate-limit sensitive for large orgs; failures abort the remaining repos rather than continuing.
- The function-variable indirection used for testability adds a layer of mutable package state that obscures the static call graph compared to direct function calls.

#### Neutral

- The org search drops the `"source:"` qualifier, so `upgrade --org` targets a broader repository set than `update --org`; this is intentional but means the two commands can act on different repo lists for the same org.
- Org mode skips the command-level preflight check and performs it per repository inside `runUpgradeForTargetRepo`, deliberately diverging from the single-repo path's placement to match the update pattern.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/28145244604) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
