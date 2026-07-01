---
title: Safe Outputs (Pull Requests)
description: Reference for pull-request safe outputs including create-pull-request, push-to-pull-request-branch, and add-reviewer.
sidebar:
  order: 801
---

This page is the primary reference for pull-request-focused safe outputs:

- [`create-pull-request`](#pull-request-creation-create-pull-request)
- [`update-pull-request`](#pull-request-updates-update-pull-request)
- [`close-pull-request`](#close-pull-request-close-pull-request)
- [`merge-pull-request`](#merge-pull-request-merge-pull-request) (experimental)
- [`create-pull-request-review-comment`](#pr-review-comments-create-pull-request-review-comment)
- [`submit-pull-request-review`](#submit-pr-review-submit-pull-request-review)
- [`reply-to-pull-request-review-comment`](#reply-to-pr-review-comment-reply-to-pull-request-review-comment)
- [`resolve-pull-request-review-thread`](#resolve-pr-review-thread-resolve-pull-request-review-thread)
- [`push-to-pull-request-branch`](#push-to-pr-branch-push-to-pull-request-branch)
- [`add-reviewer`](#add-reviewer-add-reviewer)

Code-writing types (`create-pull-request` and `push-to-pull-request-branch`) enforce [Protected Files](#protected-files) by default.

For all other safe-output types see [Safe Outputs](/gh-aw/reference/safe-outputs/).

## Pull Request Creation (`create-pull-request:`)

Creates a PR with the agent's code changes. Falls back to opening an issue if PR creation is blocked (e.g. org settings) — set `fallback-as-issue: false` to disable. Set `max` above `1` to allow multiple independent PRs per run.

```yaml wrap
safe-outputs:
  create-pull-request:
    title-prefix: "[ai] "         # prefix for titles
    labels: [automation]          # labels to attach
    reviewers: [user1, copilot]   # reviewers (use 'copilot' for bot)
    team-reviewers: [platform-reviewers] # team slugs to request as reviewers
    assignees: [user1]            # assignees for the created PR and any fallback issue
    draft: true                   # create as draft — enforced as policy (default: true)
    max: 3                        # max PRs per run (default: 1)
    expires: 14                   # auto-close after N days (same-repo only; also accepts 2h, 7d, 2w, 1m, 1y)
    if-no-changes: "warn"         # "warn" (default), "error", or "ignore"
    target-repo: "owner/repo"     # cross-repository target
    allowed-repos: ["org/repo1", "org/repo2"]  # additional allowed repositories
    base-branch: "vnext"          # PR target branch (default: github.base_ref || github.ref_name)
    allowed-base-branches:        # allow agent to override base branch at runtime (glob patterns)
      - main
      - release/*
    allowed-branches:             # restrict agent-selected source branch names (glob patterns)
      - feature/*
      - release/*
    fallback-as-issue: false      # disable issue fallback (default: true)
    auto-close-issue: false       # don't auto-add "Fixes #N" to PR description (default: true)
    normalize-closing-keywords: true # strip backticks around recognized issue-closing keywords in PR body text
    preserve-branch-name: true    # omit random salt suffix from branch name (default: false)
    recreate-ref: true            # force-recreate remote branch when it already exists (requires preserve-branch-name; default: false)
    excluded-files:               # strip these files from the patch entirely
      - "**/*.lock"
      - "dist/**"
    max-patch-files: 300          # max unique files in the patch (default: 100)
    max-patch-size: 2048          # max patch size in KB (default: 4096)
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
    github-token-for-extra-empty-commit: ${{ secrets.CI_TOKEN }} # optional token to push empty commit triggering CI
    signed-commits: true          # signed commits via GraphQL API (default: true); set false to use git push directly
    protected-files: fallback-to-issue  # push branch, create review issue if protected files modified
```

See [Cross-Repository Operations](/gh-aw/reference/cross-repository/) for `target-repo`, `allowed-repos`, and authentication configuration.

### Branch targeting

`base-branch` sets the PR's target branch. Defaults to `github.base_ref` (PR event) or `github.ref_name` (push event). Use `allowed-base-branches` to let the agent pick the target branch at runtime — the agent supplies a `base` value in the tool call and it is accepted only if it matches one of the configured glob patterns.

`allowed-branches` restricts which _source_ branch names the agent may use. The effective branch (agent-provided, or the checkout branch as fallback) must match a configured glob.

### Runtime reviewers and assignees

`reviewers`, `team-reviewers`, and `assignees` accept either a static list or a single GitHub Actions expression string. This lets you route a cross-repository PR back to the triggering actor or a runtime-selected team without recompiling the workflow.

```yaml wrap
safe-outputs:
  create-pull-request:
    target-repo: "acme/shared-infra"
    reviewers: ${{ github.event.pull_request.user.login }}
    team-reviewers: ${{ inputs.review-team }}
    assignees: ${{ github.event.pull_request.user.login }}
```

### Branch naming

By default a random hex suffix is appended to the agent-provided branch name to avoid collisions. Set `preserve-branch-name: true` to omit the suffix (useful for repositories that enforce naming conventions such as Jira keys). If `preserve-branch-name: true` and the branch already exists on the remote, use `recreate-ref: true` to force-delete and recreate it (force-push semantics, intended for long-lived branches whose previous PR was already merged).

### Patch limits

`excluded-files` strips matching files from the patch before the commit is created — they are also exempt from `allowed-files` and `protected-files` checks. `max-patch-files` (default `100`) and `max-patch-size` (default `4096 KB`) guard against unexpectedly large commits; raise them when the workflow intentionally produces many or large generated files.

### Other notes

- `draft` is a **policy**, not a default — the agent cannot override it at runtime.
- `auto-close-issue` (default `true`) appends `Fixes #N` to the PR description when the workflow is triggered from an issue. Set to `false` for partial-work or multi-PR flows.
- `normalize-closing-keywords` strips wrapping backticks from recognized issue-closing keywords in the PR body (for example, `` `Closes #123` `` → `Closes #123`).
- When `create-pull-request` is configured, git commands (`checkout`, `branch`, `switch`, `add`, `rm`, `commit`, `merge`) are automatically enabled.
- PRs do not trigger CI by default. See [Triggering CI](/gh-aw/reference/triggering-ci/).
- `create-pull-request` can be disabled at runtime without recompiling by setting the `GH_AW_POLICY_ALLOW_CREATE_PULL_REQUEST` GitHub Actions variable to `"false"` at repository, organization, or enterprise scope. See [Governance](/gh-aw/guides/governance/#disabling-create-pull-request-org-wide).

### How it works

The agent's commits are packaged as a **git bundle** and uploaded as an Actions artifact. A separate, permission-controlled `safe_outputs` job then:

1. Checks out the target repository at the base branch (shallow, depth 1). Any additional `fetch:` refs declared in `checkout:` frontmatter for the target repository are fetched so their commits are locally available.
2. Applies the bundle via `git fetch <bundle-file>`. If prerequisite commits are missing (because the base branch advanced while the agent was running), they are fetched from origin by SHA and the bundle fetch retried automatically.
3. Pushes the branch using the GitHub GraphQL API (signed commits) and creates the pull request.

If the base branch advances between agent start and `safe_outputs` apply, the PR is created slightly behind the current base — normal behavior the author can address with a rebase. If a non-fast-forward race occurs during the push itself, the job creates a fallback PR from a temporary branch so no changes are lost.

An older **patch transport** (`git format-patch` / `git am --3way`) is used when bundle data is unavailable. `--3way` resolves cleanly against an updated base when there are no conflicts; if it cannot, the patch is applied at the agent's original base commit and the PR UI shows the conflicts for manual resolution.

:::note[Cross-repo targets]
The `safe_outputs` job always mirrors the agent job's checkout layout. When a `checkout:` entry places a repository in a subdirectory (a `path:` is set), `safe_outputs` checks out **every** repository to the same location the agent used — the workflow repository at the workspace root plus each cross-repo checkout at its `path:` — regardless of whether `target-repo` names a specific repository or the wildcard `"*"`. This lets a specific `target-repo` (and the two-or-more cross-repo case) operate against an identical layout. When the target repository is checked out at the workspace root (no `path:`), it is checked out there in both jobs.
:::

## Pull Request Updates (`update-pull-request:`)

Updates PR title or body. Both fields are enabled by default. The `operation` field controls how body updates are applied: `append` (default), `prepend`, or `replace`.

```yaml wrap
safe-outputs:
  update-pull-request:
    title: true               # enable title updates (default: true)
    body: true                # enable body updates (default: true)
    update-branch: false      # update PR branch with latest base before other updates (default: false)
    footer: false             # omit AI-generated footer from body updates (default: true)
    max: 1                    # max updates (default: 1)
    target: "*"               # "triggering" (default), "*", or number
    target-repo: "owner/repo" # cross-repository
    required-labels: [automated]     # only update if PR has ALL these labels
    required-title-prefix: "[bot] "  # only update if PR title starts with this prefix
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

**Target**: `"triggering"` (requires PR event), `"*"` (any PR), or number (specific PR).

When `update-branch: true` is set, the handler calls the GitHub REST `pulls.updateBranch` API to merge the latest base branch changes into the PR branch before applying title or body updates. This requires `contents: write` permission; without it only `contents: read` is needed. The field can also be used alone (with `title: false` and `body: false`) to update the branch without changing the PR description.

If GitHub reports `There are no new commits on the base branch.` or `merge conflict between base and head`, the branch update is treated as best-effort: the workflow logs a warning and continues processing the safe output.

When using `target: "*"`, the agent must provide `pull_request_number` in the output to identify which pull request to update.

**Operation Types**: Same as `update-issue` (`append`, `prepend`, `replace`). Title updates always replace the existing title. Disable fields by setting to `false`.

## Close Pull Request (`close-pull-request:`)

Closes PRs without merging with optional comment. Filter by labels and title prefix. Target: `"triggering"` (PR event), `"*"` (any), or number.

```yaml wrap
safe-outputs:
  close-pull-request:
    target: "triggering"              # "triggering" (default), "*", or number
    required-labels: [automated, stale] # only close with these labels
    required-title-prefix: "[bot]"    # only close matching prefix
    max: 10                           # max closures (default: 1)
    target-repo: "owner/repo"         # cross-repository
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

## Merge Pull Request (`merge-pull-request:`)

:::caution[Experimental]
`merge-pull-request` is an experimental safe output. `gh aw compile` emits an experimental feature warning when a workflow uses it. The merge is blocked unless every configured policy gate passes; merges to the repository default branch are always refused.
:::

Merges a pull request only after configured policy gates pass — status checks, review decision, unresolved review threads, label and branch constraints, and GitHub mergeability.

```yaml wrap
safe-outputs:
  merge-pull-request:
    max: 1                            # max merges per run (default: 1, range: 1-10)
    required-labels: [ready-to-merge] # ALL listed labels must be present on the PR
    required-title-prefix: "[bot] "   # only merge PRs whose title starts with this prefix
    allowed-branches: ["feature/*"]   # glob patterns for the PR's source branch
    target: "triggering"              # "triggering" (default, current PR) or "*" (any PR with pull_request_number)
    target-repo: "owner/repo"         # cross-repository target
    allowed-repos: ["org/other-repo"] # additional repositories the agent can merge into
    staged: false                     # if true, evaluate gates and emit preview results without performing the merge
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

**Target**: `"triggering"` (requires a PR event) or `"*"` (the agent supplies `pull_request_number` in the tool call). When `target: "*"` is used with cross-repository configuration, the agent may also supply `repo` (in `owner/repo` format); the value must match `target-repo` or appear in `allowed-repos`.

**Merge method**: The agent selects `merge`, `squash`, or `rebase` per tool call. The base branch is taken from the pull request; merges to the repository default branch are refused by this safe output type.

**Gate semantics**: The handler validates mergeability (not draft, no conflicts), required status checks, the GitHub review decision, unresolved review thread gating, `required-labels`, `required-title-prefix`, and `allowed-branches`. If any gate fails, the merge is skipped and the reason is reported. Idempotent: already-merged PRs return success.

**Staged mode**: Setting `staged: true` runs all gate checks and emits a preview result without calling the GitHub merge API. Use this to validate policy in dry-run scenarios.

See [Cross-Repository Operations](/gh-aw/reference/cross-repository/) for `target-repo`, `allowed-repos`, and authentication configuration. See the [Safe Outputs Specification](/gh-aw/specs/safe-outputs-specification/#type-merge_pull_request) for the complete schema and operational semantics.

## PR Review Comments (`create-pull-request-review-comment:`)

Creates review comments on specific code lines in PRs. Supports single-line and multi-line comments.

```yaml wrap
safe-outputs:
  create-pull-request-review-comment:
    max: 3                    # max comments (default: 10)
    side: "RIGHT"             # "LEFT" or "RIGHT" (default: "RIGHT")
    target: "*"               # "triggering" (default), "*", or number
    target-repo: "owner/repo" # cross-repository
    allowed-repos: ["org/repo1", "org/repo2"]  # additional allowed repositories
    footer: "if-body"         # footer control: "always", "none", or "if-body"
    required-labels: [automated]     # only comment if PR has ALL these labels
    required-title-prefix: "[bot] "  # only comment if PR title starts with this prefix
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

When `target: "*"` is configured, the agent must supply `pull_request_number` in each `create_pull_request_review_comment` tool call to identify which PR to comment on — omitting it will cause the comment to fail. For cross-repository scenarios, the agent can also supply `repo` (in `owner/repo` format) to route the comment to a PR in a different repository; the value must match `target-repo` or appear in `allowed-repos`.

## Submit PR Review (`submit-pull-request-review:`)

Submits a consolidated pull request review. Inline comments buffered by `create-pull-request-review-comment` are included automatically.

```yaml wrap
safe-outputs:
  submit-pull-request-review:
    max: 1
    allowed-events: [COMMENT, REQUEST_CHANGES]  # include REQUEST_CHANGES when superseding older blocking reviews
    supersede-older-reviews: true  # dismiss older same-workflow REQUEST_CHANGES reviews after replacement
    target: "triggering"           # or "*", or explicit PR number
    target-repo: "owner/repo"      # cross-repository
    allowed-repos: ["org/repo1"]   # additional allowed repositories
    footer: "always"               # "always", "none", or "if-body"
    required-labels: [automated]   # only submit review if PR has ALL these labels
    required-title-prefix: "[bot] " # only submit review if PR title starts with this prefix
```

Use `allowed-events` to control review decisions (`APPROVE`, `COMMENT`, `REQUEST_CHANGES`). Prefer `allowed-events: [COMMENT]` by default so bot reviews remain informative and non-blocking.

When you intentionally allow `REQUEST_CHANGES`, set `supersede-older-reviews: true` to dismiss older blocking reviews from the same workflow after posting a replacement review. This behavior is best-effort.

When `target: "*"` is configured, the agent must supply `pull_request_number` in each `submit_pull_request_review` tool call to identify which PR to review — omitting it will cause the review to fail. For cross-repository scenarios, the agent can also supply `repo` (in `owner/repo` format) to route the review to a PR in a different repository; the value must match `target-repo` or appear in `allowed-repos`.

## Reply to PR Review Comment (`reply-to-pull-request-review-comment:`)

Replies to existing review comments on pull requests. Use this to respond to reviewer feedback, answer questions, or acknowledge comments. The `comment_id` must be the numeric ID of an existing review comment.

```yaml wrap
safe-outputs:
  reply-to-pull-request-review-comment:
    max: 10                              # max replies (default: 10)
    target: "triggering"                 # "triggering" (default), "*", or number
    target-repo: "owner/repo"            # cross-repository
    allowed-repos: ["org/other-repo"]    # additional allowed repositories
    footer: true                         # add AI-generated footer (default: true)
    required-labels: [automated]         # only reply if PR has ALL these labels
    required-title-prefix: "[bot] "      # only reply if PR title starts with this prefix
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

The `footer` field controls whether AI-generated footers are added to PR review comments:

- `"always"` (default) - Always include footer on review comments
- `"none"` - Never include footer on review comments
- `"if-body"` - Only include footer when the review has a body text

With `footer: "if-body"`, approval reviews without body text appear clean without the AI-generated footer, while reviews with explanatory text still include the footer for attribution.

## Resolve PR Review Thread (`resolve-pull-request-review-thread:`)

Resolves review threads on pull requests. Allows AI agents to mark review conversations as resolved after addressing the feedback. Uses the GitHub GraphQL API with the `resolveReviewThread` mutation.

By default, resolution is scoped to the triggering PR. Use `target`, `target-repo`, and `allowed-repos` for cross-repository thread resolution.

```yaml wrap
safe-outputs:
  resolve-pull-request-review-thread:
    max: 10                              # max threads to resolve (default: 10)
    target: "triggering"                 # "triggering" (default), "*", or number
    target-repo: "owner/repo"            # cross-repository
    allowed-repos: ["org/repo1", "org/repo2"]  # additional allowed repositories
    required-labels: [automated]         # only resolve if PR has ALL these labels
    required-title-prefix: "[bot] "      # only resolve if PR title starts with this prefix
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

:::note[Integration-token limitation]
GitHub can return `Resource not accessible by integration` for `resolveReviewThread` even with `pull-requests: write` on `GITHUB_TOKEN` (for example, bot-authored threads in non-interactive runs).  
When this happens, gh-aw soft-skips the message with a warning and continues processing other safe outputs.

To make resolution reliable, configure `safe-outputs.resolve-pull-request-review-thread.github-token` with a token that can resolve review threads in your repository.
:::

See [Cross-Repository Operations](/gh-aw/reference/cross-repository/) for documentation on `target-repo`, `allowed-repos`, and cross-repository authentication.

**Agent output format:**

```json
{"type": "resolve_pull_request_review_thread", "thread_id": "PRRT_kwDOABCD..."}
```

## Push to PR Branch (`push-to-pull-request-branch:`)

Pushes changes to a PR's branch. Validates via `required-title-prefix` and `required-labels` to ensure only approved PRs receive changes. Multiple pushes per run are supported by setting `max` higher than 1.

:::caution[Fork PRs Not Supported]
This safe output **cannot push to PRs from forks**. Fork PRs will fail early with a clear error message. This is a security restriction—the workflow does not have write access to fork repositories.
:::

```yaml wrap
safe-outputs:
  push-to-pull-request-branch:
    target: "*"                 # "triggering" (default), "*", or number
    required-title-prefix: "[bot] "      # require title prefix
    required-labels: [automated]         # require all labels
    max: 3                      # max pushes per run (default: 1)
    if-no-changes: "warn"       # "warn" (default), "error", or "ignore"
    excluded-files:               # files to omit from the patch entirely
      - "**/*.lock"
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
    github-token-for-extra-empty-commit: ${{ secrets.CI_TOKEN }} # optional token to push empty commit triggering CI
    fallback-as-pull-request: true        # on non-fast-forward failure, create fallback PR to original PR branch (default: true)
    signed-commits: true                  # signed commits are required (default); set false to use git push directly
    ignore-missing-branch-failure: false  # treat deleted/missing branch errors as skipped instead of failed (default: false)
    check-branch-protection: true         # set to false to skip the branch protection pre-flight check (default: true)
    protected-files: fallback-to-issue  # create review issue if protected files modified
    target-repo: "owner/repo"    # cross-repository (target repo must be checked out)
    allowed-repos: ["org/repo1"] # additional allowed repositories
```

When `push-to-pull-request-branch` is configured, git commands (`checkout`, `branch`, `switch`, `add`, `rm`, `commit`, `merge`) are automatically enabled.

### Destination branch

The agent **does not specify the destination branch**. Both the source and
destination branches are derived from the triggering pull request:

- The **source branch** is the branch currently checked out in the agent's
  workspace. The agent must commit its changes onto the PR's head ref before
  calling the tool.
- The **destination branch** is always the triggering pull request's head ref,
  resolved by the apply-time push job via `pulls.get(pull_number).head.ref`.

This eliminates a class of failures where the agent passed a wrong or synthetic
branch name (see [issue #37835](https://github.com/github/gh-aw/issues/37835)).

By default, pushes are replayed through GitHub's signed commit API because `signed-commits: true` means signed commits are required. Set `signed-commits: false` only for repositories that do not require signed commits; this uses direct `git push` and can preserve merge commits that the signed commit API cannot represent. This field is supported by both `create-pull-request` and `push-to-pull-request-branch`.

### Cross-repo usage

`push-to-pull-request-branch` supports pushing to pull requests in a different repository via `target-repo` (and optionally `allowed-repos`). When `target-repo` is set, **the target repository must be checked out into the workflow workspace** using the `checkout:` frontmatter field with a `path:` specified. Use `target-repo: "*"` to let the agent choose the target repository at runtime (the safe_outputs job will check out all `checkout:` repositories into subdirectories automatically).

```yaml wrap
checkout:
  - fetch-depth: 0                           # checkout current (source) repo
  - repository: org/target-repo
    path: ./target-repo                      # must set path for cross-repo checkout
    github-token: ${{ secrets.CROSS_REPO_PAT }}
    fetch: ["refs/pulls/open/*"]             # fetch all open PR branches

safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}
  push-to-pull-request-branch:
    target-repo: "org/target-repo"
    required-title-prefix: "[bot] "
```

The `path:` field is required so the agent knows where the target repository is mounted in the workspace. Without a `path`, the checkout action writes to the root of the workspace and overwrites the source repository, which will cause the workflow to fail.

See [Cross-Repository Operations](/gh-aw/reference/cross-repository/) for a complete example and documentation on `target-repo`, `allowed-repos`, and cross-repository authentication.

Like `create-pull-request`, pushes with GitHub Agentic Workflows do not trigger CI. See [Triggering CI](/gh-aw/reference/triggering-ci/) for how to enable automatic CI triggers.

### Checkout token for git operations

`create-pull-request` and `push-to-pull-request-branch` run their git operations (fetch/push) against a repository that the `safe_outputs` job checks out with credentials persisted in `.git/config`. A **single** token is persisted into that checkout, resolved with this precedence:

1. `create-pull-request.github-token`
2. `push-to-pull-request-branch.github-token`
3. The `safe-outputs.github-app` minted token (when a GitHub App is configured)
4. `safe-outputs.github-token`
5. The default `${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}`

Because only one token can govern the shared checkout, **if you configure both `create-pull-request` and `push-to-pull-request-branch` for the same repository, give them the same token.** If they specify different `github-token` values, the higher-precedence one wins for the checkout, so the other output's git operations run with a token you did not intend. Set the token once at `safe-outputs.github-token` (or `safe-outputs.github-app`) and let both outputs inherit it, or set identical `github-token` values on each.

:::note
This applies to the git checkout used by the handlers' `fetch`/`push`. The GitHub API calls each handler makes still honor that handler's own `github-token` precedence.
:::

## Add Reviewer (`add-reviewer:`)

Adds reviewers to pull requests. Specify `allowed-reviewers` to restrict to specific GitHub usernames and `allowed-team-reviewers` to restrict to specific team slugs.

```yaml wrap
safe-outputs:
  add-reviewer:
    allowed-reviewers: [user1, copilot]  # restrict to specific user/bot reviewers
    allowed-team-reviewers: [platform-reviewers] # restrict to specific team reviewers
    max: 3                       # max reviewers (default: 3)
    target: "*"                  # "triggering" (default), "*", or number
    target-repo: "owner/repo"    # cross-repository
    required-labels: [automated]     # only add reviewer if PR has ALL these labels
    required-title-prefix: "[bot] "  # only add reviewer if PR title starts with this prefix
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

**Target**: `"triggering"` (requires PR event), `"*"` (any PR), or number (specific PR).

Use `allowed-reviewers: [copilot]` to assign the Copilot PR reviewer bot. See [Copilot Cloud Agent](/gh-aw/reference/copilot-cloud-agent/).

## Compile-Time Warnings for `target: "*"`

When `target: "*"` is used, `gh aw compile` emits warnings for two common misconfigurations:

- **Missing wildcard fetch** — no `checkout` block with a wildcard `fetch` pattern (e.g., `fetch: ["*"]`). Without this, the agent cannot access arbitrary PR branches at runtime and will fail with permission-like errors.
- **No constraints** — neither `required-title-prefix` nor `required-labels` is set, which allows pushing to any PR in the repository with no additional gating.

Both warnings are suppressed when the recommended configuration is in place:

```yaml wrap
safe-outputs:
  push-to-pull-request-branch:
    target: "*"
    required-title-prefix: "[bot] "
    required-labels: [automated]
checkout:
  fetch: ["*"]
  fetch-depth: 0
```

### Fail-Fast on Code Push Failure

If `push-to-pull-request-branch` (or `create-pull-request`) fails, the safe-output pipeline cancels all remaining non-code-push outputs. Each cancelled output is marked with an explicit reason such as "Cancelled: code push operation failed". The failure details appear in the agent failure issue or comment generated by the conclusion job.

When `fallback-as-pull-request` is enabled (default), non-fast-forward push failures trigger a fallback pull request that targets the original PR branch. Set `fallback-as-pull-request: false` to disable this fallback behavior.

When `ignore-missing-branch-failure: true` is set, push failures caused by a deleted or missing PR branch return `skipped: true` instead of a hard failure. This is useful when the PR branch may have been deleted before the safe-output job runs (for example, on auto-merged PRs). Without this flag, a missing branch is a terminal error.

When `check-branch-protection: false` is set, the branch protection API pre-flight check is skipped. By default (`true`), the handler calls `GET /repos/{owner}/{repo}/branches/{branch}/protection` before pushing to detect whether the target branch is protected. This API call requires `administration: read`. If the token lacks that permission, the check logs a warning and continues (the GitHub platform still enforces protection at push time). Set `check-branch-protection: false` to suppress the warning and avoid the API call entirely.

## Protected Files

Both `create-pull-request` and `push-to-pull-request-branch` enforce protected file protection by default. Patches that modify package manifests, agent instruction files, repository security configuration, or any top-level directory whose name starts with `.` are refused unless you explicitly configure a policy.

This protects against supply chain attacks where an AI agent could inadvertently (or through prompt injection) alter dependency definitions, CI/CD pipelines, or agent behavior files.

### What Is Protected

The following are always protected regardless of policy (unless explicitly excluded):

- **Package manifests**: `package.json`, `go.mod`, `go.sum`, `Gemfile`, `Pipfile`, `pyproject.toml`, and other runtime lockfiles.
- **Security configuration**: `CODEOWNERS`, `DESIGN.md`.
- **Agent instruction files**: `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, and other engine-specific instruction files.
- **Common top-level documentation**: `README.md`, `CONTRIBUTING.md`, `CHANGELOG.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`. These files are frequently imported by agents as context, so they are protected by default.
- **Specific protected directories**: `.github/`, `.agents/`, `.githooks/`, `.husky/`.
- **Any top-level directory starting with `.`**: for example `.cursor/`, `.vscode/`, `.devcontainer/`, or any other hidden configuration directory at the repository root. This rule catches newly-created dot-directories without requiring an explicit list update.

### Policy Options

The `protected-files` field accepts either a string policy value or an object with a `policy` and an `exclude` list.

**String form** — set a single policy for all protected files:

| Value | Behavior |
|-------|-----------|
| `request_review` (default) | Create the pull request and submit a `REQUEST_CHANGES` review listing the protected files. The agent's work is preserved, and a human reviewer must approve before merge. |
| `blocked` | Hard-block: the safe output fails with an error |
| `fallback-to-issue` | Create a review issue with instructions for the human to apply or reject the changes manually |
| `allowed` | No restriction — all protected file changes are permitted. **Use only when the workflow is explicitly designed to manage these files.** |

**Object form** — set a policy and exclude specific files from the protected set:

```yaml wrap
safe-outputs:
  create-pull-request:
    protected-files:
      policy: fallback-to-issue   # same values as string form (default: request_review)
      exclude:
        - AGENTS.md               # allow the agent to update its own instruction file
        - CHANGELOG.md            # allow the agent to update the changelog
        - .agents/                # allow updates to the .agents/ directory
        - .cursor/                # allow updates to the .cursor/ directory
```

The `exclude` list names files by **basename** (e.g., `AGENTS.md`) or **path prefix** (e.g., `.agents/`) to remove from the default protected set. Dot-folder path prefixes in the `exclude` list (e.g. `.cursor/`) also opt that directory out of the general top-level-dot-folder protection rule. The remaining protected files still enforce the configured policy. This is useful when a workflow is explicitly designed to manage one specific instruction file or configuration directory without disabling all protection.

:::tip[Workflows that update top-level Markdown files]
If your workflow is explicitly designed to modify a root-level Markdown file such as `CHANGELOG.md` or `README.md`, add it to the `exclude` list so the agent can commit the change.

```yaml wrap
safe-outputs:
  create-pull-request:
    protected-files:
      policy: blocked
      exclude:
        - CHANGELOG.md   # this workflow updates the changelog
```
:::

**`create-pull-request` with `fallback-to-issue`**: when protected files are detected, gh-aw skips pushing and creates a review issue with a PR creation intent link, a `[!WARNING]` banner explaining why the fallback was triggered, and instructions to review carefully before creating the PR.

**`push-to-pull-request-branch` with `fallback-to-issue`**: instead of pushing to the PR branch, a review issue is created with the target PR link, patch download/apply instructions, and a review warning.

```yaml wrap
safe-outputs:
  create-pull-request:
    protected-files: fallback-to-issue  # skip push and require human review before PR

  push-to-pull-request-branch:
    protected-files: fallback-to-issue  # create issue instead of pushing when protected files change
```

When protected file protection triggers and is set to `blocked`, the 🛡️ **Protected Files** section appears in the agent failure issue or comment generated by the conclusion job. It includes the blocked operation, the specific files found, and a YAML remediation snippet showing how to configure `protected-files: fallback-to-issue`.

### Parameterizing Policy Fields in Reusable Workflows

Both `protected-files` and `patch-format` accept **GitHub Actions expression strings** so that reusable `workflow_call` workflows can let callers choose the policy without duplicating the workflow file.

```yaml wrap
on:
  workflow_call:
    inputs:
      protected-files-policy:
        type: string
        default: fallback-to-issue
        description: >
          Protected-file policy: 'request_review', 'blocked', 'fallback-to-issue', or 'allowed'.
      patch-format:
        type: string
        default: bundle
        description: Transport format: 'bundle' (default) or 'am'.
---
safe-outputs:
  push-to-pull-request-branch:
    protected-files: ${{ inputs.protected-files-policy }}
    patch-format: ${{ inputs.patch-format }}

  create-pull-request:
    protected-files: ${{ inputs.protected-files-policy }}
    patch-format: ${{ inputs.patch-format }}
```

**Literal values are still validated at compile time.** Expression strings are passed through to the runtime config where they are evaluated by GitHub Actions before the handler runs. If the resolved value is not one of the documented allowed values, the handler fails closed:

- `protected-files`: an unrecognized resolved value is treated as `blocked` (deny — most restrictive).
- `patch-format`: an unrecognized resolved value results in an explicit error before any git operations.

The object form of `protected-files` also accepts an expression for `policy`:

```yaml wrap
safe-outputs:
  create-pull-request:
    protected-files:
      policy: ${{ inputs.protected-files-policy }}
      exclude:
        - AGENTS.md   # always exclude — regardless of policy
```

### Restricting Changes to Specific Files with `allowed-files`

Use `allowed-files` to restrict a safe output to a fixed set of files. When set, it acts as an **exclusive allowlist**: every file touched by the patch must match at least one pattern, and any file outside the list is always refused — including normal source files. The `allowed-files` and `protected-files` checks are **orthogonal**: both run independently and both must pass. To modify a protected file, it must both match `allowed-files` **and** `protected-files` must be set to `allowed`.

> [!CAUTION]
> `allowed-files` is an **exclusive allowlist**, not an "additionally allow" list. Setting `allowed-files: [".github/workflows/*"]` blocks **all other files**, including normal source code like `src/**`. If you want to allow `.github/workflows/*` alongside regular source files, you must list every pattern explicitly:
> ```yaml
> allowed-files:
>   - .github/workflows/*
>   - src/**
> ```
> Files not listed are refused regardless of whether they are normally unprotected.

```yaml wrap
safe-outputs:
  push-to-pull-request-branch:
    allowed-files:
      - .changeset/**      # only changeset files may be pushed

  create-pull-request:
    allowed-files:
      - .github/aw/instructions.md  # only this one file may be modified
```

Patterns support `*` (any characters except `/`) and `**` (any characters including `/`):

| Pattern | Matches |
|---------|---------|
| `go.mod` | Exactly `go.mod` at the repository root (full path comparison) |
| `*.json` | Any JSON file at the root (e.g. `package.json`) |
| `go.*` | `go.mod`, `go.sum`, etc. at the root |
| `.github/**` | All files under `.github/` at any depth |
| `.github/workflows/*.yml` | Only YAML files directly in `.github/workflows/` |
| `**/package.json` | `package.json` at any path depth |

> [!NOTE]
> When `allowed-files` is not set, only the `protected-files` policy applies and all non-protected files are permitted.

### Allowing Workflow File Changes with `allow-workflows`

When `allowed-files` targets `.github/workflows/` paths, pushing to those paths requires the GitHub Actions `workflows` permission. This is a **GitHub App-only permission** — it cannot be granted via `GITHUB_TOKEN`.

Set `allow-workflows: true` on `create-pull-request` or `push-to-pull-request-branch` to add `workflows: write` to the minted GitHub App token. A `safe-outputs.github-app` configuration is required; the compiler will error if `allow-workflows: true` is set without one.

```yaml wrap
safe-outputs:
  github-app:
    client-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
  create-pull-request:
    allow-workflows: true
    allowed-files:
      - ".github/workflows/*.lock.yml"
    protected-files: allowed
```

> [!NOTE]
> `allow-workflows` is intentionally explicit rather than auto-inferred from `allowed-files` patterns. This makes the elevated permission visible and auditable in the workflow source.

### Protected Files

Protection covers three categories:

**1. Runtime dependency manifests** — matched by filename anywhere in the repository:

| Runtime | Protected files |
|---------|----------------|
| Node.js (npm) | `package.json`, `package-lock.json`, `yarn.lock`, `pnpm-lock.yaml`, `npm-shrinkwrap.json` |
| Node.js (Bun) | `package.json`, `bun.lockb`, `bunfig.toml` |
| Deno | `deno.json`, `deno.jsonc`, `deno.lock` |
| Go | `go.mod`, `go.sum` |
| Python (pip/setuptools) | `requirements.txt`, `Pipfile`, `Pipfile.lock`, `pyproject.toml`, `setup.py`, `setup.cfg` |
| Python (uv) | `pyproject.toml`, `uv.lock` |
| Ruby | `Gemfile`, `Gemfile.lock` |
| Java (Maven) | `pom.xml` |
| Java (Gradle) | `build.gradle`, `build.gradle.kts`, `settings.gradle`, `settings.gradle.kts`, `gradle.properties` |
| Elixir | `mix.exs`, `mix.lock` |
| Haskell | `stack.yaml`, `stack.yaml.lock` |
| .NET | `global.json`, `NuGet.Config`, `Directory.Packages.props` |

**2. Engine instruction files** — added automatically based on the active AI engine:

| Engine | Protected files | Protected directories |
|--------|----------------|----------------------|
| Copilot (default) | `AGENTS.md` | — |
| Claude | `CLAUDE.md` | `.claude/` |
| Codex | `AGENTS.md` | `.codex/` |

**3. Repository security configuration** — matched by path prefix:

- `.github/` — covers all GitHub Actions workflows, Dependabot config, and other repository-level security settings.
- `.agents/` — covers generic agent instruction and configuration files stored in the `.agents/` directory.
- `.githooks/` — covers repository-tracked git hook scripts.
- `.husky/` — covers Husky-managed git hook scripts.

**4. Repository governance files** — matched by filename anywhere in the repository:

| File | Description |
|------|-------------|
| `CODEOWNERS` | Governs required code reviewers; valid at the repository root, `.github/`, or `docs/` |
| `DESIGN.md` | Defines persistent design-system guidance for coding agents |

> [!NOTE]
> Runtime manifests and governance files (`CODEOWNERS`, `DESIGN.md`) are matched by **basename only** (the filename without its directory path), so they are protected regardless of where they appear in the repository. Path-prefix rules (`.github/`, `.agents/`, `.githooks/`, `.husky/`, `.claude/`, `.codex/`) match the full relative path from the repository root.
