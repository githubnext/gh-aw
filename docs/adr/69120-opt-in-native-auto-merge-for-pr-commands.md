# ADR-69120: Opt-In GitHub-Native Auto-Merge for All PR-Creating CLI Commands

**Date**: 2026-02-23
**Status**: Draft
**Deciders**: pelikhan

---

### Context

Several `gh aw` commands create pull requests on the user's behalf: `add`, `add-wizard`, `init`, `update`, `upgrade`, `deploy`, and `pr transfer`. Some of these (notably `update`, `upgrade`, and `deploy`) can run against a target repository or across an entire organization, producing many near-identical maintenance PRs that a human then has to merge one by one. Until now the CLI stopped at PR creation, with no supported way to say "merge this once CI is green". The repository already had an unrelated helper (`AutoMergePullRequestsCreatedAfter`) used by safe outputs, so a convention for talking to GitHub's merge APIs existed, but nothing was wired into the interactive CLI surface. Any solution had to stay conservative: it must never merge code that has not passed required checks and reviews, and it must not change behaviour for users who do not ask for it.

### Decision

We will add an opt-in `--auto-merge` boolean flag to every PR-creating command, implemented once in a shared helper `pkg/cli/pr_automerge.go` (`enablePRAutoMerge` / `configurePRAutoMerge`). The helper uses GitHub's GraphQL `enablePullRequestAutoMerge` mutation rather than merging directly, so GitHub — not the CLI — decides when the PR actually merges, and all branch protection rules continue to apply. The merge method is resolved from the repository: `viewerDefaultMergeMethod` when it is allowed, otherwise the first allowed method among `MERGE`, `SQUASH`, `REBASE`. The flag is rejected unless `--pr` or `--create-pull-request` is also given (commands that always create a PR accept those as compatibility aliases), making the user's intent explicit at the call site. Auto-merge activation is treated as a best-effort post-creation step: if it fails, the created PR and its URL are preserved and the failure is reported as an error message that names the PR, rather than rolling anything back.

### Alternatives Considered

#### Alternative 1: Merge directly via `gh pr merge` / the REST merge endpoint

The CLI could poll for checks and then issue a merge itself. This would work on repositories where the auto-merge feature is disabled, which the chosen approach cannot support. It was rejected because it puts merge-gating logic in the client: the CLI would have to replicate required-checks and required-review semantics, and a bug there could merge unreviewed code. Delegating to `enablePullRequestAutoMerge` keeps GitHub as the single authority on merge eligibility.

#### Alternative 2: Make auto-merge a persistent configuration setting instead of a per-invocation flag

Auto-merge could be enabled once in repository or user configuration and applied implicitly to every PR the CLI creates. This would be more convenient for org-wide maintenance flows, where the same choice is made repeatedly. It was rejected because implicit auto-merge is a surprising default for a tool that writes to many repositories; a per-invocation flag keeps the blast radius visible in the command the user actually typed, and the `--pr` requirement makes it impossible to enable by accident.

#### Alternative 3: Reuse the existing `AutoMergePullRequestsCreatedAfter` safe-output helper

The repository already had a helper that scans for recently created PRs and merges them. Reusing it would have avoided a new file. It was rejected because that helper is time-window based and discovers PRs indirectly, which is unsafe for an interactive CLI that already knows the exact PR URL it just created, and it does not express the "schedule, don't merge" semantics we want.

### Consequences

#### Positive
- Organization-wide maintenance flows (`update --org`, `upgrade`, `deploy --org`) can complete without a human merging each PR by hand.
- A single shared activation path means merge-method selection, error wording, and the "never bypass checks" guarantee are implemented once and inherited by every command.
- Behaviour is unchanged for all existing invocations: without `--auto-merge` nothing new happens, and the flag is a hard validation error without `--pr`/`--create-pull-request`.
- Failure to enable auto-merge never destroys work — the PR stays open and its URL is still reported.

#### Negative
- The feature silently depends on repository settings and token scopes: on repositories where auto-merge is disabled, or with a token lacking permission, the command surfaces an error after the PR already exists, which is a partially-successful outcome users must interpret.
- Automatic merge-method selection (`MERGE` → `SQUASH` → `REBASE` fallback) can pick a method the author would not have chosen, and there is no `--auto-merge-method` escape hatch.
- The flag, its validation, and its plumbing now have to be repeated in seven command definitions, so a newly added PR-creating command can easily forget it; roughly 1,200 lines were added across 28 files, over half of it tests covering that repetition.
- Combining auto-merge with bulk org operations increases the consequences of a bad workflow update: many repositories can merge changes without further human review.

#### Neutral
- `add-wizard` remains interactive; the flag only selects behaviour for the PR it eventually creates.
- `--create-pull-request` becomes an accepted no-op alias on commands that always create a PR (`deploy`, `pr transfer`), purely so `--auto-merge` has a consistent opt-in partner.
- GraphQL calls go through `workflow.RunGHContext` via an overridable package variable (`autoMergeRunGH`), which is what makes the new tests possible without network access.
- Command help, the CLI reference (`docs/src/content/docs/setup/cli.md`), and examples were updated in the same change.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
