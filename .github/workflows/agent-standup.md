---
on:
  schedule:
    # Every day at 9am UTC
    - cron: "0 9 * * *"
  workflow_dispatch:

timeout_minutes: 15
permissions:
  contents: read
  models: read
  issues: write  # needed to write the output status report to an issue
  pull-requests: read
  discussions: read
  actions: read
  checks: read
  statuses: read

steps:
  - name: Checkout code
    uses: actions/checkout@v4
  
  - name: Set up Go
    uses: actions/setup-go@v5
    with:
      go-version-file: go.mod
      cache: true
  
  - name: Install dependencies
    run: make deps
  
  - name: Build gh-aw tool
    run: make build

tools:
  github:
    allowed: [create_issue, update_issue]
  claude:
    allowed:
      WebFetch:
      WebSearch:
---

# Agent Standup

1. Search for any previous "Agent Standup" open issues in the repository. Close them.

2. Collect agentic workflow activity from the last day:
   
   - Run `./gh-aw logs --start-date $(date -d '1 day ago' '+%Y-%m-%d')` to collect agentic workflow logs from the last day
   
3. Generate a report on **Agentic workflow activity from the last day** including:

      - Overview table (markdown format) with number of runs, cost for each agents    
      - Any errors or patterns in the logs
      - Anything interresting your discover

   - If little has happened, don't write too much.

   - Be helpful, thoughtful, respectful, positive, kind, and encouraging.

   - Use emojis to make the report more engaging and fun, but don't overdo it.

 
4. Create a new GitHub issue with title starting with "Agent Standup" containing a markdown report with your findings. Use links where appropriate.

   Only a new issue should be created, no existing issues should be adjusted.

@include agentics/shared/include-link.md

@include agentics/shared/job-summary.md

@include agentics/shared/xpia.md

@include agentics/shared/gh-extra-tools.md

