---
description: Test ProjectOps PAT requirements documentation for user-owned and org-owned Projects v2
on:
  workflow_dispatch:
  schedule: weekly on monday
permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
name: Test ProjectOps PAT Requirements
engine: copilot
timeout-minutes: 30
tools:
  bash:
    - "*"
  github:
    mode: remote
    toolsets: [default]
safe-outputs:
  create-issue:
    max: 1
    expires: 7d
    labels: [documentation, projectops, testing]
---

# Test ProjectOps PAT Requirements Documentation

This workflow validates the PAT setup documentation for GitHub Projects v2 by creating temporary trial repositories, testing different PAT configurations, and verifying the documented requirements are accurate.

## Test Objectives

1. **Verify User-owned Projects PAT Requirements**
   - Classic PAT with `project` scope works
   - Fine-grained PAT does NOT work (as documented)

2. **Verify Organization-owned Projects PAT Requirements**
   - Classic PAT with `project` + `read:org` works
   - Fine-grained PAT with explicit org access + Projects: Read+Write works
   - Fine-grained PAT without explicit org access fails (as documented)

3. **Validate Documentation Accuracy**
   - Confirm all setup instructions are correct
   - Verify scope requirements match actual API behavior
   - Test both public and private repository scenarios

## Test Execution Plan

**IMPORTANT**: This is a validation test. Do NOT create actual trial repositories or test real PATs. Instead:

1. **Review Documentation**: Read the current ProjectOps PAT documentation at:
   - `docs/src/content/docs/reference/tokens.md` (GH_AW_PROJECT_GITHUB_TOKEN section)
   - `docs/src/content/docs/examples/issue-pr-events/projectops.md` (Token Requirements section)

2. **Verify Documented Scopes Match GitHub API Requirements**: 
   - Check GitHub's official Projects v2 API documentation
   - Confirm the documented scopes align with actual API permissions
   - Verify the distinction between user and org Projects is accurate

3. **Check for Consistency**: 
   - Ensure all three documentation files have consistent information
   - Verify links between documents are correct
   - Check that examples match the documented requirements

4. **Identify Gaps or Errors**:
   - Are there any missing scenarios or edge cases?
   - Are the instructions clear enough to follow?
   - Are there any contradictions between documents?

## Test Results

Create an issue with your findings:

**Title**: "ProjectOps PAT Documentation Validation - [PASS/FAIL] - $(date +%Y-%m-%d)"

**Body** (use markdown format):

### Documentation Review

- **Consistency Check**: [PASS/FAIL]
  - Note: [Explanation if any inconsistencies found]

- **Scope Accuracy**: [PASS/FAIL]  
  - User-owned Projects requirements: [Verified/Issues found]
  - Org-owned Projects requirements: [Verified/Issues found]

- **Clarity and Completeness**: [PASS/FAIL]
  - Missing information: [List if any]
  - Unclear instructions: [List if any]

### Recommendations

[List any improvements or clarifications needed]

### GitHub API Documentation Cross-Reference

[Link to relevant GitHub API docs that confirm the requirements]

---

**Test Metadata**:
- Workflow run: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}
- Repository: ${{ github.repository }}
- Test date: $(date -u +"%Y-%m-%d %H:%M:%S UTC")

**Labels**: The issue will automatically have `documentation`, `projectops`, and `testing` labels applied.

## Notes

This test validates documentation accuracy without requiring actual PAT testing, which would require:
- Creating GitHub Apps or PATs with various permission combinations
- Setting up test organizations and user accounts
- Creating and destroying trial repositories
- Managing API rate limits and quota

For actual integration testing with real PATs, use the `gh aw trial` command with appropriate test repositories as described in the TrialOps guide.
