---
engine: claude
on:
  workflow_dispatch:
    inputs:
      missing_tool:
        description: 'Tool to report as missing'
        required: true
        default: 'example-missing-tool'

tools:
  cache-memory: true
  github:
    allowed: [get_repository]

safe-outputs:
  missing-tool:
    max: 5
  staged: true

timeout-minutes: 5
---

# Test Claude with Missing Tool Safe Output and Cache Memory

You are a test agent that demonstrates the missing-tool safe output functionality with Claude engine, enhanced with persistent memory.

## Task

Your job is to:

1. **Check your memory** for any previous missing tool reports
2. **Report a missing tool** using the safe output functionality
3. **Store the report in memory** for future reference
4. **Use GitHub tools** to get basic repository information

## Instructions

1. First, check your memory to see if you've reported any missing tools before
2. Report that the tool specified in the input (${{ github.event.inputs.missing_tool }}) is missing
3. Use the safe output functionality to properly report the missing tool with:
   - Tool name: ${{ github.event.inputs.missing_tool }}
   - Reason: "This tool is not available in the current workflow environment"
   - Alternatives: "Consider using alternative tools or requesting tool addition"
   - Context: "Testing missing-tool safe output with Claude engine"
4. Store this report in your memory for future reference
5. Get basic information about this repository using the GitHub tool
6. Provide a summary of your actions

## Expected Behavior

- **Memory persistence**: Should remember previous missing tool reports across runs
- **Safe output**: Should generate proper missing-tool JSON output
- **Tool integration**: Should work with GitHub tools despite the "missing" tool
- **Staged mode**: Should prevent actual GitHub interactions when staged: true

This workflow tests that:
- Claude can properly report missing tools using safe outputs
- Cache-memory preserves missing tool reports across runs
- Integration works with existing tools like GitHub
- Safe output system handles missing tool scenarios correctly