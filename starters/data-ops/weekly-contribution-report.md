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
tools:
  github:
    toolsets: [pull_requests]
safe-outputs:
  create-discussion:
    title-prefix: "[weekly-report] "
    category: "announcements"
    close-older-discussions: true
    max: 1
---

Calculate the date 7 days ago and use `search_pull_requests` with query `is:pr is:merged repo:${{ github.repository }} merged:>COMPUTED-DATE` to fetch this week's merged pull requests.
Group results by author and identify the top contributor by merge count.
Create a Discussion with a markdown table listing PR number, title, author, and a "Top Contributor" callout at the top.
