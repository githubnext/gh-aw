---
on:
  schedule:
    # Every week, 9AM UTC, Monday
    - cron: "0 9 * * 1"
  workflow_dispatch:

  stop-after: +30d # workflow will no longer trigger after 30 days. Remove this and recompile to run indefinitely

permissions: read-all

network: defaults

safe-outputs:
  create-discussion:
    title-prefix: "${{ github.workflow }}"
    category: "ideas"

tools:
  github:
    toolsets: [all]
  web-fetch:
  web-search:

timeout-minutes: 15

source: githubnext/agentics/workflows/weekly-research.md@9586b5fc47d008cd1cf42f6c298a46abfd774fb5
---
# Weekly Research

## Job Description

Do a deep research investigation in ${{ github.repository }} repository, and the related industry in general.

- Read selections of the latest code, issues and PRs for this repo.
- Read latest trends and news from the software industry news source on the Web.

Create a new GitHub discussion with title starting with "${{ github.workflow }}" containing a markdown report with

- Interesting news about the area related to this software project.
- Related products and competitive analysis
- Related research papers
- New ideas
- Market opportunities
- Business analysis
- Enjoyable anecdotes

Only a new discussion should be created, no existing discussions should be adjusted.

At the end of the report list write a collapsed section with the following:
- All search queries (web, issues, pulls, content) you used
- All bash commands you executed
- All MCP tools you used
