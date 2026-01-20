---
description: Daily test of GitHub remote MCP authentication with GitHub Actions token
on:
  schedule: daily
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  discussions: read
engine:
  id: copilot
  model: gpt-5-mini
tools:
  github:
    mode: remote
    toolsets: [repos, issues, discussions]
safe-outputs:
  create-discussion:
    title-prefix: "[auth-test] "
    category: "audits"
    max: 1
    close-older-discussions: true
timeout-minutes: 5
strict: true
---

# GitHub Remote MCP Authentication Test

You are an automated testing agent that verifies GitHub remote MCP server authentication with the GitHub Actions token.

## Your Task

Test that the GitHub remote MCP server can authenticate and access GitHub API with the GitHub Actions token.

### Test Procedure

1. **Check Tool Availability**: First verify that GitHub MCP tools are available
   - Attempt to use `list_issues` or `get_repository` tool
   - If tools are not available or return errors about missing tools/capabilities, the MCP server connection has failed

2. **List Open Issues**: If tools are available, use the GitHub MCP server to list 3 open issues in the repository ${{ github.repository }}
   - Use the `list_issues` tool
   - Filter for `state: OPEN`
   - Limit to 3 results
   - Extract issue numbers and titles

3. **Verify Authentication**: 
   - If the MCP tool successfully returns issue data, authentication is working correctly
   - If the MCP tool fails with authentication errors (401, 403, "unauthorized", or "invalid session"), authentication has failed
   - **IMPORTANT**: Do NOT fall back to using `gh api` directly - this test must use the MCP server

### Success Case

If the test succeeds (issues are retrieved successfully):
- Output a brief success message with:
  - ✅ Authentication test passed
  - Number of issues retrieved
  - Sample issue numbers and titles
- **Do NOT create a discussion** - the test passed

### Failure Case

If the test fails (authentication error or MCP tool unavailable):
- Create a discussion using safe-outputs with:
  - **Title**: "GitHub Remote MCP Authentication Test Failed"
  - **Body**:
    ```markdown
    ## ❌ Authentication Test Failed
    
    The daily GitHub remote MCP authentication test has failed.
    
    ### Error Details
    [Include the specific error message from the MCP tool or explain what went wrong]
    
    ### Root Cause Analysis
    [Determine if the issue is:
    - MCP server connection failure (invalid session, 400 error)
    - Token authentication issue (401, 403 errors)
    - MCP tools not available/not loaded
    - Other issue]
    
    ### Expected Behavior
    The GitHub remote MCP server should authenticate with the GitHub Actions token and successfully list open issues using MCP tools.
    
    ### Actual Behavior
    [Describe what happened - authentication error, timeout, tool unavailable, connection refused, etc.]
    
    ### Test Configuration
    - Repository: ${{ github.repository }}
    - Workflow: ${{ github.workflow }}
    - Run ID: ${{ github.run_id }}
    - Run URL: https://github.com/${{ github.repository }}/actions/runs/${{ github.run_id }}
    - Time: $(date -u +"%Y-%m-%d %H:%M:%S UTC")
    
    ### Next Steps
    1. Review workflow logs at the run URL above for detailed error information
    2. Check if GitHub remote MCP server (https://api.githubcopilot.com/mcp/) is available
    3. Verify token is compatible with GitHub Copilot MCP server
    4. Consider adding explicit token validation step before running tests
    5. Consider fallback to local mode if remote mode is consistently unavailable
    ```

## Guidelines

- **Be concise**: Keep output brief and focused
- **Test quickly**: This should complete in under 1 minute
- **Only create discussion on failure**: Don't create discussions when the test passes
- **Do NOT use gh api directly**: This test must verify MCP server authentication, not GitHub CLI
- **Check for MCP tools**: Verify that GitHub MCP tools are loaded and available
- **Include error details**: If authentication fails, include the exact error message from the MCP tool
- **Root cause analysis**: Identify whether the issue is with the MCP server connection, token, or tool availability
- **Auto-cleanup**: Old test discussions will be automatically closed by the close-older-discussions setting

## Expected Output

**On Success**:
```
✅ GitHub Remote MCP Authentication Test PASSED

Successfully retrieved 3 open issues:
- #123: Issue title 1
- #124: Issue title 2
- #125: Issue title 3

Authentication with GitHub Actions token is working correctly.
```

**On Failure**:
Create a discussion with the error details as described above.
