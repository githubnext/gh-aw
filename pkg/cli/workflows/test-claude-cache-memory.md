---
engine: claude
on:
  workflow_dispatch:
    inputs:
      task:
        description: 'Task to remember'
        required: true
        default: 'Store this information for later'

cache-memory:
  docker-image: "ghcr.io/modelcontextprotocol/server-memory:v1.0.0"

tools:
  github:
    allowed: [get_repository]

timeout_minutes: 5
---

# Test Claude with Cache Memory and Custom Docker Image

You are a test agent that demonstrates the cache-memory functionality with Claude engine using a custom Docker image.

## Task

Your job is to:

1. **Store a test task** in your memory using the memory MCP server
2. **Retrieve any previous tasks** that you've stored in memory
3. **Report on the memory contents** including both current and historical tasks
4. **Use GitHub tools** to get basic repository information

## Instructions

1. First, use the memory tool to see what you already know from previous runs
2. Store a new test task: "Test task for run ${{ github.run_number }}" in your memory
3. List all tasks you now have in memory
4. Get basic information about this repository using the GitHub tool
5. Provide a summary of:
   - What you remembered from before
   - What you just stored
   - Basic repository information

## Expected Behavior

- **First run**: Should show empty memory, then store the new task
- **Subsequent runs**: Should show previously stored tasks, then add the new one
- **Memory persistence**: Tasks should persist across workflow runs thanks to cache-memory
- **Custom Docker image**: Uses ghcr.io/modelcontextprotocol/server-memory:v1.0.0 instead of latest

This workflow tests that the cache-memory configuration properly:
- Mounts the memory MCP server with custom Docker image
- Persists data between runs using GitHub Actions cache
- Works with Claude engine and MCP tools
- Integrates with other tools like GitHub