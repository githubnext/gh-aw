---
on: workflow_dispatch
permissions:
  contents: read
engine: claude
safe-outputs:
  pull-request-ready-for-review:
    max: 1
timeout-minutes: 5
strict: false
---

# Test Pull Request Ready for Review (Claude)

Test marking a draft pull request as ready for review.

You are helping test a new safe output type called `pull_request_ready_for_review`.

This tool marks a draft pull request as ready for review by setting `draft: false` and posts a comment with the reason.

Please call the `pull_request_ready_for_review` tool with:
- `reason`: "All features implemented and tests passing. Ready for team review."
- `pull_request_number`: (if testing on a specific PR, otherwise it will use the triggering PR)

The tool will:
1. Check if the PR is currently a draft
2. If it is, set draft to false 
3. Post the reason as a comment on the PR

Output the JSONL format.
