---
title: QMD Documentation Search
description: Build a local vector search index over documentation files and expose it as an MCP tool so agents can find relevant docs without contents:read permission in the agent job.
sidebar:
  order: 730
---

import { Aside } from "@astrojs/starlight/components";

<Aside type="caution" title="Experimental">
  The `qmd` tool is experimental and its API may change without notice.
</Aside>

The `qmd:` tool integrates [tobi/qmd](https://github.com/tobi/qmd) as a built-in MCP server that performs **vector similarity search** over documentation files. The search index is built in a dedicated `indexing` job (which has `contents: read`) and shared with the agent job via `actions/cache`, so the agent job does not need `contents: read`.

## How it works

1. **Indexing job** — installs `@tobilu/qmd`, registers documentation collections from configured checkouts and/or GitHub searches, builds the vector index, and saves it to `actions/cache`.
2. **Agent job** — restores the qmd cache (index and models) and starts qmd as an MCP server (`qmd mcp --http`). The agent can call the `search` tool to find relevant documentation files by natural language query.

The embedding models used to build and query the index are automatically cached in both jobs via `actions/cache` (keyed by OS at `~/.cache/qmd/models/`), so models are only downloaded once per runner OS.

## Quick start

```aw wrap
---
tools:
  qmd:
    checkouts:
      - name: docs
        paths:
          - docs/**/*.md
          - .github/**/*.md
---
```

This indexes all markdown files under `docs/` and `.github/` in the current repository.

## Configuration

### Checkouts form

Index files from one or more named collections, each with an optional repository checkout:

```yaml wrap
tools:
  qmd:
    checkouts:
      - name: current-docs
        paths:
          - docs/**/*.md
        context: "Project documentation"
      - name: other-repo-docs
        paths:
          - docs/**/*.md
        context: "Documentation for owner/other-repo"
        checkout:
          repository: owner/other-repo
          ref: main
          path: ./other-repo       # optional; defaults to /tmp/gh-aw/qmd-checkout-<name>
```

Each `checkout:` entry accepts the same options as the top-level [`checkout:`](/gh-aw/reference/frontmatter/#repository-checkout-checkout) field: `repository`, `ref`, `path`, `token`, `fetch-depth`, `sparse-checkout`, `submodules`, and `lfs`.

The optional `context:` field provides additional hints to the agent about the collection's content (e.g. product area, audience, or version).

### Searches form

Download files returned by GitHub code search and add them to the index:

```yaml wrap
tools:
  qmd:
    searches:
      - query: "repo:owner/repo language:Markdown path:docs/"
        min: 1     # fail the activation job if fewer results (default: 0)
        max: 30    # download at most this many files (default: 30)
        github-token: ${{ secrets.GITHUB_TOKEN }}
```

Each search entry runs `gh search code <query>` in the activation job, downloads every matching file via the GitHub API, and registers the result as a separate qmd collection named `search-0`, `search-1`, etc.

Use `github-app:` instead of `github-token:` for cross-organization access:

```yaml wrap
tools:
  qmd:
    searches:
      - query: "org:myorg language:Markdown path:docs/"
        github-app:
          app-id: ${{ vars.APP_ID }}
          private-key: ${{ secrets.APP_PRIVATE_KEY }}
```

### Cache key

Persist the index in GitHub Actions cache to speed up subsequent runs. On a cache hit all indexing steps are skipped automatically:

```yaml wrap
tools:
  qmd:
    checkouts:
      - name: docs
        paths: [docs/**/*.md]
    cache-key: "qmd-index-${{ hashFiles('docs/**') }}"
```

#### Read-only mode

When `cache-key` is set without any indexing sources (`checkouts` or `searches`), the tool operates in **read-only mode**: the activation job restores the index from cache (failing silently if the cache does not exist yet) and skips all Node.js, npm, and qmd build steps entirely. This is useful for maintaining a shared, pre-built documentation database:

```yaml wrap
tools:
  qmd:
    cache-key: "qmd-index-v1"
```

### Combined form

All sources can be combined in a single configuration:

```yaml wrap
tools:
  qmd:
    checkouts:
      - name: local-docs
        paths: [docs/**/*.md]
        context: "Project documentation"
      - name: sdk-docs
        paths: [README.md, docs/**/*.md]
        context: "SDK reference"
        checkout:
          repository: owner/sdk
          path: ./sdk
    searches:
      - query: "org:myorg language:Markdown path:wiki/"
        max: 50
        github-token: ${{ secrets.GITHUB_TOKEN }}
    cache-key: "qmd-index-${{ hashFiles('docs/**') }}"
```

## Configuration reference

### `qmd:` fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `checkouts` | `QmdDocCollection[]` | No | Named collections, each with optional per-collection checkout. |
| `searches` | `QmdSearchEntry[]` | No | GitHub code search queries whose results are downloaded and indexed. |
| `cache-key` | `string` | No | GitHub Actions cache key for persisting the index across runs. When set without sources, enables read-only mode. |

### `QmdDocCollection` fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | `string` | No | Collection identifier (defaults to `"docs-<index>"`). |
| `paths` | `string[]` | No | Glob patterns for files to include (defaults to `**/*.md`). |
| `context` | `string` | No | Optional context hint for the agent about this collection's content (e.g. `"GitHub Actions documentation"`). |
| `checkout` | `CheckoutConfig` | No | Repository checkout options — same syntax as the top-level [`checkout:`](/gh-aw/reference/frontmatter/#repository-checkout-checkout) field. Defaults to the current repository. |

### `QmdSearchEntry` fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `query` | `string` | Yes | GitHub code search query string (e.g., `"repo:owner/repo language:Markdown"`). |
| `min` | `int` | No | Minimum number of results required; fails the activation job if not met (default: `0`). |
| `max` | `int` | No | Maximum number of files to download (default: `30`). |
| `github-token` | `string` | No | GitHub token for authenticated search (e.g., `${{ secrets.GITHUB_TOKEN }}`). |
| `github-app` | `GitHubAppConfig` | No | GitHub App credentials for cross-organization access. |

## Permissions

The `qmd` tool does **not** require `contents: read` in the agent job. All file access happens in the activation job, which already has that permission.

```yaml wrap
# No extra permissions needed for the agent job
permissions:
  contents: read   # activation job only — already present by default
```

## Agent usage

When qmd is active, the agent's system prompt instructs it to use the `search` tool before falling back to file listing or `bash`. Example queries:

- `"how to configure MCP servers"` — finds docs about MCP setup
- `"safe-outputs create-pull-request options"` — finds safe-output option reference
- `"permissions frontmatter field"` — finds permission configuration docs

The tool returns file paths ranked by relevance. Use standard file reading to fetch full content.

## Related Documentation

- [Tools](/gh-aw/reference/tools/) - Overview of all built-in tools
- [Frontmatter](/gh-aw/reference/frontmatter/#repository-checkout-checkout) - Top-level checkout configuration
- [Permissions](/gh-aw/reference/permissions/) - GitHub Actions permission configuration
- [Dependabot](/gh-aw/reference/dependabot/) - Automatic dependency updates (tracks `@tobilu/qmd` version)
