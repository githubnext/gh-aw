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

Stores files at `/tmp/gh-aw/cache-memory/` using default key `memory-${{ github.workflow }}-${{ github.run_id }}`. Use standard file operations to store/retrieve JSON/YAML, text files, or subdirectories.

## Advanced Configuration

```aw wrap
---
tools:
  cache-memory:
    key: custom-memory-${{ github.workflow }}-${{ github.run_id }}
    retention-days: 30  # 1-90 days, extends access beyond cache expiration
    allowed-extensions: [".json", ".txt", ".md"]  # Restrict file types (default: empty/all files allowed)
---
```

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

Mounts at `/tmp/gh-aw/cache-memory/` (default) or `/tmp/gh-aw/cache-memory-{id}/`. The `id` determines folder name; `key` defaults to `memory-{id}-${{ github.workflow }}-${{ github.run_id }}`.

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

GitHub Actions cache: 7-day retention, 10GB per repo, LRU eviction. Add `retention-days` to upload artifacts (1-90 days) for extended access.

Caches accessible across branches with unique per-run keys. Custom keys auto-append `-${{ github.run_id }}`. Progressive restore splits on dashes: `custom-memory-project-v1-${{ github.run_id }}` tries `custom-memory-project-v1-`, `custom-memory-project-`, `custom-memory-`, `custom-`.

## Best Practices

Use descriptive file/directory names, hierarchical cache keys (`project-${{ github.repository_owner }}-${{ github.workflow }}`), and appropriate scope (workflow-specific default or repository/user-wide). Monitor growth within 10GB limit.

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

## Troubleshooting

- **Files not persisting**: Check cache key consistency and logs for restore/save messages.
- **File access issues**: Create subdirectories first, verify permissions, use absolute paths.
- **Cache size issues**: Track growth, clear periodically, or use time-based keys for auto-expiration.

## Security

Don't store sensitive data in cache memory. Cache memory follows repository permissions.

Logs access. With [threat detection](/gh-aw/reference/threat-detection/), cache saves only after validation succeeds (restore→modify→upload artifact→validate→save).

## Examples

See [Grumpy Code Reviewer](https://github.com/github/gh-aw/blob/main/.github/workflows/grumpy-reviewer.md) for tracking PR review history.

## Related Documentation

- [Repo Memory](/gh-aw/reference/repo-memory/) - Git branch-based persistent storage with unlimited retention
- [Frontmatter](/gh-aw/reference/frontmatter/) - Complete frontmatter configuration guide
- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Output processing and automation
- [GitHub Actions Cache Documentation](https://docs.github.com/en/actions/using-workflows/caching-dependencies-to-speed-up-workflows) - Official GitHub cache documentation
