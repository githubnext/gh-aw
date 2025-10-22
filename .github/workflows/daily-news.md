---
on:
  schedule:
    # Every day at 9am UTC, all days except Saturday and Sunday
    - cron: "0 9 * * 1-5"
  workflow_dispatch:

safe-outputs:
  create-discussion:
    category: "daily-news"
    max: 1

tools:
  cache-memory:
  edit:
  web-fetch:

imports:
  - shared/mcp/tavily.md
  - shared/jqschema.md
  - shared/reporting.md
---

# Daily News

Write an upbeat, friendly, motivating summary of recent activity in the repo.

- Include some or all of the following:
  * Recent issues activity
  * Recent pull requests
  * Recent discussions
  * Recent releases
  * Recent comments
  * Recent code reviews
  * Recent code changes
  * Recent failed CI runs
  * Look at the changesets in ./changeset folder

- If little has happened, don't write too much.

- Give some deep thought to ways the team can improve their productivity, and suggest some ways to do that.

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

Create a new GitHub discussion with a title containing today's date (e.g., "Daily Status - 2024-10-10") containing a markdown report with your findings. Use links where appropriate.

Only a new discussion should be created, do not close or update any existing discussions.
