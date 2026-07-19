---
title: Cache Memory
description: Guide to using cache-memory for persistent file storage across workflow runs with GitHub Actions cache.
sidebar:
  order: 1500
---

Cache memory provides persistent file storage across workflow runs via GitHub Actions cache with 7-day retention. The compiler automatically configures the cache directory, restore/save operations, and progressive fallback keys at `/tmp/gh-aw/cache-memory/` (default) or `/tmp/gh-aw/cache-memory-{id}/` (additional caches).

## Enabling Cache Memory

```aw wrap
---
tools:
  cache-memory: true
---
```

Stores files at `/tmp/gh-aw/cache-memory/` using a workflow-scoped cache key. Use standard file operations to store/retrieve JSON/YAML, text files, or subdirectories.

## Advanced Configuration

```aw wrap
---
tools:
  cache-memory:
    key: custom-memory-${{ github.repository_owner }}
    retention-days: 30  # 1-90 days, extends access beyond cache expiration
    allowed-extensions: [".json", ".txt", ".md"]  # Restrict file types (default: empty/all files allowed)
---
```

> [!NOTE]
> Do not include `${{ github.run_id }}` in a user-supplied key — the compiler appends it automatically to the save key and generates stable restore-keys from the prefix.

### File Type Restrictions

The `allowed-extensions` field restricts which file types can be written to cache-memory. By default, all file types are allowed (empty array). When specified, only files with listed extensions can be stored.

```aw wrap
---
tools:
  cache-memory:
    allowed-extensions: [".json", ".jsonl", ".txt"]  # Only these extensions allowed
---
```

If files with disallowed extensions are found, the workflow will report validation failures.

When a cache is restored for agent execution, gh-aw also strips execute bits from restored working-tree files and removes disallowed file types before the agent can read them. See [ADR-26587](https://github.com/github/gh-aw/blob/main/docs/adr/26587-pre-agent-cache-memory-working-tree-sanitization.md) for the pre-agent sanitization contract behind `allowed-extensions`.

## Multiple Configurations

```aw wrap
---
tools:
  cache-memory:
    - id: default
      key: memory-default
    - id: session
      key: memory-session-${{ github.run_id }}
    - id: logs
      retention-days: 7
---
```

Mounts at `/tmp/gh-aw/cache-memory/` (default) or `/tmp/gh-aw/cache-memory-{id}/`. The `id` determines the folder name; `key` defaults to a workflow-scoped prefix derived from the sanitized workflow name.

## Merging from Shared Workflows

```aw wrap
---
imports:
  - shared/mcp/server-memory.md

tools:
  cache-memory: true
---
```

Merge rules: **Single→Single** (local overrides), **Single→Multiple** (local converts to array), **Multiple→Multiple** (merge by `id`, local wins).

## Behavior

GitHub Actions cache provides 7-day retention, a 10GB per-repository limit, and LRU eviction. Add `retention-days` to upload artifacts (1-90 days) when you need access beyond normal cache expiration.

Cache memory is branch-scoped. Runs restore from caches on the same branch and can also fall back to the default branch. On a non-default branch, the first restore often comes from the default branch; later saves then create a branch-local cache lineage.

The compiler strips `${{ github.run_id }}` from restore keys so each run can fall back to earlier runs, and for `scope: repo` it adds a broader restore key for cross-workflow sharing within the same branch scope. Custom user-supplied keys automatically append `-${{ github.run_id }}` when needed.

## Best Practices

Use cache-memory for short-lived, branch-local state. Prefer scheduled runs on the default branch when a workflow depends on warmed caches, and use descriptive file names, hierarchical keys such as `project-${{ github.repository_owner }}-${{ github.workflow }}`, and the narrowest practical scope. Monitor total cache growth within the 10GB repository limit.

## Comparison with Repo Memory

| Feature | Cache Memory | Repo Memory |
|---------|--------------|-------------|
| Storage | GitHub Actions Cache | Git Branches |
| Retention | 7 days | Unlimited |
| Size Limit | 10GB/repo | Repository limits |
| Version Control | No | Yes |
| Performance | Fast | Slower |
| Best For | Temporary/sessions | Long-term/history |

For unlimited retention with version control, see [Repo Memory](/gh-aw/reference/repo-memory/).

## Automatic Cleanup

The [agentic maintenance](/gh-aw/reference/ephemerals/#cache-memory-cleanup) workflow automatically removes outdated cache-memory entries on a schedule by grouping caches by key prefix (everything before the run ID) and keeping only the latest entry in each group. You can also trigger the same cleanup manually from the GitHub Actions UI by running the `Agentic Maintenance` workflow with the `clean_cache_memories` operation.

## Troubleshooting

If files are not persisting, check cache key consistency and the restore/save log messages. For file access issues, create subdirectories first, verify permissions, and use absolute paths. If cache growth becomes a problem, clear old entries periodically or use time-based keys for auto-expiration.

When an agent calls `missing_data` with `reason: `cache_memory_miss``, the conclusion handler automatically opens a failure issue that points to a likely cache path problem. Verify that the prompt uses the correct path (`/tmp/gh-aw/cache-memory/` by default or `/tmp/gh-aw/cache-memory-{id}/` for named caches) and that the cache key stays consistent across runs.

## Integrity-Aware Caching

When a workflow uses `tools.github.min-integrity`, cache-memory automatically applies integrity-level isolation. Cache keys include the workflow's integrity level and a hash of the guard policy so that changing any policy field forces a cache miss.

The compiler generates git-backed branching steps around the agent. Before the agent runs, it checks out the matching integrity branch and merges down from all higher-integrity branches (higher integrity always wins conflicts). After the agent runs, changes are committed to that branch. The agent itself sees only plain files — the `.git/` directory rides along transparently in the Actions cache tarball.

### Merge semantics

| Run integrity | Sees data written by | Cannot see |
|---|---|---|
| `merged` | `merged` only | `approved`, `unapproved`, `none` |
| `approved` | `approved` + `merged` | `unapproved`, `none` |
| `unapproved` | `unapproved` + `approved` + `merged` | `none` |
| `none` | all levels | — |

This prevents a lower-integrity agent from poisoning data that a higher-integrity run would later read.

> [!NOTE]
> Existing caches will get a cache miss on first run after upgrading to a version that includes this feature — intentional, as legacy data has no integrity provenance.

## Security

Do not store sensitive data in cache memory. It follows repository permissions, and with [threat detection](/gh-aw/reference/threat-detection/) enabled, caches save only after validation succeeds (`restore → modify → upload artifact → validate → save`).

## Examples

See [Grumpy Code Reviewer](https://github.com/github/gh-aw/blob/main/.github/workflows/grumpy-reviewer.md) for tracking PR review history.

## Related Documentation

- [Repo Memory](/gh-aw/reference/repo-memory/) - Git branch-based persistent storage with unlimited retention
- [Frontmatter](/gh-aw/reference/frontmatter/) - Complete frontmatter configuration guide
- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Output processing and automation
- [GitHub Actions Cache Documentation](https://docs.github.com/en/actions/using-workflows/caching-dependencies-to-speed-up-workflows) - Official GitHub cache documentation
