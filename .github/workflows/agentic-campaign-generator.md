---
name: "Agentic Campaign Generator"
description: "Agentic Campaign generator that discovers workflows, generates a campaign spec and a project board, and assigns to Copilot agent for compilation"
on:
  issues:
    types: [labeled]
    names: ["create-agentic-campaign"]
    lock-for-agent: true
  workflow_dispatch:
  reaction: "eyes"
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: claude
tools:
  github:
    toolsets: [default]
safe-outputs:
  update-issue:
  assign-to-agent:
    github-token: "${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}"
  create-project:
    max: 1
    github-token: "${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}"
    target-owner: "${{ github.repository_owner }}"
    views:
      - name: "Progress Board"
        layout: "board"
        filter: "is:issue is:pr"
      - name: "Task Tracker"
        layout: "table"
        filter: "is:issue is:pr"
      - name: "Campaign Roadmap"
        layout: "roadmap"
        filter: "is:issue is:pr"
    field-definitions:
      - name: "Campaign Id"
        data-type: "TEXT"
      - name: "Worker Workflow"
        data-type: "TEXT"
      - name: "Priority"
        data-type: "SINGLE_SELECT"
        options:
          - "High"
          - "Medium"
          - "Low"
      - name: "Size"
        data-type: "SINGLE_SELECT"
        options:
          - "Small"
          - "Medium"
          - "Large"
      - name: "Start Date"
        data-type: "DATE"
      - name: "End Date"
        data-type: "DATE"
  update-project:
    max: 10
    github-token: "${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}"
  messages:
    run-started: "### :rocket: Campaign setup started

Creating a tracking Project and generating campaign files + orchestrator workflow.

No action needed — the [{workflow_name}]({run_url}) will open a pull request and post the link + checklist back on this issue when ready.

> To stop this run: remove the label that started it.

> **Docs**: https://githubnext.github.io/gh-aw/guides/campaigns/getting-started/"
    run-success: "### :white_check_mark: Campaign setup complete

Tracking Project created and pull request with generated campaign files is ready.

**Next steps**: Review + merge the PR, then run the campaign from the Actions tab.

> **Docs**: https://githubnext.github.io/gh-aw/guides/campaigns/getting-started/"
    run-failure: "### :x: Campaign setup {status}

**Common causes**:

- `GH_AW_PROJECT_GITHUB_TOKEN` is missing or invalid

- Token lacks access to GitHub Projects

**Action required**:

- Fix the first error in the logs

- Re-apply the label to re-run

> **Troubleshooting**: https://githubnext.github.io/gh-aw/guides/campaigns/flow/#when-something-goes-wrong

> **Docs**: https://githubnext.github.io/gh-aw/guides/campaigns/getting-started/"
timeout-minutes: 10
---

{{#runtime-import? .github/shared-instructions.md}}
{{#runtime-import? .github/aw/generate-agentic-campaign.md}}
