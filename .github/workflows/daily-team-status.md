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
  issues: write
  pull-requests: read
  discussions: read
  actions: read
  checks: read
  statuses: read

tools:
  github:
    allowed: [create_issue, update_issue]
  claude:
    allowed:
      WebFetch:
      WebSearch:
---

# Daily Team Status

1. Search for any previous "Daily Team Status" open issues in the repository. Close them.

2. Write an upbeat, friendly, motiviating summary of recent activity in the repo.

   - Include some or all of the following:
     * Recent issues activity
     * Recent pull requests
     * Recent discussions
     * Recent releases
     * Recent comments
     * Recent code reviews
     * Recent code changes
     * Recent failed CI runs

   - If little has happened, don't write too much.

   - Give some depth thought into ways the team can improve their productivity, and suggest some ways to do that.

   - Include a description of open source community engagement, if any.

   - Highlight suggestions for possible investment, ideas for features and project plan, ways to improve community engagement, and so on.

   - Be helpful, thoughtful, respectful, positive, kind, and encouraging.

   - Use emojis to make the report more engaging and fun, but don't overdo it.

   - Include a short haiku at the end of the report to help orient the team to the season of their work.

   - In a note at the end of the report, include a log of
     * all search queries (web, issues, pulls, content) you used to generate the data for the report
     * all commands you used to generate the data for the report
     * all files you read to generate the data for the report
     * places you didn't have time to read or search, but would have liked to

   Create a new GitHub issue with title starting with "Daily Team Status" containing a markdown report with your findings. Use links where appropriate.

   Only a new issue should be created, no existing issues should be adjusted.

@include shared/include-link.md

@include shared/job-summary.md

@include shared/xpia.md

@include shared/gh-extra-tools.md

