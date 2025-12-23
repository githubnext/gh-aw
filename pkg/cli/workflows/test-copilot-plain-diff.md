---
on:
  workflow_dispatch:
    inputs:
      pull_request_number:
        description: 'Pull request number to test with (optional)'
        required: false
permissions:
  contents: read
  pull-requests: read
engine:
  id: copilot
  args:
    - --plain-diff
safe-outputs:
  noop:
    max: 1
timeout-minutes: 5
---

# Test Copilot CLI --plain-diff Flag

This workflow tests the new `--plain-diff` flag introduced in Copilot CLI 0.0.370.

## Test Scenarios

1. **Default Behavior (Baseline)**
   - Without --plain-diff, diffs use rich rendering with syntax highlighting
   - This is the standard behavior

2. **Plain Diff Mode**
   - With --plain-diff flag, rich rendering is disabled
   - Uses git-configured diff tool instead
   - No syntax highlighting or color formatting

3. **Use Cases**
   - CI/CD pipelines that parse diff output
   - Integration with external diff tools
   - Simplified output for logging systems
   - Compatibility with git configuration (e.g., custom diff.tool)

## Test Instructions

Run this test with a pull request that has code changes:

1. Verify the --plain-diff flag is present in the Copilot CLI command
2. Check that diff output is plain text without rich formatting
3. Confirm it respects git diff configuration if set
4. Test with various file types (Go, JavaScript, Markdown, etc.)

## Expected Output

The workflow should successfully execute with the --plain-diff flag, and any diff operations should use plain text format without syntax highlighting or rich rendering features.

Generate a noop output confirming the test ran successfully.
