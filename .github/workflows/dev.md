---
on: 
  workflow_dispatch:
name: Dev
engine: copilot
permissions:
  contents: read
  actions: read
tools:
  edit:
  bash:
safe-outputs:
  staged: true
  create-pull-request:
    title-prefix: "[poetry] "
    labels: [poetry, automation]
---

# Create Two Poetry Pull Requests

Analyze the last 3 pull requests in this repository and create two separate pull requests:

## First Pull Request - Haiku Style
1. Create a new branch called "poetry-haiku"
2. Write a haiku (3 lines: 5-7-5 syllables) about the last 3 pull requests
3. Save it to a file called `poems/recent-prs-haiku.md`
4. Commit with message "Add haiku about recent PRs"
5. Create a pull request with title "Add Haiku About Recent PRs"

## Second Pull Request - Limerick Style
1. Create a new branch called "poetry-limerick"
2. Write a limerick (5 lines, AABBA rhyme scheme) about the last 3 pull requests
3. Save it to a file called `poems/recent-prs-limerick.md`
4. Commit with message "Add limerick about recent PRs"
5. Create a pull request with title "Add Limerick About Recent PRs"

Both pull requests should be created in this single workflow run.
