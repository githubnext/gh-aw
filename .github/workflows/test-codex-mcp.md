---
on:
  workflow_dispatch:
  reaction: eyes

engine: 
  id: codex

safe-outputs:
  create-issue:

network: {}

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

Create an issue with title "Hello from Codex" and a comment in the body saying what the current time is and if you were successful in using the MCP tool

### AI Attribution

Include this footer in your issue description:

```markdown
> AI-generated content by [${{ github.workflow }}](https://github.com/${{ github.repository }}/actions/runs/${{ github.run_id }}) may contain mistakes.
```