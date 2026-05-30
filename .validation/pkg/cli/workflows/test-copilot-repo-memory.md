---
engine: copilot
on:
  workflow_dispatch:
    inputs:
      task:
        description: 'Task to remember'
        required: true
        default: 'Store this information for later'

tools:
  repo-memory:
    branch-name: memory/test-agent
    description: "Test repo-memory persistence"
    max-file-size: 524288  # 512KB
    max-file-count: 10
  github:
    allowed: [get_repository]

timeout-minutes: 5
---

# Test Copilot with Repo Memory Git-Based Storage

You are a test agent that demonstrates the repo-memory functionality with Copilot engine using git-based persistent storage.

## Task

Your job is to:

1. **Store a test task** in the repo-memory folder using file operations
2. **Retrieve any previous tasks** that you've stored in previous runs
3. **Report on the memory contents** including both current and historical tasks
4. **Use GitHub tools** to get basic repository information

## Instructions

1. First, check what files exist in `/tmp/gh-aw/repo-memory-default/memory/default/` from previous runs
2. Store a new test task: "Test task for run ${{ github.run_number }}" in a file in the memory folder
3. List all files and contents you now have in the memory folder
4. Get basic information about this repository using the GitHub tool
5. Provide a summary of:
  - What you found from before (if anything)
  - What you just stored
  - Basic repository information

## Expected Behavior

- **First run**: Should show empty memory folder (or new orphan branch created), then store the new task
- **Subsequent runs**: Should show previously stored files from git branch, then add the new one
- **File persistence**: Files persist across workflow runs via git branch storage
- **Version control**: All changes are committed to the `memory/test-agent` branch
- **Automatic push**: Changes are automatically committed and pushed after workflow completion
- **Conflict resolution**: Current version wins in case of merge conflicts

This workflow tests that the repo-memory configuration properly:
- Clones the git branch at workflow start (creates orphan branch if needed)
- Provides simple file access at `/tmp/gh-aw/repo-memory-default/memory/default/`
- Persists data between runs using git branch storage
- Commits and pushes changes automatically at workflow end
- Works with Copilot engine and file operations
- Integrates with other tools like GitHub
