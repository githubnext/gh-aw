---
title: Memory
description: Complete guide to using cache-memory and repo-memory for persistent file storage across workflow runs.
sidebar:
  order: 1500
---

Agentic workflows can maintain persistent memory through GitHub Issues/Discussions/files, **cache-memory** (GitHub Actions cache with 7-day retention), or **repo-memory** (Git branches with unlimited retention).

This guide covers cache-memory and repo-memory configuration.

## Cache Memory

Enables persistent file storage across workflow runs using GitHub Actions cache. When enabled, the compiler automatically sets up the cache directory, restore/save operations, and progressive fallback keys.

Storage locations: `/tmp/gh-aw/cache-memory/` (default) or `/tmp/gh-aw/cache-memory-{id}/` (additional caches).

## Enabling Cache Memory

```aw wrap
---
tools:
  cache-memory: true
---
```

Uses default key `memory-${{ github.workflow }}-${{ github.run_id }}` and stores files at `/tmp/gh-aw/cache-memory/`.

## Using the Cache Folder

Store and retrieve information using standard file operations. Organize files as JSON/YAML (structured data), text files (notes/logs), or subdirectories.

## Advanced Configuration

```aw wrap
---
tools:
  cache-memory:
    key: custom-memory-${{ github.workflow }}-${{ github.run_id }}
    retention-days: 30  # 1-90 days, defaults to repo setting
---
```

The `retention-days` controls artifact retention, providing access beyond the 7-day cache expiration.

## Multiple Cache Configurations

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

Each cache mounts at `/tmp/gh-aw/cache-memory/` (default) or `/tmp/gh-aw/cache-memory-{id}/` (others). The `id` field is required and determines the folder name. If `key` is omitted, defaults to `memory-{id}-${{ github.workflow }}-${{ github.run_id }}`.

## Cache Merging from Shared Workflows

Import cache-memory configurations from shared workflow files:

```aw wrap
---
imports:
  - shared/mcp/server-memory.md
tools:
  cache-memory: true
---
```

Merge rules: **Single to Single** (local overrides imported), **Single to Multiple** (local converted to array and merged), **Multiple to Multiple** (merged by `id`, local takes precedence).

## Cache Behavior and GitHub Actions Integration

Uses GitHub Actions cache with 7-day retention, 10GB per repository limit, and LRU eviction. With `retention-days`, cache data uploads as artifacts (1-90 days) for long-term access.

Caches are accessible across branches with unique per-run keys. Custom keys automatically append `-${{ github.run_id }}`. Progressive restore keys split on dashes (e.g., `custom-memory-project-v1-${{ github.run_id }}` tries `custom-memory-project-v1-`, `custom-memory-project-`, `custom-memory-`, `custom-`).

## Best Practices

Organize files with descriptive names and directories. Use hierarchical cache keys like `project-${{ github.repository_owner }}-${{ github.workflow }}`. Choose appropriate scope (workflow-specific by default, or repository/user-wide by including identifiers in keys). Monitor growth and respect the 10GB limit.

## Troubleshooting

**Files Not Persisting**: Check cache key consistency and workflow logs for restore/save messages.

**File Access Issues**: Create subdirectories before use, verify permissions, use absolute paths.

**Cache Size Issues**: Track growth, clear periodically, or use time-based keys for auto-expiration.

## Security Considerations

Avoid storing sensitive data in cache files. Cache follows repository permissions and logs access in workflow logs. Files use standard runner permissions in temporary directories.

