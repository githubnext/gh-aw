---
title: Memory
description: Complete guide to using cache-memory for persistent file storage across workflow runs using GitHub Actions cache and simple file operations.
sidebar:
  order: 1500
---

"Memory" for agentic workflows can be implemented in several ways:

- GitHub Issues, Discussions, Issue Comments, files and other GitHub information elements can be emitted to act as the "memory" of the agentic workflow, which it re-reads on each workflow run
- A "cache memory" feature is available to use GitHub Actions Caches as a persisitent file storage. 

This document covers "cache memory".

## Cache Memory

The `cache-memory` feature enables agentic workflows to maintain persistent file storage across workflow runs using GitHub Actions cache infrastructure. AI agents can store and retrieve files using standard file system operations.

When enabled, the workflow compiler automatically creates the cache directory, adds GitHub Actions cache steps for restore and save operations, generates intelligent cache keys with progressive fallback, and informs the AI agent about the available storage location.

**Default cache** uses `/tmp/gh-aw/cache-memory/` for backward compatibility. **Additional caches** use `/tmp/gh-aw/cache-memory-{id}/` to prevent path conflicts.

## Enabling Cache Memory

Enable cache-memory with default settings:

```aw wrap
---
engine: claude
tools:
  cache-memory: true
  github:
    allowed: [get_repository]
---
```

This uses the default cache key `memory-${{ github.workflow }}-${{ github.run_id }}` and stores files at `/tmp/gh-aw/cache-memory/` using standard file operations.

## Using the Cache Folder

AI agents can store and retrieve information using standard file operations:

```aw wrap
Please save this information to a file in the cache folder: "User prefers verbose error messages when debugging."
```

```aw wrap
Check what information I have stored in the cache folder from previous runs.
```

Files can be organized as JSON/YAML for structured data, text files for notes and logs, or in subdirectories for better structure.

## Advanced Configuration

Customize cache key and artifact retention:

```aw wrap
---
engine: claude
tools:
  cache-memory:
    key: custom-memory-${{ github.workflow }}-${{ github.run_id }}
    retention-days: 30  # Keep artifacts for 30 days (1-90 range)
  github:
    allowed: [get_repository]
---
```

The `retention-days` option (1-90 days, defaults to repository setting) controls `actions/upload-artifact` retention and provides alternative access to cache data beyond the standard 7-day cache expiration.

## Multiple Cache Configurations

Configure multiple independent cache folders using array notation:

```aw wrap
---
engine: claude
tools:
  cache-memory:
    - id: default
      key: memory-default
    - id: session
      key: memory-session-${{ github.run_id }}
    - id: logs
      retention-days: 7
  github:
    allowed: [get_repository]
---
```

Each cache mounts at its own directory with independent persistence:
- **Default cache**: `/tmp/gh-aw/cache-memory/`
- **Other caches**: `/tmp/gh-aw/cache-memory-{id}/`

The `id` field is required for array notation and determines the cache folder name. If `key` is omitted, it defaults to `memory-{id}-${{ github.workflow }}-${{ github.run_id }}`.

When multiple caches are configured, the AI agent receives information about all available cache folders and can organize data across different storage locations based on purpose (e.g., session data, logs, persistent configuration).

## Cache Merging from Shared Workflows

Cache-memory configurations can be imported and merged from shared workflow files using the `imports:` field:

```aw wrap
---
engine: claude
imports:
  - shared/mcp/server-memory.md
tools:
  cache-memory: true
---
```

When importing shared workflows that define cache-memory configurations, the caches are merged following these rules:

**Single to Single**: The local configuration overrides the imported configuration.

**Single to Multiple**: The local single cache is converted to an array and merged with imported caches.

**Multiple to Multiple**: Cache arrays are merged by `id`. If the same `id` exists in both local and imported configs, the local configuration takes precedence.

**Example Merge Scenario**:

Import file (`shared/memory-setup.md`):
```aw
---
tools:
  cache-memory:
    - id: shared-state
      key: app-state
---
```

Local workflow:
```aw wrap
---
imports:
  - shared/memory-setup.md
tools:
  cache-memory:
    - id: local-logs
      key: workflow-logs
---
```

Result: Two cache folders at `/tmp/gh-aw/cache-memory/` and `/tmp/gh-aw/cache-memory-local-logs/`.

## Cache Behavior and GitHub Actions Integration

Cache Memory leverages GitHub Actions cache with 7-day retention, 10GB per repository limit, and LRU eviction. When `retention-days` is configured, cache data is also uploaded as artifacts (1-90 days retention) for long-term persistence.

**Scoping**: Caches are accessible across branches but each workflow maintains its own namespace by default. Each run gets unique cache keys to prevent conflicts.

**Automatic Key Generation**: Custom keys automatically get `-${{ github.run_id }}` appended (e.g., `project-memory` becomes `project-memory-${{ github.run_id }}`).

**Progressive Restore Keys**: Restore keys are generated by splitting the cache key on dashes. For key `custom-memory-project-v1-${{ github.run_id }}`, restore keys are `custom-memory-project-v1-`, `custom-memory-project-`, `custom-memory-`, `custom-`, ensuring the most specific match is found first.

## Best Practices

**File Organization**: Use descriptive file names and directory structures:

```
/tmp/gh-aw/cache-memory/
├── preferences/user-settings.json
├── logs/activity.log
├── state/context.json
└── notes.md
```

**Cache Key Naming**: Use descriptive, hierarchical keys like `project-${{ github.repository_owner }}-${{ github.workflow }}`.

**Memory Scope**: Choose workflow-specific (default), repository-wide (include repository name in key), or user-specific (include user info in key) caching based on your needs.

**Resource Management**: Monitor cache data growth, respect GitHub's 10GB limit, and consider periodic cache clearing for long-running projects.

## Troubleshooting

**Files Not Persisting**: Ensure cache keys are consistent between runs, verify the cache directory exists, and check workflow logs for cache restore/save messages.

**File Access Issues**: Create subdirectories before use, verify write permissions, and use absolute paths within the cache directory.

**Cache Size Issues**: Track cache growth, implement periodic cache clearing, or use time-based cache keys for automatic expiration.

## Security Considerations

**Data Privacy**: Avoid storing sensitive information in cache files. Cache data follows repository access permissions and access is logged in workflow execution logs.

**File Security**: Files use standard GitHub Actions runner permissions. The cache directory is temporary and cleaned between runs, with no external access.

## Examples

### Basic File Storage

```aw wrap
---
engine: claude
on:
  workflow_dispatch:
    inputs:
      note:
        description: 'Note to remember'
        required: true
tools:
  cache-memory: true
---

# Store the note "${{ inputs.note }}" in a timestamped file
# List all files in the cache folder
```

### Project-Specific Cache

```aw wrap
---
engine: claude
tools:
  cache-memory:
    key: project-docs-${{ github.repository }}
---

# Documentation Assistant maintaining context across runs
```

### Multiple Cache Folders

```aw wrap
---
engine: claude
on: workflow_dispatch
tools:
  cache-memory:
    - id: context
      key: agent-context
    - id: artifacts
      key: build-artifacts
      retention-days: 14
---

# Store agent context in /tmp/gh-aw/cache-memory/
# Store build artifacts in /tmp/gh-aw/cache-memory-artifacts/
```

## Related Documentation

- [Frontmatter](/gh-aw/reference/frontmatter/) - Complete frontmatter configuration guide
- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Output processing and automation
- [GitHub Actions Cache Documentation](https://docs.github.com/en/actions/using-workflows/caching-dependencies-to-speed-up-workflows) - Official GitHub cache documentation