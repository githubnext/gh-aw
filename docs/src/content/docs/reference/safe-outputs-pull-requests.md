---
title: Safe Outputs (Pull Requests)
description: Reference for create-pull-request and push-to-pull-request-branch safe outputs, including protected files policy.
sidebar:
  order: 801
---

This page covers the two safe-output types that write code to a repository: [`create-pull-request`](#pull-request-creation-create-pull-request) and [`push-to-pull-request-branch`](#push-to-pr-branch-push-to-pull-request-branch). Both types enforce [Protected Files](#protected-files) by default.

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
    draft: true                   # create as draft (default: true)
    max: 3                        # max PRs per run (default: 1)
    expires: 14                   # auto-close after 14 days (same-repo only)
    if-no-changes: "warn"         # "warn" (default), "error", or "ignore"
    target-repo: "owner/repo"     # cross-repository
    allowed-repos: ["org/repo1", "org/repo2"]  # additional allowed repositories
    base-branch: "vnext"          # target branch for PR (default: github.base_ref || github.ref_name)
    fallback-as-issue: false      # disable issue fallback (default: true)
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
    github-token-for-extra-empty-commit: ${{ secrets.CI_TOKEN }} # optional token to push empty commit triggering CI
    protected-files: fallback-to-issue  # push branch, create review issue if protected files modified
```

The `base-branch` field specifies which branch the pull request should target. This is particularly useful for cross-repository PRs where you need to target non-default branches (e.g., `vnext`, `release/v1.0`, `staging`). When not specified, defaults to `github.base_ref` (the PR's target branch) with a fallback to `github.ref_name` (the workflow's branch) for push events.

**Example use case:** A workflow in `org/engineering` that creates PRs in `org/docs` targeting the `vnext` branch for feature documentation:

```yaml wrap
safe-outputs:
  create-pull-request:
    target-repo: "org/docs"
    base-branch: "vnext"
    draft: true
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

PR creation may fail if "Allow GitHub Actions to create and approve pull requests" is disabled in Organization Settings. By default (`fallback-as-issue: true`), fallback creates an issue with branch link and requires `issues: write` permission. Set `fallback-as-issue: false` to disable fallback and only require `contents: write` + `pull-requests: write`.

When `create-pull-request` is configured, git commands (`checkout`, `branch`, `switch`, `add`, `rm`, `commit`, `merge`) are automatically enabled.

By default, PRs created with GitHub Agentic Workflows do not trigger CI. See [Triggering CI](/gh-aw/reference/triggering-ci/) for how to configure CI triggers.

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
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
    github-token-for-extra-empty-commit: ${{ secrets.CI_TOKEN }} # optional token to push empty commit triggering CI
    protected-files: fallback-to-issue  # create review issue if protected files modified
```

When `push-to-pull-request-branch` is configured, git commands (`checkout`, `branch`, `switch`, `add`, `rm`, `commit`, `merge`) are automatically enabled.

Like `create-pull-request`, pushes with GitHub Agentic Workflows do not trigger CI. See [Triggering CI](/gh-aw/reference/triggering-ci/) for how to enable automatic CI triggers.

### Fail-Fast on Code Push Failure

If `push-to-pull-request-branch` (or `create-pull-request`) fails, the safe-output pipeline cancels all remaining non-code-push outputs. Each cancelled output is marked with an explicit reason such as "Cancelled: code push operation failed". The failure details appear in the agent failure issue or comment generated by the conclusion job.

## Protected Files

Both `create-pull-request` and `push-to-pull-request-branch` enforce protected file protection by default. Patches that modify package manifests, agent instruction files, or repository security configuration are refused unless you explicitly configure a policy.

This protects against supply chain attacks where an AI agent could inadvertently (or through prompt injection) alter dependency definitions, CI/CD pipelines, or agent behaviour files.

### Policy Options

Configure the `protected-files` field on either safe output:

| Value | Behaviour |
|-------|-----------|
| `blocked` (default) | Hard-block: the safe output fails with an error |
| `fallback-to-issue` | Create a review issue with instructions for the human to apply or reject the changes manually |
| `allowed` | No restriction — all protected file changes are permitted. **Use only when the workflow is explicitly designed to manage these files.** |

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

### Exempting Specific Files with `allowed-files`

When a workflow is designed to modify only specific files, use `allowed-files` to define a strict allowlist. When set, every file touched by the patch must match at least one pattern — any file outside the list is refused. The `allowed-files` and `protected-files` checks are **orthogonal**: both run independently and both must pass. To modify a protected file, it must both match `allowed-files` **and** `protected-files` must be set to `allowed`.

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
> When `allowed-files` is set, it acts as a strict scope filter: only files matching the patterns may be modified, and any file outside the list is always refused. Files that *do* match are still subject to the `protected-files` policy, which runs independently. To modify a protected file, it must both match `allowed-files` **and** `protected-files` must be set to `allowed`. When `allowed-files` is not set, only the `protected-files` policy applies.

> [!WARNING]
> `allowed-files` should enumerate exactly the files the workflow legitimately manages. Overly broad patterns (e.g., `**`) disable all protection.

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

- `.github/` — covers all GitHub Actions workflows, CODEOWNERS, Dependabot config, and other repository-level security settings.
- `.agents/` — covers generic agent instruction and configuration files stored in the `.agents/` directory.

> [!NOTE]
> Runtime manifests are matched by **basename only** (the filename without its directory path), so `src/package.json`, `frontend/package.json`, and `package.json` at the root are all protected. Path-prefix rules (`.github/`, `.agents/`, `.claude/`, `.codex/`) match the full relative path from the repository root.
