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

The `cache-memory` feature enables agentic workflows to maintain persistent file storage across workflow runs using GitHub Actions cache infrastructure and simple file operations.

Cache Memory provides:

- **Persistent File Storage**: AI agents can store and retrieve files across multiple workflow runs
- **GitHub Actions Integration**: Built on top of GitHub Actions cache infrastructure 
- **Simple File Operations**: Uses standard file system operations instead of specialized tools
- **Automatic Configuration**: Seamlessly integrates with Claude and Custom engines
- **Smart Caching**: Intelligent cache key generation and restoration strategies

## How It Works

When `cache-memory` is enabled, the workflow compiler automatically:

1. **Creates Cache Directory**: Sets up `/tmp/gh-aw/cache-memory/` directory for file storage
2. **Creates Cache Steps**: Adds GitHub Actions cache steps to restore and save data
3. **Persistent Storage**: Maps `/tmp/gh-aw/cache-memory/` to store user data files
4. **Cache Key Management**: Generates intelligent cache keys with progressive fallback
5. **Prompts LLM**: Informs the AI agent about the available cache folder

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

This uses:
- **Default cache key**: `memory-${{ github.workflow }}-${{ github.run_id }}`
- **Simple file access**: Uses standard file operations for read/write
- **Default storage path**: `/tmp/gh-aw/cache-memory/` for data files

## Using the Cache Folder

The cache folder is accessible to AI agents and provides file system access for persistent storage.

### Storing Information

The AI agent can store information using standard file operations:

```aw wrap
Please save this information to a file in the cache folder: "User prefers verbose error messages when debugging."
```

### Retrieving Information

The AI agent can read files from the cache folder:

```aw wrap
Check what information I have stored in the cache folder from previous runs.
```

### File Organization

The AI agent can organize files as needed:

- **Structured data**: JSON or YAML files for configuration and preferences
- **Text files**: Plain text for notes, logs, and observations
- **Directories**: Organize files into subdirectories for better structure

## Advanced Configuration

You can customize cache key and artifact retention:

```aw wrap
---
engine: claude
tools:
  cache-memory:
    key: custom-memory-${{ github.workflow }}-${{ github.run_id }}
    retention-days: 30
  github:
    allowed: [get_repository]
---
```

Artifact Retention configures how long cache data artifacts are retained:

```aw wrap
---
engine: claude
tools:
  cache-memory:
    key: persistent-memory
    retention-days: 90  # Keep artifacts for 90 days (1-90 range)
  github:
    allowed: [get_repository]
---
```

The `retention-days` option controls the `actions/upload-artifact` retention period:
- **Range**: 1-90 days
- **Default**: Repository setting (if not specified)
- **Purpose**: Provides alternative access to cache data beyond cache expiration

## Cache Behavior and GitHub Actions Integration

Cache Memory leverages GitHub Actions cache with these characteristics:

#### Cache Retention
- **Retention Period**: 7 days (GitHub Actions standard)
- **Size Limit**: 10GB per repository (GitHub Actions standard)
- **LRU Eviction**: Least recently used caches are evicted when limits are reached

#### Artifact Upload (Optional)
When `retention-days` is configured, cache data is also uploaded as artifacts:
- **Retention Period**: 1-90 days (configurable via `retention-days`)
- **Purpose**: Alternative access to cache data beyond cache expiration
- **Use Case**: Long-term persistence for workflows that run infrequently

#### Cache Scoping
- **Branch Scoping**: Caches are accessible across branches in the same repository
- **Workflow Scoping**: Each workflow maintains its own cache namespace by default
- **Run Scoping**: Each run gets unique cache keys to prevent conflicts

#### Automatic Key Generation

- **Default Pattern**: `memory-${{ github.workflow }}-${{ github.run_id }}`
- **Custom Keys**: Any custom key gets `-${{ github.run_id }}` appended automatically
- **Example**: `project-memory` becomes `project-memory-${{ github.run_id }}`

#### Progressive Restore Keys

Restore keys are automatically generated by splitting the cache key on dashes, creating a fallback hierarchy:

For key `custom-memory-project-v1-${{ github.run_id }}`, restore keys are:
```
custom-memory-project-v1-
custom-memory-project-
custom-memory-
custom-
```

This ensures the most specific match is found first, with progressive fallbacks.

## File Access and Organization

The cache folder provides standard file system access:

### File Operations
```
/tmp/gh-aw/cache-memory/notes.txt           # Simple text files
/tmp/gh-aw/cache-memory/config.json         # Structured data
/tmp/gh-aw/cache-memory/logs/activity.log   # Organized in subdirectories
/tmp/gh-aw/cache-memory/state/session.yaml  # State management
```

