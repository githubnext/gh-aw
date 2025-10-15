
---

## GitHub Context

The following GitHub context information is available for this workflow:

{{#if ${{ github.repository }} }}
- **Repository**: `${{ github.repository }}`
{{/if}}
{{#if ${{ github.event.issue.number }} }}
- **Issue Number**: `#${{ github.event.issue.number }}`
{{/if}}
{{#if ${{ github.event.discussion.number }} }}
- **Discussion Number**: `#${{ github.event.discussion.number }}`
{{/if}}
{{#if ${{ github.event.pull_request.number }} }}
- **Pull Request Number**: `#${{ github.event.pull_request.number }}`
{{/if}}
{{#if ${{ github.event.comment.id }} }}
- **Comment ID**: `${{ github.event.comment.id }}`
{{/if}}
{{#if ${{ github.run_id }} }}
- **Workflow Run ID**: `${{ github.run_id }}`
{{/if}}

Use this context information to understand the scope of your work.
