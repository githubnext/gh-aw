---
description: Daily test of GitHub remote MCP authentication with GitHub Actions token
on:
  schedule: daily
  workflow_dispatch:
permissions:
  contents: read
  issues: read
tools:
  github:
    mode: remote
    toolsets: [repos, issues]
safe-outputs:
  create-issue:
    title-prefix: "[auth-test] "
    labels: [test, automated]
    expires: 7d
    max: 1
  close-issue:
    required-title-prefix: "[auth-test]"
    required-labels: [test]
    max: 10
timeout-minutes: 5
strict: true
---

# GitHub Remote MCP Authentication Test

You are an automated testing agent that verifies GitHub remote MCP server authentication with the GitHub Actions token.

## Your Task

Test that the GitHub remote MCP server can authenticate and access GitHub API with the GitHub Actions token. Also close any older test failure issues to keep the repository clean.

### Test Procedure

1. **Close Old Test Issues**: Before running the test, close any previous test failure issues
   - Search for open issues with title prefix "[auth-test]" and label "test"
   - Close up to 10 old test failure issues with a comment: "Closing old test issue. New test run in progress."
   - This keeps the repository clean and prevents accumulation of stale test issues

2. **List Open Issues**: Use the GitHub MCP server to list 3 open issues in the repository ${{ github.repository }}
   - Use the `list_issues` tool or equivalent
   - Filter for `state: OPEN`
   - Limit to 3 results
   - Extract issue numbers and titles

3. **Verify Authentication**: 
   - If the MCP tool successfully returns issue data, authentication is working correctly
   - If the MCP tool fails with authentication errors (401, 403, or "unauthorized"), authentication has failed

### Success Case

If the test succeeds (issues are retrieved successfully):
- Output a brief success message with:
  - ✅ Authentication test passed
  - Number of issues retrieved
  - Sample issue numbers and titles
- **Do NOT create an issue** - the test passed

### Failure Case

If the test fails (authentication error or MCP tool unavailable):
- Create an issue using safe-outputs with:
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
- **Only create issue on failure**: Don't create issues when the test passes
- **Include error details**: If authentication fails, include the exact error message
- **Auto-cleanup**: Old test failure issues will be automatically closed after 7 days

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
Create an issue with the error details as described above.
