---
title: Automated CI failure investigation
description: Use CI Doctor as a gh-aw example that diagnoses failed GitHub Actions runs and opens actionable reports automatically.
---

CI failure investigation with gh-aw means starting an agent after a failed GitHub Actions workflow so it can inspect failed jobs and logs, correlate the failure with repository changes, and produce an actionable diagnosis. This example is a portable adaptation of [CI Doctor](https://github.com/githubnext/agentics/blob/main/workflows/ci-doctor.md).

```aw wrap title=".github/workflows/ci-doctor.md"
---
on:
  workflow_run:
    workflows: ["CI"]
    types: [completed]
    branches: [main]

permissions:
  actions: read
  contents: read
  issues: read
  pull-requests: read

safe-outputs:
  create-issue:
    title-prefix: "[CI failure] "
    labels: [ci-failure]
    close-older-issues: true
---

# CI Failure Doctor

If the workflow run succeeded or was skipped, do nothing. Otherwise, inspect its jobs and logs, identify the first meaningful error, and correlate it with recent code or configuration changes.

Create one issue that includes the failing workflow and job, relevant log evidence, the likely root cause, confidence level, and concrete next steps. Do not create an issue when the failure was cancelled intentionally or has already been reported.
```

The agent has read-only access to Actions and repository content. `create-issue` is a safe output, so gh-aw validates the report before creating it. Change `workflows: ["CI"]` to the workflow names used by the repository, and create the `ci-failure` label before enabling the example.

## Learn More

- [CI Doctor source workflow](https://github.com/githubnext/agentics/blob/main/workflows/ci-doctor.md)
- [Workflow run triggers](/gh-aw/reference/triggers/#workflow-run-triggers-workflow_run)
- [Safe outputs](/gh-aw/reference/safe-outputs/)
- [Debugging workflows](/gh-aw/troubleshooting/debugging/)