---
on: workflow_dispatch
permissions:
  contents: read
  actions: read
engine: claude
safe-outputs:
  close-pull-request:
    max: 3
    required-labels: ["test", "automated"]
    required-title-prefix: "[bot]"
    target: "*"
timeout-minutes: 5
---

# Test Close Pull Request

Test the close-pull-request safe output functionality.

Close pull requests that match the following criteria:
- Have labels: test, automated
- Have title prefix: [bot]

Create close-pull-request entries with:
- body: "This pull request is being closed automatically as part of testing. The PR met the required criteria for automated closure."
- pull_request_number: 123 (example)

Output as JSONL format.
