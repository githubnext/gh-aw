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
    draft: true                   # create as draft — enforced as policy (default: true)
    max: 3                        # max PRs per run (default: 1)
    expires: 14                   # auto-close after 14 days (same-repo only)
    if-no-changes: "warn"         # "warn" (default), "error", or "ignore"
    target-repo: "owner/repo"     # cross-repository
    allowed-repos: ["org/repo1", "org/repo2"]  # additional allowed repositories
    base-branch: "vnext"          # target branch for PR (default: github.base_ref || github.ref_name)
    fallback-as-issue: false      # disable issue fallback (default: true)
    preserve-branch-name: true    # omit random salt suffix from branch name (default: false)
    excluded-files:               # files to omit from the patch entirely
      - "**/*.lock"
      - "dist/**"
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

The `excluded-files` field accepts a list of glob patterns. Each matching file is stripped from the patch using `git format-patch`'s `:(exclude)` magic pathspec at generation time, so the file never appears in the commit. Excluded files are also exempt from `allowed-files` and `protected-files` checks. This is useful for suppressing auto-generated or lock files that the agent must not commit (e.g. `**/*.lock`, `dist/**`). Supports `*` (any characters except `/`) and `**` (any characters including `/`).

The `preserve-branch-name` field, when set to `true`, omits the random hex salt suffix that is normally appended to the agent-specified branch name. This is useful when the target repository enforces branch naming conventions such as Jira keys in uppercase (e.g., `bugfix/BR-329-red` instead of `bugfix/br-329-red-cde2a954`). Invalid characters are always replaced for security, and casing is always preserved regardless of this setting. Defaults to `false`.

The `draft` field is a **configuration policy**, not a default. Whatever value is set in the workflow frontmatter is always used — the agent cannot override it at runtime.

PR creation may fail if "Allow GitHub Actions to create and approve pull requests" is disabled in Organization Settings. By default (`fallback-as-issue: true`), fallback creates an issue with branch link. Set `fallback-as-issue: false` to disable fallback.

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
    excluded-files:               # files to omit from the patch entirely
      - "**/*.lock"
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
    github-token-for-extra-empty-commit: ${{ secrets.CI_TOKEN }} # optional token to push empty commit triggering CI
    protected-files: fallback-to-issue  # create review issue if protected files modified
```

When `push-to-pull-request-branch` is configured, git commands (`checkout`, `branch`, `switch`, `add`, `rm`, `commit`, `merge`) are automatically enabled.

Like `create-pull-request`, pushes with GitHub Agentic Workflows do not trigger CI. See [Triggering CI](/gh-aw/reference/triggering-ci/) for how to enable automatic CI triggers.

### Compile-Time Warnings for `target: "*"`

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
