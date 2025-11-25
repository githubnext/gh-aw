<github-context>
<description>The following GitHub context information is available for this workflow:</description>
{{#if ${{ github.repository }} }}
<context name="repository">${{ github.repository }}</context>
{{/if}}
{{#if ${{ github.workspace }} }}
<context name="workspace">${{ github.workspace }}</context>
{{/if}}
{{#if ${{ github.event.issue.number }} }}
<context name="issue-number">#${{ github.event.issue.number }}</context>
{{/if}}
{{#if ${{ github.event.discussion.number }} }}
<context name="discussion-number">#${{ github.event.discussion.number }}</context>
{{/if}}
{{#if ${{ github.event.pull_request.number }} }}
<context name="pull-request-number">#${{ github.event.pull_request.number }}</context>
{{/if}}
{{#if ${{ github.event.comment.id }} }}
<context name="comment-id">${{ github.event.comment.id }}</context>
{{/if}}
{{#if ${{ github.run_id }} }}
<context name="workflow-run-id">${{ github.run_id }}</context>
{{/if}}
<instruction>Use this context information to understand the scope of your work.</instruction>
</github-context>
