## Workflow Failure

**Status:** Failed  
**Workflow:** [{workflow_name}]({workflow_source_url})  
**Run URL:** {run_url}{pull_request_info}

## Root Cause

The agentic workflow has encountered a failure. This indicates a configuration error, runtime issue, or missing dependencies that must be resolved.

## Action Required

**Agent Assignment:** This issue should be debugged using the `agentic-workflows` agent.

**Instructions for Agent:**

1. Analyze the workflow run logs at: {run_url}
2. Identify the specific failure point and error messages
3. Determine the root cause (configuration, missing tools, permissions, etc.)
4. Propose specific fixes with code changes or configuration updates
5. Validate the fix resolves the issue

**Agent Invocation:**
```
/agent agentic-workflows
```
When prompted, instruct the agent to debug this workflow failure.

## Expected Outcome

- Root cause identified and documented
- Specific fix provided (code changes, configuration updates, or dependency additions)
- Verification that the fix resolves the failure
