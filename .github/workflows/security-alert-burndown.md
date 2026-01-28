---
name: Security Alert Burndown
description: Discovers security work items (Dependabot PRs, code scanning alerts, secret scanning alerts)
on:
  #schedule:
  #  - cron: "0 * * * *"
  workflow_dispatch:
permissions:
  issues: read
  pull-requests: read
  contents: read
  security-events: read
tools:
  github:
    toolsets: [repos, issues, pull_requests]
safe-outputs:
  update-project:
    max: 100
  create-agent-session:
    base: main
    max: 3
project: https://github.com/orgs/githubnext/projects/144
---

# Security Alert Burndown

This workflow discovers security alert work items in the githubnext/gh-aw repository and updates the project board with their status:

- Dependabot-created PRs for JavaScript dependency updates

## Task

You need to discover and update security work items on the project board. Follow these steps:

### Step 1: Discover Dependabot PRs

Use the GitHub MCP server to search for pull requests in the `githubnext/gh-aw` repository with:
- Author: `app/dependabot`
- Labels: `dependencies`, `javascript`
- State: open

Example search query:
```
repo:githubnext/gh-aw is:pr author:app/dependabot label:dependencies label:javascript is:open
```

### Step 2: Check for Work

If *no* Dependabot PRs are found:
- Call the `noop` tool with message: "No security alerts found to process"
- Exit successfully

### Step 3: Update Project Board

For each discovered item (up to 100 total per run):
- Add or update the corresponding work item on the project board:
- Use the `update-project` safe output tool
- Always include the campaign project URL (this is what makes it a campaign):
  - `project`: "https://github.com/orgs/githubnext/projects/144"
- Always include the content identity:
  - `content_type`: `pull_request` (Dependabot PRs)
  - `content_number`: PR/issue number
- Set fields:
  - `campaign_id`: "security-alert-burndown"
  - `status`: "Todo" (for open items)
  - `target_repo`: "githubnext/gh-aw"
  - `worker_workflow`: who discovered it, using one of:
    - "dependabot"
  - `priority`: Estimate priority:
    - "High" for critical/severe alerts
    - "Medium" for moderate alerts
    - "Low" for low/none alerts
  - `size`: Estimate size:
    - "Small" for single dependency updates
    - "Medium" for multiple dependency updates
    - "Large" for complex updates with breaking changes
  - `start_date`: Item created date (YYYY-MM-DD format)
  - `end_date`: Item closed date (YYYY-MM-DD format) or today's date if still open

### Step 4: Assign work

After updating project items, **create agent sessions** to bundle and merge Dependabot PRs:

**Selection Criteria:**
1. Review all discovered PRs
2. Group by **runtime** (Node.js, Python, etc.) and **target dependency file**
3. Select up to **3 agent sessions** total following the bundling rules below

**Dependabot Bundling Rules:**

- Group work by **runtime** (Node.js, Python, etc.). Never mix runtimes.
- Group changes by **target dependency file**. Each PR must modify **one manifest (and its lockfile) only**.
- Bundle updates **only within a single target file**.
- Patch and minor updates **may be bundled**; major updates **should be isolated** unless dependencies are tightly coupled.
- Bundled releases **must include a research report** describing:
  - Packages updated and old → new versions
  - Breaking or behavioral changes
  - Migration steps or code impact
  - Risk level and test coverage impact
- Prioritize **security alerts and high-risk updates** first within each runtime.
- Enforce **one runtime + one target file per PR**.
- All PRs must pass **CI and relevant runtime tests** before merge.

**Creating Agent Sessions:**

For each selected group (up to 3 total), use the `create_agent_session` tool with a detailed task description:

```
create_agent_session(body="Bundle and merge Dependabot PRs for [runtime] [package.json/requirements.txt/go.mod]:

PRs to merge:
- #[pr_number]: [title] ([old_version] → [new_version])
- #[pr_number]: [title] ([old_version] → [new_version])

Task:
1. Research each package update for breaking changes
2. Create a research report documenting:
   - Packages updated and version changes
   - Breaking or behavioral changes found
   - Migration steps or code impact
   - Risk level and test coverage impact
3. Bundle the PRs into a single update
4. Test the bundled changes (run tests, verify CI passes)
5. Create a PR with the bundled update and research report

Constraints:
- All changes must target [manifest file] and its lockfile only
- Must pass all CI checks and relevant runtime tests
- Research report required before merging")
```

**Important:**
- Create agent sessions for highest priority updates first
- Limit to 3 agent sessions per run (max configured in safe-outputs)
- Each session should target a distinct runtime + file combination

### Step 5: Report

Summarize how many items were discovered and added/updated on the project board, broken down by category.

## Important

- Always use the `update-project` tool for project board updates
- If no work is found, call `noop` to indicate successful completion with no actions
- Focus only on open items:
  - PRs: open only
- Limit updates to 100 items per run to respect rate limits (prioritize highest severity/most recent first)
