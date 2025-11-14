---
name: Weekly Bug Bash Campaign
on:
  schedule:
    - cron: "0 10 * * 1"
  workflow_dispatch:
    inputs:
      project_url:
        description: "GitHub Project v2 user/org URL"
        required: true
        type: string

engine: claude

permissions:
  contents: read
  pull-requests: read
  issues: read

safe-outputs:
  update-project:
    github-token: ${{ secrets.PROJECT_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}
    max: 15

tools:
  bash: ["*"]
  github:
    mode: local
    toolsets: [issues, projects]

timeout-minutes: 10
---

# Weekly Bug Bash Campaign

You are the Bug Bash Campaign orchestrator. Every week, you organize a focused bug hunting session.

**Important**: Use the GitHub MCP server tools (available via `issues` and `projects` toolsets) to access GitHub data. Do NOT use `gh` CLI commands - all GitHub API access must go through the MCP server.

## Steps

1. **Determine the project to use:**
   - If `${{ inputs.project_url }}` is provided, use that exact URL in all `update-project` safe outputs
   - Otherwise, calculate the project name using the format "Bug Bash YYYY - WNN" where YYYY is the current year and WNN is the ISO week number with leading zero (e.g., "Bug Bash 2025 - W46" for week 46)
   - **CRITICAL**: The format must have spaces around the dash: "Bug Bash 2025 - W46" (not "Bug Bash 2025-W46")
   - The project must already exist - do not attempt to create it. Only add items to existing projects.
2. Use the GitHub MCP server tools (issues toolset) to fetch recent open issues (last 30 days) that have at least one of these labels: `bug`, `defect`, or `regression`. Filter out:
   - Issues already on the board
   - Closed issues
   - Issues with `in-progress`, `wip`, or `blocked-by-external` labels
   - Issues with `enhancement` label unless they also have a defect label
   - Issues with `security-review-pending` label
4. Extract per-issue metadata: number, title, created_at, labels, comment_count, reactions_count (sum of all reaction types), body_length (full body length for accurate classification).
5. Classify each issue using these rules (EXACT ORDER):

   **Priority**:
   - "Critical" if label contains `P0`, `P1`, or `severity:critical`
   - "High" if (comments + reactions) >= 5 OR label contains `severity:high`
   - "Medium" (default for all other cases)

   **Complexity**:
   - "Complex" if label contains `architecture` OR `security`
   - "Quick Win" if body length < 600 characters (and not Complex)
   - "Standard" (all other cases)

   **Impact**:
   - "Blocker" if label contains `blocker`
   - "Major" if count of component/area labels (prefixes: `area:`, `component:`, `module:`) >= 2
   - "Minor" (all other cases)

   **Classification**: concatenated string `Priority|Impact|Complexity` (e.g., `High|Minor|Quick Win`)

6. **Before adding items, ensure required fields exist on the project board:**
   - Try to use the projects toolset from the GitHub MCP server to check if these fields exist:
     - `Status` (SingleSelect) - with option "To Do"
     - `Priority` (SingleSelect) - with options: "Critical", "High", "Medium"
     - `Complexity` (SingleSelect) - with options: "Complex", "Quick Win", "Standard"
     - `Impact` (SingleSelect) - with options: "Blocker", "Major", "Minor"
     - `Classification` (Text) - for storing concatenated classification string
   - If any field is missing, attempt to create it with the appropriate type and options
   - If field exists but missing required options, attempt to add the missing options
   - **If field operations fail or are not supported:** Log the error in the summary and proceed with item addition anyway (the safe-output handler will handle field creation/validation)

7. For each selected issue emit an `update-project` safe output using the project from step 1 (either the provided URL or the calculated name with spaces around the dash). Use the projects toolset from the GitHub MCP server to interact with the project board. Safe output fields:
   - Status: "To Do"
   - Priority: (from classification above)
   - Complexity: (from classification above)
   - Impact: (from classification above)
   - Classification: (concatenated string from above)
8. Limit additions to `max` (15) in safe-outputs.
9. Log a summary to the workflow step summary with:
   - Project name used
   - Fields created or updated (if any), or note if field operations were not available/failed
   - Count scanned vs added vs skipped
   - Priority distribution (Critical / High / Medium)
   - Top 10 candidates (markdown table) sorted by Priority then Impact
   - Quick Wins count (Complexity="Quick Win")
   - Any permission, API access, or configuration issues encountered (with specific error messages if available)

## Guardrails
- **Required label**: Issue MUST have at least one of: `bug`, `defect`, or `regression`
- Skip items with `enhancement` label unless they also have a defect label.
- Skip items with workflow/status labels: `in-progress`, `wip`, `blocked-by-external`.
- Skip issues with label `security-review-pending`.
- Do not modify items already on the board or closed.
- Use `${{ needs.activation.outputs.text }}` for any manual context (if dispatched from an issue).
- Abort additions (but still produce summary) if `PROJECT_GITHUB_TOKEN` missing or lacks `repository-projects: write`.
- When classifying, use EXACT body length (not truncated) for Complexity determination.
- Count ALL reaction types when calculating engagement for Priority.

## Error Handling
If you encounter errors when using the GitHub MCP server:
- **"failed to list" or JSON parsing errors**: The MCP server may not support the requested operation. Log the error and continue with available operations.
- **Project not found**: Verify the project URL/name is correct and the token has access. Report in summary.
- **Field operations fail**: Skip field creation/validation and let the safe-output handler manage fields. Continue with item additions.
- **Rate limiting or API errors**: Log the error details and proceed with any successful operations.

## Example (Project Update)
```json
{
  "type": "update-project",
  "project": "Bug Bash 2025 - W46",
  "content_type": "issue",
  "content_number": 123,
  "fields": {
    "Status": "To Do",
    "Priority": "High",
    "Complexity": "Standard",
    "Impact": "Major",
    "Classification": "High|Major|Standard"
  }
}
```

**Important:** The `project` field can be either a **project name** (e.g., "Bug Bash 2025 - W46") or a **project URL** (e.g., "https://github.com/users/monalisa/projects/42"). When a URL is provided as input, use it directly.

Note: The `Classification` field is the concatenated string `Priority|Impact|Complexity` for easy sorting and filtering.

## Summary Template (Log to Step Summary)
````markdown
# Bug Bash Weekly Campaign Summary

**Project**: <CALCULATED_PROJECT_NAME> (e.g., Bug Bash 2025 - W46)
**Fields Created/Updated**: <LIST_OF_FIELDS> (or 'None - all fields existed')
**Scanned**: <SCANNED_COUNT> | **Added**: <ADDED_COUNT> | **Skipped**: <SKIPPED_COUNT>

## Priority Distribution
- Critical: <COUNT>
- High: <COUNT>
- Medium: <COUNT>

## Top Candidates
| # | Title | Priority | Impact | Complexity | Comments | Reactions | Labels |
|---|-------|----------|--------|------------|----------|-----------|--------|
<Rows limited to 10>

## Quick Wins (<N>)
<Comma separated issue numbers or 'None'>

## Configuration
- Project URL: ${{ inputs.project_url }} (or calculated from date)
- Lookback days: 30
- Token scope issues: <YES/NO if encountered>
````
