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
{{#if ${{ github.event.issue.number || (fromJSON(github.event.inputs.aw_context || '{}').item_type == 'issue' && fromJSON(github.event.inputs.aw_context || '{}').item_number) }} }}
- **issue-number**: #${{ github.event.issue.number || (fromJSON(github.event.inputs.aw_context || '{}').item_type == 'issue' && fromJSON(github.event.inputs.aw_context || '{}').item_number) }}
{{/if}}
{{#if ${{ github.event.discussion.number || (fromJSON(github.event.inputs.aw_context || '{}').item_type == 'discussion' && fromJSON(github.event.inputs.aw_context || '{}').item_number) }} }}
- **discussion-number**: #${{ github.event.discussion.number || (fromJSON(github.event.inputs.aw_context || '{}').item_type == 'discussion' && fromJSON(github.event.inputs.aw_context || '{}').item_number) }}
{{/if}}
{{#if ${{ github.event.pull_request.number || (fromJSON(github.event.inputs.aw_context || '{}').item_type == 'pull_request' && fromJSON(github.event.inputs.aw_context || '{}').item_number) }} }}
- **pull-request-number**: #${{ github.event.pull_request.number || (fromJSON(github.event.inputs.aw_context || '{}').item_type == 'pull_request' && fromJSON(github.event.inputs.aw_context || '{}').item_number) }}
{{/if}}
{{#if ${{ github.event.comment.id || fromJSON(github.event.inputs.aw_context || '{}').comment_id }} }}
- **comment-id**: ${{ github.event.comment.id || fromJSON(github.event.inputs.aw_context || '{}').comment_id }}
{{/if}}
{{#if ${{ github.run_id }} }}
- **workflow-run-id**: ${{ github.run_id }}
{{/if}}
</github-context>
