---
tools:
  github:
    allowed: [add_issue_comment]
---

## Report Results in Issue Comment

After completing your workflow tasks, add a concise comment to issue #${{ github.event.issue.number }} summarizing your results. Keep the comment brief and focused.

### Comment Format
- Start with a simple status indicator (✅ Complete, ⚠️ Partial, ❌ Failed)
- Provide a 1-2 sentence summary of what was accomplished
- Include a link to the full action run for detailed information
- Avoid lengthy explanations or excessive emojis

### Example Structure
```markdown
✅ [Brief status summary in 1-2 sentences]

[📋 View detailed logs and full analysis](${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }})
```

### Guidelines
- **Be concise**: The comment should be scannable in under 10 seconds
- **Reference the action run**: Direct users to the GitHub Actions run for complete details
- **Focus on outcomes**: What was done, not how it was done
- **Use minimal formatting**: Simple bullet points or short paragraphs only
- **Link to job summary**: The GitHub Actions job summary should contain the full analysis and details

### When to Comment
- Always comment when the workflow completes successfully
- Always comment when the workflow encounters errors or limitations  
- Comment on issue #${{ github.event.issue.number }} (the issue that triggered this workflow)

> **Note**: Use the `add_issue_comment` GitHub tool to post the comment on issue #${{ github.event.issue.number }}.