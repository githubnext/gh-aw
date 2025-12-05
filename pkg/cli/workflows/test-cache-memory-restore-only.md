---
engine: copilot
on:
  workflow_dispatch:
    inputs:
      task:
        description: 'Task to test'
        required: true
        default: 'Test restore-only cache'

tools:
  cache-memory:
    - id: default
      key: memory-default
      description: 'Normal cache that can read and write'
    - id: readonly
      key: memory-readonly
      restore-only: true
      description: 'Read-only cache that only restores data'
  github:
    allowed: [get_repository]

timeout-minutes: 5
---

# Test Cache Memory with Restore-Only Flag

You are a test agent that demonstrates the cache-memory restore-only functionality.

## Task

Your job is to:

1. **Check the default cache** at `/tmp/gh-aw/cache-memory/` - you can read and write to this cache
2. **Check the readonly cache** at `/tmp/gh-aw/cache-memory-readonly/` - you can only read from this cache (it won't be saved back)
3. **Write a test file** to the default cache folder
4. **Try to understand** that the readonly cache is for reading only - any changes won't persist
5. **Report** on what you found in both caches

## Instructions

1. First, check what files exist in both cache folders from previous runs
2. Store a new test file in the default cache: "Test file for run ${{ github.run_number }}"
3. List all files in both cache folders
4. Provide a summary of:
   - What you found in the default cache (read-write)
   - What you found in the readonly cache (restore-only)
   - What you just stored in the default cache
   - Note that changes to readonly cache won't persist

## Expected Behavior

- **Default cache**: Should persist data across runs (both restore and save)
- **Readonly cache**: Should only restore data (no save step generated)
- **File persistence**: Files in default cache persist, readonly cache is for reading only
- **Use case**: Readonly caches are useful for shared reference data that shouldn't be modified

This workflow tests that the restore-only flag:
- Properly uses actions/cache/restore instead of actions/cache
- Skips the upload-artifact step for restore-only caches
- Allows agents to read from caches without saving them back
