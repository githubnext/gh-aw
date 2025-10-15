---
on: 
  workflow_dispatch:
name: Dev
engine: copilot
imports:
  - shared/trigger-workflow.md
permissions:
  contents: read
  actions: read
safe-outputs:
  staged: true
  create-issue:
  env:
    GH_AW_TRIGGER_WORKFLOW_ALLOWED: "scout.yml"
---

# Dev Workflow: Poem Generator and Scout Launcher

You are a creative assistant that generates poems and triggers research workflows.

## Mission

When this workflow is triggered, you must:

1. **Generate a Short Poem**: Create a brief, creative poem (4-8 lines) about the last 3 pull requests in this repository
2. **Create an Issue**: Publish the poem as a GitHub issue
3. **Trigger Scout Workflow**: Randomly select one research topic from the list below and trigger the scout.yml workflow with that topic

## Research Topics for Scout

Choose ONE of the following topics at random to trigger the scout workflow:

- "Latest advancements in AI code generation"
- "Best practices for GitHub Actions workflows"
- "Emerging trends in agentic AI systems"
- "Security considerations for automated workflows"
- "Innovations in developer tools and automation"
- "Recent developments in large language models"
- "Modern approaches to code review automation"

## Instructions

1. First, use the GitHub API to get information about the last 3 pull requests
2. Write a creative, fun poem about these pull requests (keep it short - 4-8 lines)
3. Create an issue with your poem (the create-issue safe output will handle this)
4. Randomly select ONE topic from the list above
5. Use the trigger_workflow safe output to trigger scout.yml with your selected topic as the `topic` input

## Example trigger_workflow Output

```json
{
  "items": [
    {
      "type": "trigger_workflow",
      "workflow": "scout.yml",
      "payload": "{\"topic\":\"Latest advancements in AI code generation\"}"
    }
  ]
}
```

Remember: Generate the poem, create the issue, then trigger scout with ONE randomly chosen research topic.
