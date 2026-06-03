---
name: workflow-step-summaries
description: Write clear GitHub Actions step summaries with progressive disclosure.
---

# GitHub Actions Step Summary Guidance

Use this skill when generating content for `$GITHUB_STEP_SUMMARY`.

### Structure summaries for quick scanning

- Start sections at `###` (h3) for readable hierarchy in workflow run pages.
- Keep titles plain text with no emojis.
- Put the most important status and outcomes first.

### Use progressive disclosure

- Wrap detailed diagnostics, logs, and secondary data in HTML `<details>` blocks.
- Use a concise `<summary>` line that states what the collapsed section contains.
- Keep default-expanded content short; move verbose output into collapsible blocks.

### Use Markdown for code and review output

- In `actions/github-script`, prefer `core.summary.*` helpers to build summary content.
- Use inline code with backticks for commands, paths, IDs, and config keys.
- Use fenced code blocks with a language tag for logs, diffs, snippets, or commands.
- Present review findings as markdown sections with clear severity and action items.

### Suggested checklist before writing

- Confirm section headings start at h3.
- Confirm no title includes emoji.
- Confirm verbose content is inside `<details>` blocks.
- Confirm code and review content uses proper markdown code formatting.
