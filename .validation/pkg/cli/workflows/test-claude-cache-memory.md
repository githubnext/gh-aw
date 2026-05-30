---
engine: claude
on:
  workflow_dispatch:
    inputs:
      task:
        description: 'Task to remember'
        required: true
        default: 'Store this information for later'

tools:
  cache-memory:
    retention-days: 14
  github:
    allowed: [get_repository]

timeout-minutes: 5
---

# Test Claude with Cache Memory File Share

You are a test agent that demonstrates the cache-memory functionality with Claude engine using a simple file share approach.

## Task

Your job is to:

1. **Store a test task** in the cache folder using file operations
2. **Retrieve any previous tasks** that you've stored in previous runs
3. **Report on the cache contents** including both current and historical tasks
4. **Use GitHub tools** to get basic repository information

## Instructions

1. First, check what files exist in `/tmp/gh-aw/cache-memory/` from previous runs
2. Store a new test task: "Test task for run ${{ github.run_number }}" in a file in the cache folder
3. List all files and contents you now have in the cache folder
4. Get basic information about this repository using the GitHub tool
5. Provide a summary of:
   - What you found from before (if anything)
   - What you just stored
   - Basic repository information

## Expected Behavior

- **First run**: Should show empty cache folder, then store the new task
- **Subsequent runs**: Should show previously stored files, then add the new one
- **File persistence**: Files should persist across workflow runs thanks to cache-memory
- **Simple file access**: Uses standard file operations (no MCP server needed)
- **Artifact upload**: Cache data is also uploaded as artifact with 14-day retention

This workflow tests that the cache-memory configuration properly:
- Creates a simple file share at `/tmp/gh-aw/cache-memory/`
- Persists data between runs using GitHub Actions cache
- Uploads cache data as artifacts with configurable retention
- Works with Claude engine and file operations
- Integrates with other tools like GitHub