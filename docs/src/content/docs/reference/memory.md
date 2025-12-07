---
title: Memory
description: Complete guide to using cache-memory and repo-memory for persistent file storage across workflow runs.
sidebar:
  order: 1500
---

"Memory" for agentic workflows can be implemented in several ways:

- **GitHub Issues, Discussions, Issue Comments, files** and other GitHub information elements can be emitted to act as the "memory" of the agentic workflow, which it re-reads on each workflow run
- **Cache Memory** - Uses GitHub Actions Caches as persistent file storage with 7-day retention
- **Repo Memory** - Uses Git branches as persistent storage with unlimited retention

This document covers both **cache-memory** and **repo-memory**.

## Cache Memory

The `cache-memory` feature enables agentic workflows to maintain persistent file storage across workflow runs using GitHub Actions cache infrastructure. AI agents can store and retrieve files using standard file system operations.

When enabled, the workflow compiler automatically creates the cache directory, adds GitHub Actions cache steps for restore and save operations, generates intelligent cache keys with progressive fallback, and informs the AI agent about the available storage location.

**Default cache** uses `/tmp/gh-aw/cache-memory/` for backward compatibility. **Additional caches** use `/tmp/gh-aw/cache-memory-{id}/` to prevent path conflicts.

## Enabling Cache Memory

Enable cache-memory with default settings:

```aw wrap
---
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
```aw wrap
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
tools:
  cache-memory:
    key: project-docs-${{ github.repository }}
---

# Documentation Assistant maintaining context across runs
```

### Multiple Cache Folders

```aw wrap
---
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

## Cache Memory Real-World Examples

