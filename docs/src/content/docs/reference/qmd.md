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

**Missing or expired cache behavior:** When the configured `cache-key` is absent (the cache was never populated or has been evicted by GitHub Actions' 7-day cache expiry policy):

- qmd MUST log a warning describing the missing cache key.
- The agent job MUST proceed with an empty search index — it MUST NOT abort.
- `qmd_search` tool calls will return no results until the index is rebuilt.
- Operators SHOULD monitor for empty index warnings and re-run or pre-populate the cache key when eviction is detected.
- When `checkouts` or `searches` are also configured alongside a `cache-key`, qmd MUST rebuild the index from those sources during the current run regardless of cache availability.

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

## Telemetry

Use `otlp.cjs` in a shared import step to record qmd index size and search hits alongside the workflow's distributed trace.

```yaml wrap title=".github/workflows/shared/qmd-otlp.md"
---
# Shared import: emit qmd index size and search hit counts after the agent job.

steps:
  - name: Record qmd telemetry
    id: qmd-otlp
    uses: actions/github-script@v8
    with:
      script: |
        const fs   = require('fs');
        const otlp = require('/tmp/gh-aw/actions/otlp.cjs');

        // qmd writes index stats to /tmp/gh-aw/qmd/stats.json after indexing.
        let indexSize = 0;
        try {
          const stats = JSON.parse(fs.readFileSync('/tmp/gh-aw/qmd/stats.json', 'utf8'));
          indexSize = stats.index_size ?? 0;
        } catch { /* index not available */ }

        // qmd appends one JSON line per query to /tmp/gh-aw/qmd/queries.jsonl.
        let hits = 0;
        try {
          const lines = fs.readFileSync('/tmp/gh-aw/qmd/queries.jsonl', 'utf8').trim().split('\n');
          hits = lines.reduce((sum, l) => {
            try { return sum + (JSON.parse(l).hits ?? 0); } catch { return sum; }
          }, 0);
        } catch { /* no queries yet */ }

        await otlp.logSpan('qmd', {
          'qmd.index.size':   indexSize,
          'qmd.search.hits':  hits,
        });
---
```

Import the shared file in any workflow that uses qmd:

```aw wrap
---
on: push

engine: copilot

imports:
  - shared/otlp.md   # sets OTEL_EXPORTER_OTLP_ENDPOINT
  - shared/qmd-otlp.md             # records index size and search hits

tools:
  qmd:
    checkouts:
      - pattern: "docs/**/*.md"
---
```

## Norms

The following normative statements govern qmd cache-key management, index staleness, and performance expectations.

- Cache keys SHOULD be rotated at least once per week to prevent index drift as documentation sources change. Dynamic cache keys using `${{ github.run_id }}` MUST produce a fresh index on every run. Static cache keys MUST share one index across all runs until the key is manually invalidated or expired.
- Stale indexes (restored from a cache key whose content has diverged from the source files) are accepted by the agent without error. Operators who require freshness guarantees SHOULD include a content hash or date component in the `cache-key` expression to force periodic rebuilds.
- Index builds SHOULD complete within 5 minutes for collections up to 10,000 files. Collections that exceed this budget SHOULD use `searches` queries in place of broad `checkouts` patterns to reduce the file set.
- When a `cache-key` is set without indexing sources, the index MUST be treated as read-only.
- Any attempt to write to the index in read-only mode MUST be rejected and reported as a tool call error to the agent.
- The agent job itself MUST continue running after a read-only write rejection; the error MUST NOT abort the job.
- Implementations MUST NOT fail the agent job when the cache is absent; the agent MUST proceed with an empty search index and log a diagnostic warning (see [`cache-key` → Missing or expired cache behavior](#cache-key)).

## Related Documentation

- [Tools](/gh-aw/reference/tools/) - Overview of all available tools and configuration
- [Frontmatter](/gh-aw/reference/frontmatter/) - Complete frontmatter configuration guide
- [Cache Memory](/gh-aw/reference/cache-memory/) - Persistent memory across workflow runs
- [GitHub Tools](/gh-aw/reference/github-tools/) - GitHub API operations
- [OpenTelemetry](/gh-aw/guides/open-telemetry/#custom-spans-from-shared-imports) - Emit telemetry from shared imports
