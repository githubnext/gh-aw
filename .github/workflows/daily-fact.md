---
description: Posts a daily random fact about the gh-aw project to a discussion thread
on:
  schedule:
    - cron: "0 11 * * 1-5"  # 11 AM UTC, weekdays only
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
  actions: read
  discussions: read
tracker-id: daily-fact-thread
engine:
  id: codex
  model: gpt-5-mini
strict: false  # Required: codex engine doesn't support network firewall
timeout-minutes: 15

network:
  allowed:
    - defaults

tools:
  github:
    toolsets:
      - default
      - discussions
safe-outputs:
  add-comment:
    target: "4750"
    discussion: true
---

# Daily Fact About gh-aw

Your task is to post a fun, interesting fact about the ${{ github.repository }} project to discussion #4750.

## Data Sources

Mine recent activity from the repository to find interesting facts. Focus on:

1. **Recent PRs** (merged in the last 1-2 weeks)
   - New features added
   - Bug fixes
   - Refactoring efforts
   - Performance improvements

2. **Recent Releases** (if any)
   - New version highlights
   - Breaking changes
   - Notable improvements

3. **Recent Closed Issues** (resolved in the last 1-2 weeks)
   - Bugs that were fixed
   - Feature requests implemented
   - Community contributions

## Guidelines

- **Favor recent updates** but include variety - pick something interesting, not just the most recent
- **Be specific**: Include PR numbers, issue references, or release tags when relevant
- **Keep it short**: One or two sentences for the main fact, optionally with a brief context
- **Be engaging**: Use a casual, friendly tone that makes the fact memorable
- **Add variety**: Don't repeat the same type of fact every day (e.g., alternate between PRs, issues, releases, contributors, code patterns)

## Output Format

Create a single comment with this structure:

```
📊 **Daily gh-aw Fact**

[Your interesting fact here, referencing specific PRs, issues, or releases with links]

---
*Today's fact brought to you by the Daily Fact Bot 🤖*
```

## Examples

Good facts:
- "Did you know? PR #1234 added support for the `playwright` tool, enabling browser automation in agentic workflows! 🎭"
- "This week, 5 issues related to MCP server configuration were resolved, making it easier than ever to set up custom tools."
- "The latest release v0.45.0 introduced the `cache-memory` feature, allowing agents to persist data across workflow runs! 💾"
- "Fun fact: The most active contributor this week fixed 3 bugs related to YAML parsing. Thanks @contributor! 🙌"

Bad facts:
- "The repository was updated today." (too vague)
- "There were some changes." (not specific)
- Long paragraphs (keep it brief)

Now, analyze the recent activity and post one interesting fact to discussion #4750.
