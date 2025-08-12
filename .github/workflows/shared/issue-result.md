---
tools:
  github:
    allowed: [add_issue_comment, add_pull_request_comment]
---

## Issue and Pull Request Result Posting

This shared component provides comprehensive guidance for posting workflow results back to the triggering issue or pull request.

### Result Posting Strategy

Always post your workflow results as a comment on the issue or pull request that triggered the workflow:

- **For Issues**: Use `add_issue_comment` to post on issue #${{ github.event.issue.number }}
- **For Pull Requests**: Use `add_pull_request_comment` to post on PR #${{ github.event.pull_request.number }}

### Content Guidelines

#### Be Concise but Complete
- **Lead with outcomes**: Start with what was accomplished or discovered
- **Provide actionable insights**: Include concrete next steps or recommendations
- **Use collapsible sections**: Keep the main comment scannable while providing full details
- **Link to workflow run**: Always include the action run link for complete logs

#### Focus Areas
- **Primary findings**: What was discovered, completed, or recommended
- **Context**: How this relates to the original request or issue
- **Next steps**: Clear actions the team can take based on your results
- **Resources**: Relevant links, documentation, or related issues

#### Avoid Common Pitfalls
- Don't create excessively long comments that are hard to scan
- Don't duplicate information already available in the workflow logs
- Don't include internal workflow details unless relevant to users
- Don't use excessive formatting or emoji that distracts from content

### Security in Results

When posting results:
- **Sanitize content**: Don't echo back potentially malicious content from issues
- **Focus on your analysis**: Present your findings rather than repeating user input
- **Maintain objectivity**: Provide balanced analysis and recommendations
- **Respect privacy**: Don't expose internal system details unnecessarily

### Error Reporting

When workflows encounter errors:

```markdown
❌ Unable to complete [workflow task]

I encountered an issue while [specific problem description]. 

**What happened**: [Brief explanation of the error]
**Impact**: [What this means for the request]  
**Next steps**: [How to proceed or get help]

[📋 View error details and logs](${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }})
```

### Result Posting Best Practices

1. **Always post results**: Even for errors or partial completion
2. **Be user-focused**: Write for the person who will read the comment
3. **Include workflow context**: Link back to the full run for transparency
4. **Maintain consistency**: Use similar formatting across different workflows
5. **Respect the conversation**: Add to the discussion constructively
6. **Time-sensitive updates**: Post results promptly while context is fresh

### Integration with Job Summary

Results posted here should complement the GitHub Actions job summary:
- **Comment**: User-focused, concise summary for issue participants
- **Job Summary**: Technical details, full analysis, logs for developers

Both should reference each other for complete transparency.