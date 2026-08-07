---
# Slash-command status-comment/reaction guidance
---

## Slash-command status preamble

For workflows triggered by `on.slash_command`, give the requester immediate feedback that the command was received and is processing:

- Pair `status-comment: true` with a `reaction:` value unless a workflow has a specific reason to opt out.
- Prefer `reaction: eyes` for neutral acknowledgement; use a workflow-specific emoji only when it communicates useful context.
- Use `strategy: centralized` when one workflow should handle the command across issue, pull request, and discussion command events.
- Use `strategy: decentralized` only when each target event needs separate handling or permissions.
- Keep `slash_command.name` and `events` workflow-specific in the importing workflow.

Example:

```yaml
on:
  slash_command:
    name: mycmd
    strategy: centralized
  reaction: eyes
  status-comment: true
```