### Best Practices for File Organization

Use descriptive file names and directory structures:

```
/tmp/gh-aw/cache-memory/
├── preferences/
│   ├── user-settings.json
│   └── workflow-config.yaml
├── logs/
│   ├── activity.log
│   └── errors.log
├── state/
│   ├── last-run.txt
│   └── context.json
└── notes.md
```

## Best Practices

### Cache Key Naming

Use descriptive, hierarchical cache keys:

```yaml
tools:
  cache-memory:
    key: project-${{ github.repository_owner }}-${{ github.workflow }}
```

### Memory Scope

Consider the scope of cache needed:

- **Workflow-specific**: Default behavior, cache per workflow
- **Repository-wide**: Use repository name in cache key
- **User-specific**: Include user information in cache key

### Resource Management

Be mindful of cache usage:

- **File Size**: Monitor cache data growth over time
- **Cache Limits**: Respect GitHub's 10GB repository cache limit
- **Cleanup Strategy**: Consider periodic cache clearing for long-running projects

## Troubleshooting

### Common Issues

#### Files Not Persisting
- **Check Cache Keys**: Ensure keys are consistent between runs
- **Verify Paths**: Confirm `/tmp/gh-aw/cache-memory/` directory exists
- **Review Logs**: Check workflow logs for cache restore/save messages

#### File Access Issues
- **Directory Creation**: Ensure subdirectories are created before use
- **Permissions**: Verify file write permissions in the cache folder
- **Path Resolution**: Use absolute paths within `/tmp/gh-aw/cache-memory/`

#### Cache Size Issues
- **Monitor Usage**: Track cache size growth over time
- **Cleanup Strategy**: Implement periodic cache clearing
- **Key Rotation**: Use time-based cache keys for automatic expiration

### Debugging

Enable verbose logging to debug cache-memory issues:

```aw wrap
---
engine: claude
tools:
  cache-memory: true
timeout_minutes: 10  # Allow time for debugging
---

# Debug Cache Memory

Please debug the cache-memory functionality by:

1. Checking what files exist in the cache folder
2. Creating a test file with current timestamp
3. Reading the test file back
4. Listing all files in the cache folder
5. Reporting on file persistence
```

## Security Considerations

### Data Privacy

- **Sensitive Data**: Avoid storing sensitive information in cache files
- **Access Control**: Cache data follows repository access permissions
- **Audit Trail**: Cache access is logged in workflow execution logs

### File Security

- **Standard Permissions**: Files use standard GitHub Actions runner permissions
- **Temporary Storage**: Cache directory is temporary and cleaned between runs
- **No External Access**: Cache folder is only accessible within workflow execution

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
  github:
    allowed: [get_repository]
---

# File Storage Test Workflow

Store and retrieve information using simple file operations.

## Task

1. Check what files exist in the cache folder from previous runs
2. Store the new note: "${{ inputs.note }}" in a timestamped file
3. List all files in the cache folder
4. Provide a summary of stored files and persistence
```

### Project-Specific Cache

```aw wrap
---
engine: claude
tools:
  cache-memory:
    key: project-docs-${{ github.repository }}-${{ github.workflow }}
  github:
    allowed: [get_repository, list_files]
---

# Documentation Assistant

Use project-specific cache to maintain context about documentation updates.
Store progress, preferences, and notes in organized files.
```

### Multi-Workflow File Sharing

```aw wrap
---
engine: claude
tools:
  cache-memory:
    key: shared-cache-${{ github.repository }}
---

# Shared Cache Workflow

Share cache data across multiple workflows in the same repository using files.
```

## Migration from MCP Memory Server

If migrating from the previous MCP memory server approach:

### Changes
- **No MCP server**: Cache-memory no longer uses `@modelcontextprotocol/server-memory`
- **File operations**: Use standard file read/write instead of memory tools
- **Direct access**: Access files directly at `/tmp/gh-aw/cache-memory/`
- **No tools**: No `mcp__memory` tool is provided

### Migration Steps
1. **Update workflow syntax**: Remove any `docker-image` configuration
2. **Modify prompts**: Update prompts to reference file operations instead of memory tools
3. **Test file access**: Verify file operations work as expected

## Related Documentation

- [Frontmatter](/gh-aw/reference/frontmatter/) - Complete frontmatter configuration guide
- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Output processing and automation
- [GitHub Actions Cache Documentation](https://docs.github.com/en/actions/using-workflows/caching-dependencies-to-speed-up-workflows) - Official GitHub cache documentation