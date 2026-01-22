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
  update-project:
    max: 10
    github-token: "${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}"
    field-definitions:
      - name: "status"
        data-type: "SINGLE_SELECT"
        options:
          - "Todo"
          - "In Progress"
          - "Review Required"
          - "Blocked"
          - "Done"
      - name: "campaign_id"
        data-type: "TEXT"
      - name: "worker_workflow"
        data-type: "TEXT"
      - name: "repository"
        data-type: "TEXT"
      - name: "priority"
        data-type: "SINGLE_SELECT"
        options:
          - "High"
          - "Medium"
          - "Low"
      - name: "size"
        data-type: "SINGLE_SELECT"
        options:
          - "Small"
          - "Medium"
          - "Large"
      - name: "start_date"
        data-type: "DATE"
      - name: "end_date"
        data-type: "DATE"
  messages:
    footer: "> *Managed by [{workflow_name}]({run_url})*
Docs: https://githubnext.github.io/gh-aw/guides/campaigns/getting-started/"
    run-started: "### :rocket: Campaign setup started

I’m creating a tracking Project and generating the campaign files + orchestrator workflow.

Next, I’ll open a pull request and post the link + checklist in this issue.

> To stop this run: remove the label that started it.

> To learn more about campaigns, visit: https://githubnext.github.io/gh-aw/guides/campaigns/getting-started/"
    run-success: "### :white_check_mark: Campaign setup complete

All set — the tracking Project is created and the pull request with the generated campaign files is ready.

Next: review + merge the PR, then run the orchestrator from the Actions tab.

> To learn more about campaigns, visit: https://githubnext.github.io/gh-aw/guides/campaigns/getting-started/"
    run-failure: "### :x: Campaign setup {status}

This is usually a quick fix. First check that `GH_AW_PROJECT_GITHUB_TOKEN` is set and has access to GitHub Projects.

Retry: fix the first error in the logs, then re-apply the label to re-run

> Troubleshooting: https://githubnext.github.io/gh-aw/guides/campaigns/flow/#when-something-goes-wrong

> To learn more about campaigns, visit: https://githubnext.github.io/gh-aw/guides/campaigns/getting-started/"
timeout-minutes: 10
---

{{#runtime-import? .github/shared-instructions.md}}
{{#runtime-import? .github/aw/generate-agentic-campaign.md}}
