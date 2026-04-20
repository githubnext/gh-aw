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
- [`create-pull-request-review-comment`](#pr-review-comments-create-pull-request-review-comment)
- [`reply-to-pull-request-review-comment`](#reply-to-pr-review-comment-reply-to-pull-request-review-comment)
- [`resolve-pull-request-review-thread`](#resolve-pr-review-thread-resolve-pull-request-review-thread)
- [`push-to-pull-request-branch`](#push-to-pr-branch-push-to-pull-request-branch)
- [`add-reviewer`](#add-reviewer-add-reviewer)

Code-writing types (`create-pull-request` and `push-to-pull-request-branch`) enforce [Protected Files](#protected-files) by default.

For all other safe-output types see [Safe Outputs](/gh-aw/reference/safe-outputs/).

## Pull Request Creation (`create-pull-request:`)

Creates PRs with code changes. By default, falls back to creating an issue if PR creation fails (e.g., org settings block it). Set `fallback-as-issue: false` to disable this fallback and avoid requiring `issues: write` permission. `expires` field (same-repo only) auto-closes after period: integers (days) or `2h`, `7d`, `2w`, `1m`, `1y` (hours < 24 treated as 1 day).

Multiple PRs per run are supported by setting `max` higher than 1. Each PR is created from its own branch with an independent patch, so concurrent calls do not conflict.

```yaml wrap
safe-outputs:
  create-pull-request:
    title-prefix: "[ai] "         # prefix for titles
    labels: [automation]          # labels to attach
    reviewers: [user1, copilot]   # reviewers (use 'copilot' for bot)
    team-reviewers: [platform-reviewers] # team slugs to request as reviewers
    assignees: [user1]            # assignees for fallback issues (including protected-files and PR creation failure fallbacks)
    draft: true                   # create as draft — enforced as policy (default: true)
    max: 3                        # max PRs per run (default: 1)
    expires: 14                   # auto-close after 14 days (same-repo only)
    if-no-changes: "warn"         # "warn" (default), "error", or "ignore"
    target-repo: "owner/repo"     # cross-repository
    allowed-repos: ["org/repo1", "org/repo2"]  # additional allowed repositories
    base-branch: "vnext"          # target branch for PR (default: github.base_ref || github.ref_name)
    allowed-base-branches:        # allow agent to override base branch at runtime (glob patterns)
      - main
      - release/*
    fallback-as-issue: false      # disable issue fallback (default: true)
    auto-close-issue: false       # don't auto-add "Fixes #N" to PR description (default: true)
    preserve-branch-name: true    # omit random salt suffix from branch name (default: false)
    excluded-files:               # files to omit from the patch entirely
      - "**/*.lock"
      - "dist/**"
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
    github-token-for-extra-empty-commit: ${{ secrets.CI_TOKEN }} # optional token to push empty commit triggering CI
    protected-files: fallback-to-issue  # push branch, create review issue if protected files modified
```

The `base-branch` field specifies which branch the pull request should target. This is particularly useful for cross-repository PRs where you need to target non-default branches (e.g., `vnext`, `release/v1.0`, `staging`). When not specified, defaults to `github.base_ref` (the PR's target branch) with a fallback to `github.ref_name` (the workflow's branch) for push events.

The `allowed-base-branches` field enables per-run base branch overrides by the agent at runtime. When configured, the agent may supply a `base` field in the `create_pull_request` tool call to target a branch other than the compiled `base-branch`. The override is accepted only when it matches one of the configured glob patterns (e.g., `main`, `release/*`). Without `allowed-base-branches`, only the compiled `base-branch` is used regardless of what the agent requests. This is useful when agent-computed data (such as a version string or user request) determines the target branch at runtime:

```yaml wrap
safe-outputs:
  create-pull-request:
    base-branch: main
    allowed-base-branches:
      - main
      - release/*
```

**Example use case:** A workflow in `org/engineering` that creates PRs in `org/docs` targeting the `vnext` branch for feature documentation:

```yaml wrap
safe-outputs:
  create-pull-request:
    target-repo: "org/docs"
    base-branch: "vnext"
    draft: true
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

The `excluded-files` field accepts a list of glob patterns. Each matching file is stripped from the patch using `git format-patch`'s `:(exclude)` magic pathspec at generation time, so the file never appears in the commit. Excluded files are also exempt from `allowed-files` and `protected-files` checks. This is useful for suppressing auto-generated or lock files that the agent must not commit (e.g. `**/*.lock`, `dist/**`). Supports `*` (any characters except `/`) and `**` (any characters including `/`).

The `preserve-branch-name` field, when set to `true`, omits the random hex salt suffix that is normally appended to the agent-specified branch name. This is useful when the target repository enforces branch naming conventions such as Jira keys in uppercase (e.g., `bugfix/BR-329-red` instead of `bugfix/br-329-red-cde2a954`). Invalid characters are always replaced for security, and casing is always preserved regardless of this setting. Defaults to `false`.

The `draft` field is a **configuration policy**, not a default. Whatever value is set in the workflow frontmatter is always used — the agent cannot override it at runtime.

By default, when a workflow is triggered from an issue, the `create-pull-request` handler automatically appends `- Fixes #N` to the PR description if no closing keyword is already present. This causes GitHub to auto-close the triggering issue when the PR is merged. Set `auto-close-issue: false` to opt out of this behavior — useful for partial-work PRs, multi-PR workflows, or any case where the PR should reference but not close the issue.

PR creation may fail if "Allow GitHub Actions to create and approve pull requests" is disabled in Organization Settings. By default (`fallback-as-issue: true`), fallback creates an issue with branch link. Set `fallback-as-issue: false` to disable fallback.

When `create-pull-request` is configured, git commands (`checkout`, `branch`, `switch`, `add`, `rm`, `commit`, `merge`) are automatically enabled.

By default, PRs created with GitHub Agentic Workflows do not trigger CI. See [Triggering CI](/gh-aw/reference/triggering-ci/) for how to configure CI triggers.

### How PR creation works

When the coding agent finishes its task, it records the requested changes in a structured output file. A separate, permission-controlled job then reads that output and applies the changes:

1. The agent's commits are exported as a `git format-patch` file covering everything since the original checkout commit.
2. The safe-output job checks out the target repository and fetches the latest state of the base branch.
3. The patch is applied to a new branch using `git am --3way`. The `--3way` flag allows the patch to succeed even when the agent's source repository differs from the target (for example, in cross-repository workflows).
4. The branch is pushed and the GitHub API creates the pull request.

### If the target branch has changed

If commits have been pushed to the base branch after the agent started, two outcomes are possible:

- **No conflicts** — `git am --3way` resolves the patch cleanly against the updated base. The PR is created normally and targets the current head of the base branch.
- **Conflicts** — if `--3way` cannot resolve the conflicts automatically, the safe-output job falls back to applying the patch at the commit the agent originally branched from. The PR is created with the branch based on that earlier commit, and GitHub's pull request UI shows the conflicts for manual resolution.

> [!NOTE]
> The fallback to the original base commit requires that commit to be present in the target repository. In cross-repository scenarios where the agent repository's history is unrelated, only the `--3way` attempt is made and a hard failure is returned if that also fails.

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
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

**Target**: `"triggering"` (requires PR event), `"*"` (any PR), or number (specific PR).

When `update-branch: true` is set, the handler calls the GitHub REST `pulls.updateBranch` API to merge the latest base branch changes into the PR branch before applying title or body updates. This requires `contents: write` permission; without it only `contents: read` is needed. The field can also be used alone (with `title: false` and `body: false`) to update the branch without changing the PR description.

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
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

When `target: "*"` is configured, the agent must supply `pull_request_number` in each `create_pull_request_review_comment` tool call to identify which PR to comment on — omitting it will cause the comment to fail. For cross-repository scenarios, the agent can also supply `repo` (in `owner/repo` format) to route the comment to a PR in a different repository; the value must match `target-repo` or appear in `allowed-repos`.

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
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

See [Cross-Repository Operations](/gh-aw/reference/cross-repository/) for documentation on `target-repo`, `allowed-repos`, and cross-repository authentication.

**Agent output format:**

```json
{"type": "resolve_pull_request_review_thread", "thread_id": "PRRT_kwDOABCD..."}
```

## Push to PR Branch (`push-to-pull-request-branch:`)

Pushes changes to a PR's branch. Validates via `title-prefix` and `labels` to ensure only approved PRs receive changes. Multiple pushes per run are supported by setting `max` higher than 1.

:::caution[Fork PRs Not Supported]
This safe output **cannot push to PRs from forks**. Fork PRs will fail early with a clear error message. This is a security restriction—the workflow does not have write access to fork repositories.
:::

```yaml wrap
safe-outputs:
  push-to-pull-request-branch:
    target: "*"                 # "triggering" (default), "*", or number
    title-prefix: "[bot] "      # require title prefix
    labels: [automated]         # require all labels
    max: 3                      # max pushes per run (default: 1)
    if-no-changes: "warn"       # "warn" (default), "error", or "ignore"
    excluded-files:               # files to omit from the patch entirely
      - "**/*.lock"
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
    github-token-for-extra-empty-commit: ${{ secrets.CI_TOKEN }} # optional token to push empty commit triggering CI
    fallback-as-pull-request: true        # on non-fast-forward failure, create fallback PR to original PR branch (default: true)
    ignore-missing-branch-failure: false  # treat deleted/missing branch errors as skipped instead of failed (default: false)
    protected-files: fallback-to-issue  # create review issue if protected files modified
```

When `push-to-pull-request-branch` is configured, git commands (`checkout`, `branch`, `switch`, `add`, `rm`, `commit`, `merge`) are automatically enabled.

Like `create-pull-request`, pushes with GitHub Agentic Workflows do not trigger CI. See [Triggering CI](/gh-aw/reference/triggering-ci/) for how to enable automatic CI triggers.

## Add Reviewer (`add-reviewer:`)

Adds reviewers to pull requests. Specify `reviewers` to restrict to specific GitHub usernames and `team-reviewers` to restrict to specific team slugs.

```yaml wrap
safe-outputs:
  add-reviewer:
    reviewers: [user1, copilot]  # restrict to specific user/bot reviewers
    team-reviewers: [platform-reviewers] # restrict to specific team reviewers
    max: 3                       # max reviewers (default: 3)
    target: "*"                  # "triggering" (default), "*", or number
    target-repo: "owner/repo"    # cross-repository
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

**Target**: `"triggering"` (requires PR event), `"*"` (any PR), or number (specific PR).

Use `reviewers: [copilot]` to assign the Copilot PR reviewer bot. See [Assign to Agent](/gh-aw/reference/assign-to-copilot/).

## Compile-Time Warnings for `target: "*"`

When `target: "*"` is used, `gh aw compile` emits warnings for two common misconfigurations:

- **Missing wildcard fetch** — no `checkout` block with a wildcard `fetch` pattern (e.g., `fetch: ["*"]`). Without this, the agent cannot access arbitrary PR branches at runtime and will fail with permission-like errors.
- **No constraints** — neither `title-prefix` nor `labels` is set, which allows pushing to any PR in the repository with no additional gating.

Both warnings are suppressed when the recommended configuration is in place:

```yaml wrap
safe-outputs:
  push-to-pull-request-branch:
    target: "*"
    title-prefix: "[bot] "
checkout:
  fetch: ["*"]
  fetch-depth: 0
```

### Fail-Fast on Code Push Failure

If `push-to-pull-request-branch` (or `create-pull-request`) fails, the safe-output pipeline cancels all remaining non-code-push outputs. Each cancelled output is marked with an explicit reason such as "Cancelled: code push operation failed". The failure details appear in the agent failure issue or comment generated by the conclusion job.

When `fallback-as-pull-request` is enabled (default), non-fast-forward push failures trigger a fallback pull request that targets the original PR branch. Set `fallback-as-pull-request: false` to disable this fallback behavior.

When `ignore-missing-branch-failure: true` is set, push failures caused by a deleted or missing PR branch return `skipped: true` instead of a hard failure. This is useful when the PR branch may have been deleted before the safe-output job runs (for example, on auto-merged PRs). Without this flag, a missing branch is a terminal error.

## Protected Files

Both `create-pull-request` and `push-to-pull-request-branch` enforce protected file protection by default. Patches that modify package manifests, agent instruction files, or repository security configuration are refused unless you explicitly configure a policy.

This protects against supply chain attacks where an AI agent could inadvertently (or through prompt injection) alter dependency definitions, CI/CD pipelines, or agent behaviour files.

### Policy Options

The `protected-files` field accepts either a string policy value or an object with a `policy` and an `exclude` list.

**String form** — set a single policy for all protected files:

| Value | Behaviour |
|-------|-----------|
| `blocked` (default) | Hard-block: the safe output fails with an error |
| `fallback-to-issue` | Create a review issue with instructions for the human to apply or reject the changes manually |
| `allowed` | No restriction — all protected file changes are permitted. **Use only when the workflow is explicitly designed to manage these files.** |

**Object form** — set a policy and exclude specific files from the protected set:

```yaml wrap
safe-outputs:
  create-pull-request:
    protected-files:
      policy: fallback-to-issue   # same values as string form (default: blocked)
      exclude:
        - AGENTS.md               # allow the agent to update its own instruction file
        - .agents/                # allow updates to the .agents/ directory
```

The `exclude` list names files by **basename** (e.g., `AGENTS.md`) or **path prefix** (e.g., `.agents/`) to remove from the default protected set. The remaining protected files still enforce the configured policy. This is useful when a workflow is explicitly designed to manage one specific instruction file without disabling all protection.

**`create-pull-request` with `fallback-to-issue`**: the branch is pushed normally, then a review issue is created with a PR creation intent link, a `[!WARNING]` banner explaining why the fallback was triggered, and instructions to review carefully before creating the PR.

**`push-to-pull-request-branch` with `fallback-to-issue`**: instead of pushing to the PR branch, a review issue is created with the target PR link, patch download/apply instructions, and a review warning.

```yaml wrap
safe-outputs:
  create-pull-request:
    protected-files: fallback-to-issue  # push branch, require human review before PR

  push-to-pull-request-branch:
    protected-files: fallback-to-issue  # create issue instead of pushing when protected files change
```

When protected file protection triggers and is set to `blocked`, the 🛡️ **Protected Files** section appears in the agent failure issue or comment generated by the conclusion job. It includes the blocked operation, the specific files found, and a YAML remediation snippet showing how to configure `protected-files: fallback-to-issue`.

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

> [!WARNING]
> `allowed-files` should enumerate exactly the files the workflow legitimately manages. Overly broad patterns (e.g., `**`) disable all protection.

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

**4. Repository access control files** — matched by filename anywhere in the repository:

| File | Description |
|------|-------------|
| `CODEOWNERS` | Governs required code reviewers; valid at the repository root, `.github/`, or `docs/` |

> [!NOTE]
> Runtime manifests and access control files (`CODEOWNERS`) are matched by **basename only** (the filename without its directory path), so they are protected regardless of where they appear in the repository. Path-prefix rules (`.github/`, `.agents/`, `.claude/`, `.codex/`) match the full relative path from the repository root.
