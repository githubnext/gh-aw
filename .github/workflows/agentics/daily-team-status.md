---
on:
  schedule:
    # Every day at 9am UTC, all days except Saturday and Sunday
    - cron: "0 9 * * 1-5"
  workflow_dispatch:

  stop-after: +30d # workflow will no longer trigger after 30 days. Remove this and recompile to run indefinitely

permissions: read-all

network: defaults

safe-outputs:
  create-discussion:
    title-prefix: "[team-status] "
    category: "announcements"

timeout-minutes: 15

tools:
  github:
source: githubnext/agentics/workflows/daily-team-status.md@a9694364f9aed4a0b67a0617d354b109542c1b80
---
# Daily Team Status

1. Search for recent open discussions with title "${{ github.workflow }}" in the repository. Read them to understand the context of the team and recent activity, and to avoid duplication.

2. Write an upbeat, friendly, motiviating summary of recent activity in the repo.

   - Include some or all of the following:
     - Recent issues activity
     - Recent pull requests
     - Recent discussions
     - Recent releases
     - Recent comments
     - Recent code reviews
     - Recent code changes
     - Recent failed CI runs

   - If little has happened, don't write too much.

   - Give some depth thought into ways the team can improve their productivity, and suggest some ways to do that.

   - Include a description of open source community engagement, if any.

   - Highlight suggestions for possible investment, ideas for features and project plan, ways to improve community engagement, and so on.

   - Be helpful, thoughtful, respectful, positive, kind, and encouraging.

   - Use emojis to make the report more engaging and fun, but don't overdo it.

   - Include a short haiku at the end of the report to help orient the team to the season of their work.

   - In a note at the end of the report, include a log of
     - all search queries (web, issues, pulls, content) you used to generate the data for the report
     - all commands you used to generate the data for the report
     - all files you read to generate the data for the report
     - places you didn't have time to read or search, but would have liked to

   Create a new GitHub discussion containing a markdown report with your findings. Use links where appropriate.

   Only a new discussion should be created, no existing discussions should be adjusted.
