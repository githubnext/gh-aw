---
emoji: "💻"
on:
  workflow_dispatch:
  label_command:
    name: dev
    strategy: decentralized
  schedule:
    - cron: 'daily around 9:00'  # ~9 AM UTC
name: Dev
description: Daily status report for gh-aw project
timeout-minutes: 30
strict: true
engine:
  id: codex
  model: gpt-5.4

permissions:
  contents: read
  issues: read
  pull-requests: read

safe-outputs:
  create-issue:
    expires: 7d
    title-prefix: "[Daily Report] "

imports:
  - shared/otlp.md
  - shared/reporting.md
tools:
  github:
    mode: gh-proxy
    toolsets:
      - default
  cli-proxy: true

---

<!--
# GitHub Agentic Workflows — README Summary

Write agentic workflows in natural language markdown and run them in GitHub Actions.

## Key Concepts
- **Quick Start**: Step-by-step guide at https://github.github.com/gh-aw/setup/quick-start/
- **Overview**: Agentic workflows let AI automate repository tasks using natural language prompts.
  See https://github.github.com/gh-aw/introduction/how-they-work/
- **Guardrails**: Workflows run with read-only permissions by default. Write operations require
  sanitized `safe-outputs`. Security layers include sandboxed execution, input sanitization,
  network isolation, SHA-pinned supply chain, tool allow-listing, and compile-time validation.
  Human approval gates are available for critical operations.
  See https://github.github.com/gh-aw/introduction/architecture/
- **Documentation**: https://github.github.com/gh-aw/ — machine-readable llms.txt also available.
- **Contributing**: See CONTRIBUTING.md for development setup and contribution guidelines.

## Related Projects
- **AWF** (Agent Workflow Firewall): network egress control — https://github.com/github/gh-aw-firewall
- **MCP Gateway**: unified HTTP gateway for MCP server calls — https://github.com/github/gh-aw-mcpg
- **gh-aw-actions**: shared GitHub Actions library — https://github.com/github/gh-aw-actions
-->

# Daily Documentation Quality Report

Generate a daily documentation quality report for the gh-aw project.

## Step 1: Find Documentation Problems in Issues

Search GitHub issues for documentation problems using these queries:
- `repo:${{ github.repository }} is:issue is:open label:documentation`
- `repo:${{ github.repository }} is:issue is:open docs OR documentation OR "missing docs" OR "unclear" OR "broken link"`

For each issue found, record: issue number, title, and a brief description of the documentation problem reported.

## Step 2: Cross-Reference with Current Documentation

For the top documentation issues found (up to 10), check whether the relevant documentation section in `docs/` already addresses the issue:
- If yes, note that the issue could be closed with a pointer to the docs.
- If no, note the gap.

## Step 3: Create the Report Issue

**MANDATORY**: After completing Steps 1 and 2, you MUST call one of these safe-output tools:

### If documentation problems were found:

Call `create_issue` with:
- **Title**: `Documentation Quality Report — YYYY-MM-DD` (use today's date)
- **Body**: Use this exact template (all sections required — do NOT leave the body empty):

```markdown
### Documentation Quality Report — YYYY-MM-DD

### Summary

Brief 2-3 sentence overview of what was found.

- **Issues analyzed**: [N]
- **Issues with documentation gaps**: [N]
- **Issues already answered in docs**: [N]

### Documentation Issues Found

| Issue | Title | Status |
|-------|-------|--------|
| #NNN | [title] | ⚠️ Gap / ✅ Already documented |

### Documentation Gaps

For each issue where documentation is missing or incorrect:

#### Gap 1: [Short description]
- **Issue**: #NNN — [title]
- **Missing coverage**: [What is not documented]
- **Suggested doc location**: `docs/[path]`

### Issues That Could Be Closed

Issues where the documentation already contains the answer:

- #NNN — [title]: See `[doc path]` section "[section heading]"

### Recommendations

1. [Specific actionable recommendation]
2. [Another recommendation]
```

### If NO documentation problems were found:

Call `noop` with message: "No documentation problems found in open issues — no action needed."

{{#runtime-import shared/noop-reminder.md}}
