---
title: Keeping documentation up to date automatically
description: Use gh-aw to detect documentation drift and open reviewable pull requests that update docs automatically.
---

Documentation automation with GitHub Agentic Workflows means running an agent on a schedule or after code changes so it can detect drift between code and docs, prepare updates, and propose them as a pull request. gh-aw keeps the agent inside a controlled workflow and uses a safe output to turn proposed documentation changes into a reviewable PR.

Install the starter with `gh aw add-wizard githubnext/agentics/docs-updater`.

```aw wrap title=".github/workflows/docs-updater.md"
---
on:
  schedule: weekly

permissions:
  contents: read
  pull-requests: read

safe-outputs:
  create-pull-request:
    title-prefix: "[docs] "
    draft: true
---

# Documentation Updater

Review code and documentation changes from the last seven days.

Identify outdated setup steps, missing option descriptions, and examples that no longer match current behavior. Update the relevant documentation files and open a draft pull request describing the changes and any areas that still require human review.
```

`create-pull-request` matters for security because the agent does not push directly to the default branch. gh-aw validates the proposed changes and opens a pull request for human review, which keeps documentation updates reviewable before merge.

## Learn More

- [Run Claude Code in GitHub Actions with gh-aw](/gh-aw/engines/claude/)
- [Run GitHub Copilot agents in GitHub Actions with gh-aw](/gh-aw/engines/copilot/)
- [Safe outputs](/gh-aw/reference/safe-outputs/)
- [Quick start](/gh-aw/setup/quick-start/)
