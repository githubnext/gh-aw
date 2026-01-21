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
    footer: "> *Campaign coordination by [{workflow_name}]({run_url})*"
    run-started: "[{workflow_name}]({run_url}) is processing your campaign request for this {event_type}."
    run-success: "[{workflow_name}]({run_url}) has successfully set up your campaign. Copilot Coding Agent will now create a PR."
    run-failure: "[{workflow_name}]({run_url}) {status}. Please check the details and try again."
timeout-minutes: 10
---

{{#runtime-import? .github/shared-instructions.md}}
{{#runtime-import? .github/aw/generate-agentic-campaign.md}}
