---
"gh-aw": minor
---

Add create-commit-status safe output type with pending/final status lifecycle

This PR implements a new safe output type `create-commit-status` for GitHub Agentic Workflows, allowing AI agents to create commit statuses on pull requests and commits with automatic pending/final status lifecycle management.

Key features:
- Automatic SHA detection for PR head, issues on PRs, push events, and workflow context
- Custom context string configuration (defaults to workflow name)
- Automatic pending status on workflow start in activation job
- Automatic final status on workflow completion in conclusion job (always runs)
- State mapping based on agent conclusion (success, failure, cancelled, unknown)
- Maximum of one status per workflow enforced
- Full staged mode support for testing
- Requires `statuses: write` permission

Configuration example:
```yaml
safe-outputs:
  create-commit-status:
    context: "ai-review/nitpick"
```

The lifecycle is fully automated - no manual status creation needed. The activation job creates a pending status, and the conclusion job updates it to the final status based on the agent's conclusion.