With [threat detection](/gh-aw/reference/safe-outputs/#threat-detection) enabled, cache updates defer until validation completes: restored via `actions/cache/restore`, modified by agent, uploaded as artifacts, validated, then saved via `actions/cache/save` only if detection succeeds. Without threat detection, updates occur automatically via standard cache post-action.

## Examples

Basic usage with `cache-memory: true`, project-specific with custom keys, or multiple caches with different retention. See [Grumpy Code Reviewer](https://github.com/githubnext/gh-aw/blob/main/.github/workflows/grumpy-reviewer.md) for tracking PR review history.

---

# Repo Memory

Enables persistent file storage using Git branches with unlimited retention. When enabled, the compiler automatically clones/creates the branch, provides file access at `/tmp/gh-aw/repo-memory-{id}/memory/{id}/`, commits and pushes changes, and handles merge conflicts (your changes win).

Default: branch `memory/default` at `/tmp/gh-aw/repo-memory-default/memory/default/`.

## Enabling Repo Memory

```aw wrap
---
tools:
  repo-memory: true
---
```

Creates branch `memory/default` with file access at `/tmp/gh-aw/repo-memory-default/memory/default/`. Files are automatically committed and pushed after workflow completion.

## Advanced Configuration

```aw wrap
---
tools:
  repo-memory:
    branch-name: memory/custom-agent
    description: "Long-term insights and patterns"
    file-glob: ["*.md", "*.json"]
    max-file-size: 1048576  # 1MB, default 10KB
    max-file-count: 50      # default 100
    target-repo: "owner/repository"
    create-orphan: true     # default true
---
```

Options: `branch-name` (default `memory/default`), `description`, `file-glob`, `max-file-size`, `max-file-count`, `target-repo`, `create-orphan`.

## Multiple Repo Memory Configurations

```aw wrap
---
tools:
  repo-memory:
    - id: insights
      branch-name: memory/insights
      file-glob: ["*.md"]
    - id: state
      file-glob: ["*.json"]
      max-file-size: 524288  # 512KB
---
```

Each mounts at `/tmp/gh-aw/repo-memory-{id}/memory/{id}/`. The `id` field is required and determines folder/branch names. If `branch-name` is omitted, defaults to `memory/{id}`.

## Behavior and Git Integration

Branches are auto-created as orphans (if `create-orphan: true`, default) or cloned with `--depth 1`. Changes commit automatically after validation (against `file-glob`, `max-file-size`, `max-file-count`), pull with `-X ours` (your changes win conflicts), and push. Push occurs only if changes detected and threat detection passes (if configured). Automatically adds `contents: write` permission.

## Best Practices

Organize files with descriptive names and directories. Use hierarchical branch names (`memory/default`, `memory/insights`). Choose scope (workflow-specific default, shared across workflows, or cross-repository with `target-repo`). Set constraints (`file-glob`, `max-file-size`, `max-file-count`) to prevent abuse. Monitor branch size and clean periodically.

## Comparing Cache Memory vs Repo Memory

| Feature | Cache Memory | Repo Memory |
|---------|--------------|-------------|
| **Storage** | GitHub Actions Cache | Git Branches |
| **Retention** | 7 days | Unlimited |
| **Size Limit** | 10GB/repo | Repository limits |
| **Version Control** | No | Yes |
| **Performance** | Fast | Slower |
| **Best For** | Temporary/sessions | Long-term/history |

Choose cache for fast temporary storage, repo for permanent version-controlled storage.

## Troubleshooting

**Branch Not Created**: Ensure `create-orphan: true` or create branch manually.

**Permission Denied**: `contents: write` is auto-added by compiler.

**File Validation Failures**: Verify files match `file-glob`, are under `max-file-size` (10KB default), and within `max-file-count` (100 default).

**Changes Not Persisting**: Check correct directory (`/tmp/gh-aw/repo-memory-{id}/memory/{id}/`), successful workflow completion, and push errors in logs.

**Merge Conflicts**: Uses `-X ours`, your changes always win. Read before writing to preserve previous data.

## Security Considerations

Memory branches follow repository permissions. Use private repos for sensitive data. Avoid storing secrets. Use `file-glob`, `max-file-size`, and `max-file-count` to restrict files. Consider branch protection rules for production. Use `target-repo` to isolate memory.

## Examples

Basic usage with `repo-memory: true`, custom branches with constraints, multiple memory locations by ID, or cross-repository with `target-repo`. See [Deep Report](https://github.com/githubnext/gh-aw/blob/main/.github/workflows/deep-report.md) and [Daily Firewall Report](https://github.com/githubnext/gh-aw/blob/main/.github/workflows/daily-firewall-report.md) for tracking long-term insights and historical security data.

---

## Related Documentation

- [Frontmatter](/gh-aw/reference/frontmatter/) - Complete frontmatter configuration guide
- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Output processing and automation
- [GitHub Actions Cache Documentation](https://docs.github.com/en/actions/using-workflows/caching-dependencies-to-speed-up-workflows) - Official GitHub cache documentation