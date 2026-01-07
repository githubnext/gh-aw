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

1. **List Open Issues**: Use the GitHub MCP server to list 3 open issues in the repository ${{ github.repository }}
   - Use the `list_issues` tool or equivalent
   - Filter for `state: OPEN`
   - Limit to 3 results
   - Extract issue numbers and titles

2. **Verify Authentication**: 
   - If the MCP tool successfully returns issue data, authentication is working correctly
   - If the MCP tool fails with authentication errors (401, 403, or "unauthorized"), authentication has failed

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
    [Include the specific error message from the MCP tool]
    
    ### Expected Behavior
    The GitHub remote MCP server should authenticate with the GitHub Actions token and successfully list open issues.
    
    ### Actual Behavior
    [Describe what happened - authentication error, timeout, tool unavailable, etc.]
    
    ### Test Configuration
    - Repository: ${{ github.repository }}
    - Workflow: ${{ github.workflow }}
    - Run: ${{ github.run_id }}
    - Time: $(date -u +"%Y-%m-%d %H:%M:%S UTC")
    
    ### Next Steps
    1. Verify GitHub Actions token permissions
    2. Check GitHub remote MCP server availability
    3. Review workflow logs for detailed error information
    4. Test with local mode as fallback if remote mode continues to fail
    ```

## Guidelines

- **Be concise**: Keep output brief and focused
- **Test quickly**: This should complete in under 1 minute
- **Only create discussion on failure**: Don't create discussions when the test passes
- **Include error details**: If authentication fails, include the exact error message
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
