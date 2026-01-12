# Analysis of Campaign Creation Run Issue

## Problem Statement
Workflow run https://github.com/githubnext/gh-aw/actions/runs/20908422378 had:
1. Safe outputs being skipped
2. 2 blocked network requests (mentioned in issue)

## Root Cause Analysis

### Issue 1: Safe Outputs Skipped

**Root Cause**: The Copilot agent wrote safe outputs in markdown code fence format instead of making actual MCP tool calls.

**Evidence**:
- Log shows: "Output file does not exist: /tmp/gh-aw/safeoutputs/outputs.jsonl"
- Agent output shows it wrote code fences like:
  ```
  ```create_project
  {
    "title": "Campaign: Lint Guardian",
    "owner": "",
    "item_url": "https://github.com/githubnext/gh-aw/issues/9685"
  }
  ```
  ```
- MCP gateway logs confirm tools were available (safeoutputs→tools/list succeeded)
- Safe outputs job was skipped because no outputs.jsonl file was created

**Why This Happened**:
The prompt instructions in `pkg/workflow/prompts.go` said:
> "you MUST call the appropriate safe output tool. Simply writing content will NOT work - the workflow requires actual tool calls."

But this was ambiguous. The agent interpreted "tool call" as writing JSON in markdown code fences (a common pattern for some AI assistants), rather than making an actual MCP tool invocation.

**Fix Applied**:
Updated the prompt instructions to be explicit:
```
To create or modify GitHub resources (issues, discussions, pull requests, etc.), you MUST use the MCP tool calling mechanism to invoke the safe output tools. Do NOT write markdown code fences or JSON - you must make actual MCP tool calls.

**Available MCP tools**: add_comment, assign_to_agent, create_project, missing_tool, noop, update_issue

**How to use**: Call the tools using your MCP tool calling capability. For example, to create a project, invoke the create_project tool with the required parameters.

**Critical**: MCP tool calls write structured data that downstream jobs process. Without proper MCP tool invocations, follow-up actions will be skipped.
```

Key changes:
- Explicitly mentions "MCP tool calling mechanism"
- Adds explicit prohibition: "Do NOT write markdown code fences or JSON"
- Adds "How to use" section with example
- Changes "Available tools" to "Available MCP tools" for clarity

### Issue 2: Blocked Network Requests

**Status**: Unable to verify from available logs. The firewall logs were not accessible in the analysis. However, the fix to safe outputs should indirectly help:

1. With proper MCP tool calls, the safe outputs will be processed correctly
2. This may have been related to the agent trying to make direct GitHub API calls when safe outputs didn't work
3. The MCP gateway was running correctly (logs show successful connection)

## Impact

The fix was applied to:
- `pkg/workflow/prompts.go` - Updated safe output prompt generation
- All 113 workflow `.lock.yml` files that use safe outputs

This ensures consistent behavior across all Copilot engine workflows with safe outputs.

## Testing

1. ✅ Code compiled successfully with `make build`
2. ✅ Campaign-generator workflow recompiled successfully
3. ✅ All 118 workflows recompiled successfully with `make recompile`
4. ✅ Code formatted with `make fmt`
5. ⏳ Future workflow runs will validate the fix in production

## Next Steps

1. Monitor future campaign creation runs to verify safe outputs work correctly
2. If blocked requests persist, analyze firewall logs from future runs
3. Consider adding explicit examples of MCP tool usage to documentation

## Files Modified

### Source Code
- `pkg/workflow/prompts.go` - Updated safe output instructions

### Compiled Workflows (113 files)
All `.lock.yml` files with safe outputs were recompiled with the updated instructions.

## Commits

1. `f047829` - Fix safe output prompt to clarify MCP tool calling
2. `223178a` - Recompile all workflows with updated safe output instructions
