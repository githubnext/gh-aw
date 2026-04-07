---
title: QMD Documentation Search
description: Configure vector similarity search over documentation files using the qmd tool in agentic workflows.
sidebar:
  order: 820
---

QMD Documentation Search provides vector similarity search over documentation files. It runs [tobi/qmd](https://github.com/tobi/qmd) as an MCP server so agents can find relevant documentation by natural language query.

The search index is built in a dedicated indexing job (which has `contents: read`) and shared with the agent job via GitHub Actions cache, so the agent job does not need `contents: read` permission.

:::caution[Experimental]
QMD Documentation Search is an experimental feature. The API may change in future releases.
:::

## Basic Configuration

```aw wrap
---
tools:
  qmd:
    checkouts:
      - pattern: "docs/**/*.md"
---
```

## Configuration Options

### `checkouts`

A list of named documentation collections built from checked-out repositories. Each entry specifies which files to index from the current repository or a different repository.

```aw wrap
---
tools:
  qmd:
    checkouts:
      - pattern: "docs/**/*.md"
      - pattern: "README.md"
---
```

Each checkout entry can optionally specify its own checkout configuration to target a different repository.

### `searches`

A list of GitHub code search queries whose results are downloaded and added to the qmd index.

```aw wrap
---
tools:
  qmd:
    searches:
      - query: "repo:github/gh-aw language:markdown"
---
```

### `cache-key`

A GitHub Actions cache key used to persist the qmd index across workflow runs. When set without any indexing sources (`checkouts`/`searches`), qmd operates in read-only mode: the index is restored from cache and all indexing steps are skipped.

```aw wrap
---
tools:
  qmd:
    cache-key: "qmd-docs-${{ github.repository }}"
---
```

### `gpu`

Enable GPU acceleration for the embedding model (`node-llama-cpp`). Defaults to `false`: `NODE_LLAMA_CPP_GPU=false` is injected into the indexing step so GPU probing is skipped on CPU-only runners. Set to `true` only when the indexing runner has a GPU.

```aw wrap
---
tools:
  qmd:
    gpu: true
    runs-on: gpu-runner
---
```

### `runs-on`

Override the runner image for the qmd indexing job. Defaults to the same runner as the agent job. Use this when the indexing job requires a different runner (e.g. a GPU runner).

```aw wrap
---
tools:
  qmd:
    runs-on: ubuntu-latest
---
```

## Example: Index Documentation from Multiple Sources

```aw wrap
---
tools:
  qmd:
    checkouts:
      - pattern: "docs/**/*.md"
      - pattern: "*.md"
    cache-key: "qmd-docs-${{ github.repository }}-${{ github.run_id }}"
---
```

## Example: Read-Only Mode with Pre-Built Index

```aw wrap
---
tools:
  qmd:
    cache-key: "qmd-docs-my-project"
---
```

In read-only mode, the index is restored from cache and no indexing steps are run. This is useful when the index is built separately and shared across workflows.

## Related Documentation

- [Tools](/gh-aw/reference/tools/) - Overview of all available tools and configuration
- [Frontmatter](/gh-aw/reference/frontmatter/) - Complete frontmatter configuration guide
- [Cache Memory](/gh-aw/reference/cache-memory/) - Persistent memory across workflow runs
- [GitHub Tools](/gh-aw/reference/github-tools/) - GitHub API operations
