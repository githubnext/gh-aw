---
on:
  command:
    name: test-claude-create-pull-request-review-comment
  reaction: eyes

engine: 
  id: claude

safe-outputs:
  create-pull-request-review-comment:
    max: 3
---

Analyze the pull request #${github.event.pull_request.number} and create a few targeted review comments on the code changes. 

You MUST create 2 review comments focusing on:
1. Code quality and best practices
2. Potential security issues or improvements  
3. Performance optimizations or concerns

For each review comment, specify:
- The exact file path where the comment should be placed
- The specific line number in the diff
- A helpful comment body with actionable feedback

If you find multi-line issues, use start_line to comment on ranges of lines.
