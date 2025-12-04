---
on: workflow_dispatch
permissions:
  contents: read
  actions: read
engine: copilot
tools:
  repo-memory:
    branch-name: memory/test-agent
    description: "Test repo-memory persistence"
    max-file-size: 524288  # 512KB
    max-file-count: 10
timeout-minutes: 5
---

# Test Repo Memory

Test the repo-memory tool functionality for git-based persistent storage.

## Task

1. Check if a notes file exists at `/tmp/gh-aw/repo-memory-default/memory/default/test-notes.txt`
2. If it exists, read it and add a new line with the current timestamp
3. If it doesn't exist, create it with an initial message and timestamp
4. Also create or update a JSON file at `/tmp/gh-aw/repo-memory-default/memory/default/test-data.json` with:
   - A counter that increments on each run
   - The current timestamp
   - A list of previous run timestamps

## Expected Behavior

- Files should persist across workflow runs
- The notes file should accumulate lines over multiple runs
- The JSON counter should increment on each run
- Changes should be automatically committed and pushed to the memory/test-agent branch

## Verification

After the workflow completes:
- Check the memory/test-agent branch exists
- Verify files are stored under memory/default/ directory
- Confirm changes are committed with proper messages
