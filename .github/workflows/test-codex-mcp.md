---
on:
  issues:
    types: [opened]
  reaction: eyes

engine: 
  id: codex

safe-outputs:
  add-issue-comment:

tools:
  time:
    mcp:
      type: stdio
      container: "mcp/time"
      env:
        LOCAL_TIMEZONE: "${LOCAL_TIMEZONE}"
    allowed: ["get_current_time"]
---

**First, get the current time using the get_current_time tool to timestamp your analysis.**

If the title of the issue #${{ github.event.issue.number }} is "Hello from Codex" then add a comment on the issue "Reply from Codex" with the current time.

### AI Attribution

Include this footer in your PR comment:

```markdown
> AI-generated content by [${{ github.workflow }}](https://github.com/${{ github.repository }}/actions/runs/${{ github.run_id }}) may contain mistakes.
```