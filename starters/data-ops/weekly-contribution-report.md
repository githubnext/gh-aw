---
name: Weekly Contribution Report
description: Post a weekly digest of merged PRs and top contributors as a GitHub Discussion.
on:
  schedule:
    - cron: "0 9 * * 1"
  workflow_dispatch:
permissions:
  contents: read
  pull-requests: read
steps:
  - name: Collect merged PRs
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      gh api "repos/${{ github.repository }}/pulls?state=closed&sort=updated&direction=desc&per_page=50" \
        --jq '[.[] | select(.merged_at != null and (.merged_at > (now - 604800 | todate))) | {number, title, user: .user.login, merged_at}]' \
        > /tmp/gh-aw/merged-prs.json
safe-outputs:
  create-discussion:
    title-prefix: "[weekly-report] "
    category: "announcements"
    close-older-discussions: true
    max: 1
---

Read `/tmp/gh-aw/merged-prs.json` and create a weekly contribution summary listing the merged pull requests and their authors.
Group contributions by author and call out the top contributor by number of merges.
Format the report as a markdown table with columns for PR number, title, and author.
