---
on:
  push:
    branches: [main]
  pull_request:
    types: [ready_for_review]
  workflow_dispatch:

permissions:
  contents: read
  models: read
  issues: read
  pull-requests: read
  actions: read
  checks: read
  statuses: read
  discussions: read

tools:
  github:
    allowed: [
      list_issues,
      get_issue,
      get_issue_comments,
      search_issues,
      list_pull_requests,
      get_pull_request,
      get_pull_request_comments,
      list_commits,
      get_commit,
      get_file_contents,
      list_branches,
      search_code,
      list_workflows,
      get_workflow_run,
      list_workflow_runs,
      get_job_logs,
      list_workflow_jobs
    ]
  claude:
    allowed:
      Task:
      Read:
      Write:
      Bash: ["echo", "date", "curl --version"]

timeout_minutes: 10
---

# GitHub Agentic Workflows Integration Test

This workflow serves as a comprehensive integration test for the GitHub Agentic Workflows (gh-aw) tool. It exercises most available GitHub MCP features and validates connectivity to the GitHub API.

## Test Objectives

1. **GitHub Issues API Testing**: Query and retrieve the last 2 issues from this repository
2. **Repository Information Testing**: Test various repository access operations
3. **Workflow Management Testing**: Test workflow-related API operations
4. **Feature Coverage Testing**: Exercise a broad range of GitHub MCP tools
5. **Error Handling**: Fail clearly if operations cannot be performed

## Integration Test Execution

### Phase 1: GitHub Issues API Testing

**Critical Test**: Query the last 2 issues from this repository using GitHub MCP.

First, let me retrieve the most recent issues from this repository:

1. List the most recent issues in the repository
2. Get detailed information for the last 2 issues
3. Retrieve comments for those issues if they exist

If this fails, the entire test should fail as it's the core requirement.

### Phase 2: Repository Information Testing

Test various repository access operations:

1. Get repository file contents (README.md)
2. List recent commits
3. Get details of the latest commit
4. List repository branches
5. Search for code patterns in the repository

### Phase 3: Pull Request Testing

Test pull request operations:

1. List recent pull requests
2. Get details of recent PRs if available
3. Test PR comment retrieval

### Phase 4: Workflow Management Testing

Test workflow-related operations:

1. List available GitHub Actions workflows
2. Get recent workflow runs
3. Test workflow job information retrieval

### Phase 5: Search and Discovery Testing

Test search capabilities:

1. Search for issues with specific patterns
2. Search for code in the repository
3. Test advanced search operations

## Execution Script

Let me execute these tests systematically:

**Step 1: Test GitHub Issues API (Critical Requirement)**

```
Testing GitHub Issues API connectivity and functionality...
```

Using the `list_issues` tool to get the most recent issues from this repository. This is the critical test that must succeed.

Using the `get_issue` tool to retrieve detailed information for the last 2 issues.

If issues are found, I'll also test `get_issue_comments` to retrieve any comments.

**Step 2: Repository Information Tests**

Testing repository access using `get_file_contents` to read the README.md file.

Testing commit history using `list_commits` to get recent commits.

Testing branch information using `list_branches`.

**Step 3: Workflow Operations Tests**

Testing workflow information using `list_workflows` to get available workflows.

Testing workflow runs using `list_workflow_runs` for recent executions.

**Step 4: Search Operations Tests**

Testing issue search using `search_issues` with various patterns.

Testing code search using `search_code` to find specific code patterns.

## Test Results and Reporting

At the end of each test phase, I will report:

- ✅ **SUCCESS**: Operation completed successfully
- ❌ **FAILURE**: Operation failed (will cause workflow failure)
- ⚠️ **WARNING**: Operation had issues but didn't fail

### Final Integration Test Report

I will provide a comprehensive summary including:

1. **GitHub API Connectivity**: Status of GitHub MCP connection
2. **Issues API Test Results**: Success/failure of the critical requirement
3. **Feature Coverage**: Which GitHub MCP tools were successfully tested
4. **Performance Metrics**: Response times and operation success rates
5. **Error Analysis**: Any failures encountered and their causes

**Integration Test Result**: The workflow will exit with failure status if any critical operations (especially the GitHub Issues API requirement) fail.

## Workflow Validation

This workflow validates that:

- ✅ GitHub MCP server connectivity is working
- ✅ Authentication and permissions are properly configured
- ✅ Core GitHub API operations are functional
- ✅ The gh-aw tool is properly integrated with GitHub Actions
- ✅ The last 2 issues can be successfully queried and retrieved

If all tests pass, this confirms that the GitHub Agentic Workflows tool is functioning correctly and can be used for production workflows.

---

**Note**: This integration test workflow is designed to be comprehensive but safe - it only performs read operations and does not modify any repository data.