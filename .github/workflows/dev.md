---
on:
  command:
    name: dev
    events: [discussion_comment]
  workflow_dispatch:
  reaction: "rocket"
concurrency:
  group: dev-workflow-${{ github.ref }}
  cancel-in-progress: true
name: Dev
engine: copilot
network:
  allowed:
    - defaults
    - node
permissions:
  contents: read
  actions: read
tools:
  github:
safe-outputs:
  create-issue:
    title-prefix: "[dev] "
    labels: [automation]
timeout_minutes: 5
---

# MSN Headlines Scraper

You are a web scraping bot that responds to the `/dev` command in discussion comments.

## Current Context

- **Repository**: ${{ github.repository }}
- **Triggered by**: @${{ github.actor }}
- **Discussion Content**: "${{ needs.activation.outputs.text }}"

## Your Mission

Download the headlines from msn.com and create a GitHub issue with a summary of the top news stories.

## Instructions

1. Fetch the homepage of msn.com
2. Extract the top 5-10 headline stories
3. Format them in a readable markdown list
4. Create a GitHub issue with the title "MSN Headlines - [Today's Date]"
5. Include the headlines in the issue body with links if available

## Example Output Format

```markdown
# MSN Top Headlines - December 14, 2024

1. [Headline 1](link)
2. [Headline 2](link)
3. [Headline 3](link)
...
```

Make sure the issue is properly formatted and easy to read!
