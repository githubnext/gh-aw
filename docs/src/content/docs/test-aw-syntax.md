---
title: Test AW Syntax Highlighting
description: Test the new "aw" code region type for agentic workflow files
---

This page tests the new "aw" syntax highlighting for agentic workflow files that contain both YAML frontmatter and markdown content.

## Example Agentic Workflow

```aw
---
on:
  issues:
    types: [opened]

permissions:
  issues: write

tools:
  github:
    allowed: [add_issue_comment]

engine: claude
timeout_minutes: 10
---

# Issue Responder Workflow

This workflow automatically responds to new issues with helpful information.

When a new issue is opened in repository ${{ github.repository }}, analyze the issue content and provide:

1. **Initial assessment** of the issue type
2. **Relevant resources** and documentation links  
3. **Next steps** for resolution

The response should be helpful and welcoming to new contributors.
```

## Regular Markdown for Comparison

```markdown
# Regular Markdown

This is regular markdown content without frontmatter.

- Item 1
- Item 2
- Item 3
```

## Regular YAML for Comparison

```yaml
on:
  issues:
    types: [opened]
permissions:
  issues: write
tools:
  github:
    allowed: [add_issue_comment]
```