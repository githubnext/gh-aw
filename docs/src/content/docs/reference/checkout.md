---
title: GitHub Repository Checkout
description: Configure how actions/checkout is invoked in the agent job — disable checkout, override settings, check out multiple repositories, fetch additional refs, and mark a primary target repository.
sidebar:
  order: 852
---

The `checkout:` frontmatter field controls how `actions/checkout` is invoked in the agent job. Configure custom checkout settings, check out multiple repositories, or disable checkout entirely.

By default, the agent checks out the repository where the workflow is running with a shallow fetch (`fetch-depth: 1`). If triggered by a pull request event, it also checks out the PR head ref. For most workflows, this default checkout is sufficient and no `checkout:` configuration is necessary.

Use `checkout:` when you need to check out additional branches, check out multiple repositories, or to disable checkout entirely for workflows that don't need to access code or can access code dynamically through the GitHub Tools.

## Custom Checkout Settings

You can use `checkout:` to override default checkout settings (e.g., fetch depth, sparse checkout) without needing to define a custom job:

```yaml wrap
checkout:
  fetch-depth: 0                              # Full git history
  github-token: ${{ secrets.MY_TOKEN }}        # Custom authentication
```

Or use GitHub App authentication:

```yaml wrap
checkout:
  fetch-depth: 0
  github-app:
    app-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
```

You can also use `checkout:` to check out additional repositories alongside the main repository:

```yaml wrap
checkout:
  - fetch-depth: 0
  - repository: owner/other-repo
    path: ./libs/other
    ref: main
    github-token: ${{ secrets.CROSS_REPO_PAT }}
```

## Configuration Options

| Field | Type | Description |
|-------|------|-------------|
| `repository` | string | Repository in `owner/repo` format. Defaults to the current repository. |
| `ref` | string | Branch, tag, or SHA to checkout. Defaults to the triggering ref. |
| `path` | string | Path within `GITHUB_WORKSPACE` to place the checkout. Defaults to workspace root. |
| `github-token` | string | Token for authentication. Use `${{ secrets.MY_TOKEN }}` syntax. |
| `github-app` | object | GitHub App credentials (`app-id`, `private-key`, optional `owner`, `repositories`). Mutually exclusive with `github-token`. `app` is a deprecated alias. |
| `fetch-depth` | integer | Commits to fetch. `0` = full history, `1` = shallow clone (default). |
| `fetch` | string \| string[] | Additional Git refs to fetch after checkout. See [Fetching Additional Refs](#fetching-additional-refs). |
| `sparse-checkout` | string | Newline-separated patterns for sparse checkout (e.g., `.github/\nsrc/`). |
| `submodules` | string/bool | Submodule handling: `"recursive"`, `"true"`, or `"false"`. |
| `lfs` | boolean | Download Git LFS objects. |
| `current` | boolean | Marks this checkout as the primary working repository. The agent uses this as the default target for all GitHub operations. Only one checkout may set `current: true`; the compiler rejects workflows where multiple checkouts enable it. |

## Fetching Additional Refs

By default, `actions/checkout` performs a shallow clone (`fetch-depth: 1`) of a single ref. For workflows that need to work with other branches — for example, a scheduled workflow that must push changes to open pull-request branches — use the `fetch:` option to retrieve additional refs after the checkout step.

A dedicated git fetch step is emitted after the `actions/checkout` step. Authentication re-uses the checkout token (or falls back to `github.token`) via a transient `http.extraheader` credential — no credentials are persisted to disk, consistent with the enforced `persist-credentials: false` policy.

| Value | Description |
|-------|-------------|
| `"*"` | All remote branches. |
| `"refs/pulls/open/*"` | All open pull-request head refs (GH-AW shorthand). |
| `"main"` | A specific branch name. |
| `"feature/*"` | A glob pattern matching branch names. |

```yaml wrap
checkout:
  - fetch: ["*"]                 # fetch all branches (default checkout)
    fetch-depth: 0               # fetch full history to ensure we can see all commits and PR details
```

```yaml wrap
checkout:
  - repository: githubnext/gh-aw-side-repo
    github-token: ${{ secrets.GH_AW_SIDE_REPO_PAT }}
    fetch: ["refs/pulls/open/*"]      # fetch all open PR refs after checkout
    fetch-depth: 0               # fetch full history to ensure we can see all commits and PR details
```

```yaml wrap
checkout:
  - repository: org/target-repo
    github-token: ${{ secrets.CROSS_REPO_PAT }}
    fetch: ["main", "feature/*"] # fetch specific branches
    fetch-depth: 0               # fetch full history to ensure we can see all commits and PR details
```

:::note
If a branch you need is not available after checkout and is not covered by a `fetch:` pattern, and you're in a private or internal repo, then the agent cannot access its Git history except inefficiently, file by file, via the GitHub MCP. For private repositories, it will be unable to fetch or explore additional branches. If the branch is required and unavailable, configure the appropriate pattern in `fetch:` (e.g., `fetch: ["*"]` for all branches, or `fetch: ["refs/pulls/open/*"]` for PR branches) and recompile the workflow.
:::

## Disabling Checkout (`checkout: false`)

Set `checkout: false` to suppress the default `actions/checkout` step entirely. Use this for workflows that access repositories through MCP servers or other mechanisms that do not require a local clone:

```yaml wrap
checkout: false
```

This is equivalent to omitting the checkout step from the agent job. Custom dev-mode steps (such as "Checkout actions folder") are unaffected.

## Marking a Primary Repository (`current: true`)

When a workflow running from a central repository targets a different repository, use `current: true` to tell the agent which repository to treat as its primary working target. The agent uses this as the default for all GitHub operations (creating issues, opening PRs, reading content) unless the prompt instructs otherwise. When omitted, the agent defaults to the repository where the workflow is running.

```yaml wrap
checkout:
  - repository: org/target-repo
    path: ./target
    github-token: ${{ secrets.CROSS_REPO_PAT }}
    current: true                                    # agent's primary target
```

> [!IMPORTANT]
> `current: true` only annotates the agent's system prompt to identify the target repository — it does **not** automatically change the working directory. If the agent needs to run local tools (tests, linters, build scripts) against the checked-out repository, add an explicit `cd` instruction to the prompt:
>
> ```
> Navigate into the folder where the target repository has been checked out into: cd ${{ github.workspace }}/target
> ```
>
> Without this instruction, the agent starts in `$GITHUB_WORKSPACE` (the side repository checkout) and must infer the correct directory on its own.

## Checkout Merging

Multiple `checkout:` configurations can target the same path and repository. This is useful for monorepos where different parts of the repository must be merged into the same workspace directory with different settings (e.g., sparse checkout for some paths, full checkout for others).

When multiple `checkout:` entries target the same repository and path, their configurations are merged with the following rules:

- **Fetch depth**: Deepest value wins (`0` = full history always takes precedence)
- **Fetch refs**: Merged (union of all patterns; duplicates are removed)
- **Sparse patterns**: Merged (union of all patterns)
- **LFS**: OR-ed (if any config enables `lfs`, the merged configuration enables it)
- **Submodules**: First non-empty value wins for each `(repository, path)`; once set, later values are ignored
- **Ref/Token/App**: First-seen wins

## Related Documentation

- [Cross-Repository Operations](/gh-aw/reference/cross-repository/) - Reading and writing across multiple repositories
- [Authentication Reference](/gh-aw/reference/auth/) - PAT and GitHub App setup
- [Multi-Repository Examples](/gh-aw/examples/multi-repo/) - Complete working examples
