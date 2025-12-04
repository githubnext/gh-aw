<github-context>
The following GitHub context information is available for this workflow:
{{#if ${GH_AW_GITHUB_ACTOR} }}
- **actor**: ${{ github.actor }}
{{/if}}
{{#if ${GH_AW_GITHUB_REPOSITORY} }}
- **repository**: ${{ github.repository }}
{{/if}}
{{#if ${GH_AW_GITHUB_WORKSPACE} }}
- **workspace**: ${{ github.workspace }}
{{/if}}
{{#if ${GH_AW_GITHUB_EVENT_ISSUE_NUMBER} }}
- **issue-number**: #${{ github.event.issue.number }}
{{/if}}
{{#if ${GH_AW_GITHUB_EVENT_DISCUSSION_NUMBER} }}
- **discussion-number**: #${{ github.event.discussion.number }}
{{/if}}
{{#if ${GH_AW_GITHUB_EVENT_PULL_REQUEST_NUMBER} }}
- **pull-request-number**: #${{ github.event.pull_request.number }}
{{/if}}
{{#if ${GH_AW_GITHUB_EVENT_COMMENT_ID} }}
- **comment-id**: ${{ github.event.comment.id }}
{{/if}}
{{#if ${GH_AW_GITHUB_RUN_ID} }}
- **workflow-run-id**: ${{ github.run_id }}
{{/if}}
</github-context>
