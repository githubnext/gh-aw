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
  messages:
    footer: "> *Campaign coordination by [{workflow_name}]({run_url})*"
    run-started: "Campaign Generator started:
[{workflow_name}]({run_url}) is processing your campaign request for this {event_type}..."
    run-success: "Campaign setup complete:
This issue has been assigned to Copilot Coding Agent to compile the campaign and create a PR."
    run-failure: "Campaign setup interrupted!
[{workflow_name}]({run_url}) {status}. Please check the details and try again."
timeout-minutes: 10
---

{{#runtime-import? .github/shared-instructions.md}}
{{#runtime-import? .github/aw/generate-agentic-campaign.md}}
