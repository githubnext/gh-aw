<github-context>
The following GitHub context information is available for this workflow:
{{#if ${{ github.actor }} }}
- **actor**: ${{ github.actor }}
{{/if}}
{{#if ${{ github.repository }} }}
- **repository**: ${{ github.repository }}
{{/if}}
{{#if ${{ github.workspace }} }}
- **workspace**: ${{ github.workspace }}
{{/if}}
{{#if ${{ github.event.issue.number }} }}
- **issue-number**: #${{ github.event.issue.number }}
{{/if}}
{{#if ${{ github.event.discussion.number }} }}
- **discussion-number**: #${{ github.event.discussion.number }}
{{/if}}
{{#if ${{ github.event.pull_request.number }} }}
- **pull-request-number**: #${{ github.event.pull_request.number }}
{{/if}}
{{#if ${{ github.event.comment.id }} }}
- **comment-id**: ${{ github.event.comment.id }}
{{/if}}
{{#if ${{ github.run_id }} }}
- **workflow-run-id**: ${{ github.run_id }}
{{/if}}
</github-context>
