---
safe-outputs:
  steer: true
  create-pull-request:
    expires: 2d
    title-prefix: "[q] "
    labels: [automation, workflow-optimization]
    reviewers: copilot
    draft: false
    if-no-changes: "ignore"
    protected-files: fallback-to-issue
    max-patch-files: 500
---
