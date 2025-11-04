---
on: 
  schedule:
    - cron: "0 0,6,12,18 * * *"  # Every 6 hours
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["smoke"]
permissions:
  contents: read
  issues: read
  pull-requests: read
  
name: Smoke Claude
engine: claude
tools:
  github:
safe-outputs:
    staged: true
    create-issue:
timeout_minutes: 10
strict: true
jobs:
  investigate-on-failure:
    if: failure()
    needs: agent
    uses: ./.github/workflows/smoke-detector.lock.yml
    permissions:
      contents: read
      issues: write
      pull-requests: read
      actions: read
    with:
      workflow_name: Smoke Claude
      run_id: ${{ github.run_id }}
      run_number: ${{ github.run_number }}
      conclusion: failure
      html_url: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}
      head_sha: ${{ github.sha }}
---

Review the last 5 merged pull requests in this repository and post summary in an issue.