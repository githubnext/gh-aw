---
on:
  pull_request:
    types: [opened, synchronize, reopened]
  reaction: eyes

engine: 
  id: codex

if: contains(github.event.pull_request.title, 'prr')

safe-outputs:
  create-pull-request-review-comment:
    max: 3
---

Analyze the pull request and create a few targeted review comments on the code changes.

Create 2-3 review comments focusing on:
1. Code quality and best practices
2. Potential security issues or improvements
3. Performance optimizations or concerns

For each review comment, specify:
- The exact file path where the comment should be placed
- The specific line number in the diff
- A helpful comment body with actionable feedback

If you find multi-line issues, use start_line to comment on ranges of lines.
