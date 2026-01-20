---
name: "Agentic Campaign Generator"
description: "Agentic Campaign generator that creates project board, discovers workflows, generates campaign spec, and assigns to Copilot agent for compilation"
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
  update-project:
    max: 10
    github-token: "${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}"
  messages:
    footer: "> *Campaign coordination by [{workflow_name}]({run_url})*"
    run-started: "Campaign Generator starting! [{workflow_name}]({run_url}) is processing your campaign request for this {event_type}..."
    run-success: "Campaign setup complete! [{workflow_name}]({run_url}) has successfully coordinated your campaign creation. Your project is ready!"
    run-failure: "Campaign setup interrupted! [{workflow_name}]({run_url}) {status}. Please check the details and try again..."
timeout-minutes: 10
---

{{#runtime-import? .github/shared-instructions.md}}
{{#runtime-import? .github/aw/generate-agentic-campaign.md}}
