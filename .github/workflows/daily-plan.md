---
# Run once a day at midnight UTC
on:
  schedule:
    - cron: "0 0 * * *"
  workflow_dispatch:

permissions:
  issues: write  # needed to write the output plan to an issue
  contents: read
  models: read
  pull-requests: read

timeout_minutes: 15

tools:
  github:
    allowed:
      [
        create_issue,
        update_issue,
      ]
  claude:
    allowed:
      WebFetch:
      WebSearch:
---

# Agentic Planner

## Job Description

Your job is to act as a planner for the GitHub repository ${{ env.GITHUB_REPOSITORY }}.

1. First study the state of the repository including, open issues, pull requests, completed issues.

   - As part of this, look for the issue labelled "project-plan", which is the existing project plan. Read the plan, and any comments on the plan. If no issue is labelled "project-plan" ignore this step.

   - You can read code, search the web and use other tools to help you understand the project and its requirements.

2. Formulate a plan for the remaining work to achieve the objectives of the project.

3. Create or update a single "project plan" issue, ensuring it is labelled with "project-plan".

   - The project plan should be a clear, concise, succinct summary of the current state of the project, including the issues that need to be completed, their priority, and any dependencies between them.

   - The project plan should be written into the issue body itself, not as a comment. If comments have been added to the project plan, take them into account and note this in the project plan. Never add comments to the project plan issue.

   - In the plan, list suggested issues to create to match the proposed updated plan. Don't create any issues, just list the suggestions. Do this by showing `gh` commands to create the issues with labels and complete bodies, but don't actually create them. Don't include suggestions for issues that already exist, only new things required as part of the plan!

   - Do not create any other issues, just the project plan issue. Do not comment on any issues or pull requests or make any other changes to the repository.

@include agentics/shared/tool-refused.md

@include agentics/shared/include-link.md

@include agentics/shared/job-summary.md

@include agentics/shared/xpia.md

@include agentics/shared/gh-extra-tools.md