- **[Grumpy Code Reviewer](https://github.com/githubnext/gh-aw/blob/main/.github/workflows/grumpy-reviewer.md)** - Uses cache memory to track previous PR reviews, storing review history per PR to avoid duplicate comments and maintain context across multiple review invocations.

---

# Repo Memory

The `repo-memory` feature enables agentic workflows to maintain persistent file storage across workflow runs using Git branches. Unlike cache-memory which expires after 7 days, repo-memory provides **unlimited retention** through version-controlled storage.

When enabled, the workflow compiler automatically:
- Clones or creates the specified Git branch
- Provides file system access at `/tmp/gh-aw/repo-memory-{id}/memory/{id}/`
- Commits and pushes changes back to the branch after workflow completion
- Handles merge conflicts automatically (your changes win)
- Informs the AI agent about the available storage location

**Default memory** uses branch `memory/default` and directory `/tmp/gh-aw/repo-memory-default/memory/default/`.

## Enabling Repo Memory

Enable repo-memory with default settings:

```aw wrap
---
tools:
  repo-memory: true
  github:
    allowed: [get_repository]
---
```

This creates a Git branch at `memory/default` in the current repository and provides file access at `/tmp/gh-aw/repo-memory-default/memory/default/`.

## Using the Repo Memory Folder

AI agents can store and retrieve information using standard file operations:

```aw wrap
Please save this information to a file in the repo memory folder: "User prefers verbose error messages when debugging."
```

```aw wrap
Check what information I have stored in the repo memory folder from previous runs.
```

Files are automatically committed and pushed to the Git branch after the workflow completes, providing permanent version-controlled storage.

## Advanced Configuration

Customize branch name, file constraints, and repository target:

```aw wrap
---
tools:
  repo-memory:
    branch-name: memory/custom-agent
    description: "Long-term insights and patterns"
    file-glob: ["*.md", "*.json"]
    max-file-size: 1048576  # 1MB
    max-file-count: 50
    target-repo: "owner/repository"
  github:
    allowed: [get_repository]
---
```

### Configuration Options

- **`branch-name`** (default: `memory/default`) - Git branch name for storage
- **`description`** (optional) - Description shown to the AI agent in prompts
- **`file-glob`** (optional) - Array of file patterns allowed (e.g., `["*.md", "*.txt"]`)
- **`max-file-size`** (default: 10240 bytes = 10KB) - Maximum size per file in bytes
- **`max-file-count`** (default: 100) - Maximum number of files per commit
- **`target-repo`** (default: current repository) - Target repository in `owner/name` format
- **`create-orphan`** (default: true) - Create orphan branch if it doesn't exist

## Multiple Repo Memory Configurations

Configure multiple independent memory locations using array notation:

```aw wrap
---
tools:
  repo-memory:
    - id: insights
      branch-name: memory/insights
      description: "Long-term patterns and trends"
      file-glob: ["*.md"]
    - id: state
      branch-name: memory/state
      description: "Current state and context"
      file-glob: ["*.json"]
      max-file-size: 524288  # 512KB
    - id: history
      branch-name: memory/history
      description: "Historical data"
  github:
    allowed: [get_repository]
---
```

Each memory mounts at its own directory:
- **Default memory**: `/tmp/gh-aw/repo-memory-default/memory/default/`
- **Other memories**: `/tmp/gh-aw/repo-memory-{id}/memory/{id}/`

The `id` field is required for array notation and determines both the folder name and default branch name. If `branch-name` is omitted, it defaults to `memory/{id}`.

## Behavior and Git Integration

### Branch Creation and Cloning

**Automatic Branch Creation**: If the specified branch doesn't exist and `create-orphan: true` (default), an orphan branch is automatically created with no history.

**Existing Branch**: If the branch exists, it's cloned at workflow start with `--depth 1` for efficiency.

### Commit and Push Process

Changes are automatically committed and pushed at workflow completion:

1. **Validation**: Files are validated against `file-glob`, `max-file-size`, and `max-file-count` constraints
2. **Commit**: All changes in the memory directory are staged and committed with message: `"Update memory from workflow run {run_id}"`
3. **Pull**: Latest changes are pulled with merge strategy `-X ours` (your changes win on conflicts)
4. **Push**: Changes are pushed to the remote branch

**Conditional Execution**: Push only occurs if:
- Changes are detected in the memory directory
- Threat detection is enabled and passed (if configured)

### Merge Conflict Resolution

If concurrent workflows modify the same files:
- Pull uses `git pull --no-rebase -X ours` strategy
- **Your changes (current workflow) win** in all conflicts
- This ensures the agent's latest changes are always preserved

### Permissions

The workflow automatically adds `contents: write` permission to the push job to enable branch updates.

## Best Practices

**File Organization**: Use descriptive file names and directory structures:

```
/tmp/gh-aw/repo-memory-default/memory/default/
├── insights/
│   ├── patterns.md
│   └── trends.md
├── state/
│   ├── current.json
│   └── metadata.json
└── history/
    └── 2024-12-07.md
```

**Branch Naming**: Use descriptive, hierarchical branch names:
- `memory/default` - Default memory
- `memory/insights` - Long-term insights
- `memory/agent-name` - Agent-specific memory

**Memory Scope**:
- **Workflow-specific** (default): Each workflow has its own memory branch
- **Shared**: Use the same branch name across multiple workflows to share memory
- **Cross-repository**: Set `target-repo` to store memory in a different repository

**File Constraints**: Use constraints to prevent abuse:
- Set `file-glob` to restrict file types (e.g., `["*.md", "*.json"]`)
- Set `max-file-size` to prevent large files
- Set `max-file-count` to limit number of files

**Storage Management**: 
- Monitor branch size and commit frequency
- Consider periodic cleanup of old data
- Use structured formats (JSON, YAML) for easier processing

## Comparing Cache Memory vs Repo Memory

| Feature | Cache Memory | Repo Memory |
|---------|--------------|-------------|
| **Storage** | GitHub Actions Cache | Git Branches |
| **Retention** | 7 days | Unlimited |
| **Size Limit** | 10GB per repository | Repository size limits |
| **Version Control** | No | Yes (full Git history) |
| **Access** | File operations | File operations |
| **Performance** | Fast (cached) | Slower (Git clone/push) |
| **Best For** | Temporary data, sessions | Long-term insights, history |

**Choose Cache Memory** when you need fast, temporary storage for session data or short-term context.

**Choose Repo Memory** when you need permanent, version-controlled storage for insights, patterns, or historical data.

## Troubleshooting

**Branch Not Created**: Ensure `create-orphan: true` (default) is set, or manually create the branch before first run.

**Permission Denied**: Ensure the workflow has `contents: write` permission. This is automatically added by the compiler.

**File Validation Failures**: Check that your files:
- Match the `file-glob` patterns (if specified)
- Are smaller than `max-file-size` (default: 10KB)
- Don't exceed `max-file-count` (default: 100 files)

**Changes Not Persisting**: Verify:
- Files are in the correct directory: `/tmp/gh-aw/repo-memory-{id}/memory/{id}/`
- Workflow completed successfully
- Check workflow logs for push errors

**Merge Conflicts**: Repo memory uses `-X ours` strategy, so your changes always win. If you need to preserve previous data, read it before writing new data.

## Security Considerations

**Data Privacy**: 
- Memory branches are part of the repository and follow repository access permissions
- Use private repositories for sensitive data
- Avoid storing secrets or credentials in memory files

**File Validation**: 
- Use `file-glob` to restrict allowed file types
- Set `max-file-size` to prevent large file abuse
- Set `max-file-count` to limit storage growth

**Branch Protection**:
- Memory branches are standard Git branches
- Consider branch protection rules for production workflows
- Use `target-repo` to isolate memory in a separate repository

## Examples

### Basic Memory Storage

```aw wrap
---
on: workflow_dispatch
tools:
  repo-memory: true
---

# Store insights in repo memory
# Check what previous insights exist
# Add new insights based on current analysis
```

### Long-Term Insights Tracking

```aw wrap
---
tools:
  repo-memory:
    branch-name: memory/deep-insights
    description: "Long-term insights, patterns, and trend data"
    file-glob: ["*.md"]
    max-file-size: 1048576  # 1MB
---

# Deep Analysis Agent
# Store patterns and trends over time
```

### Multiple Memory Locations

```aw wrap
---
on: workflow_dispatch
tools:
  repo-memory:
    - id: insights
      branch-name: memory/insights
      description: "Long-term patterns and trends"
    - id: state
      branch-name: memory/state
      description: "Current state and context"
    - id: history
      branch-name: memory/history
      description: "Historical event log"
---

# Organize different types of memory
# Store insights in /tmp/gh-aw/repo-memory-insights/memory/insights/
# Store state in /tmp/gh-aw/repo-memory-state/memory/state/
# Store history in /tmp/gh-aw/repo-memory-history/memory/history/
```

### Cross-Repository Memory

```aw wrap
---
tools:
  repo-memory:
    target-repo: "myorg/memory-store"
    branch-name: memory/shared-insights
    description: "Shared insights across multiple repositories"
---

# Store insights in a central memory repository
# Access memory from multiple project repositories
```

## Repo Memory Real-World Examples

- **[Deep Report](https://github.com/githubnext/gh-aw/blob/main/.github/workflows/deep-report.md)** - Uses repo memory to track long-term insights, patterns, and trends from discussion analysis, storing findings in markdown files with constraints to prevent abuse.
- **[Daily Firewall Report](https://github.com/githubnext/gh-aw/blob/main/.github/workflows/daily-firewall-report.md)** - Uses repo memory to maintain historical data about security patterns and anomalies over time.

---

## Related Documentation

- [Frontmatter](/gh-aw/reference/frontmatter/) - Complete frontmatter configuration guide
- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Output processing and automation
- [GitHub Actions Cache Documentation](https://docs.github.com/en/actions/using-workflows/caching-dependencies-to-speed-up-workflows) - Official GitHub cache documentation