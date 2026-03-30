---
# APM (Agent Package Manager) - Shared Workflow
# Install Microsoft APM packages in your agentic workflow.
#
# This shared workflow wraps the Microsoft APM action to install and pack
# agent skill packages into an artifact that the agent job restores at runtime.
# It creates a dedicated packing job, uploads the bundle as a workflow artifact,
# and automatically restores it before the agent runs.
#
# Documentation: https://github.com/microsoft/APM
#
# Usage (simple array form):
#   imports:
#     - uses: shared/apm.md
#       with:
#         packages:
#           - microsoft/apm-sample-package
#           - github/awesome-copilot/skills/review-and-refactor
#
# Usage (with private package authentication):
#   imports:
#     - uses: shared/apm.md
#       with:
#         packages:
#           - myorg/private-skills
#         github-token: ${{ secrets.MY_PAT }}
#
# Usage (with GitHub App for cross-org private packages):
#   imports:
#     - uses: shared/apm.md
#       with:
#         packages:
#           - partner-org/their-skills
#         github-app:
#           app-id: ${{ vars.APP_ID }}
#           private-key: ${{ secrets.APP_PRIVATE_KEY }}
#
# Usage (with isolated restore — clears primitive dirs before unpacking):
#   imports:
#     - uses: shared/apm.md
#       with:
#         packages:
#           - microsoft/apm-sample-package
#         isolated: true

import-schema:
  packages:
    type: array
    items:
      type: string
    required: true
    description: >
      List of APM package references to install.
      Format: owner/repo or owner/repo/path/to/skill.
      Examples: microsoft/apm-sample-package, github/awesome-copilot/skills/review-and-refactor
  github-token:
    type: string
    required: false
    description: >
      GitHub token for accessing private package repositories.
      Defaults to cascading fallback: GH_AW_PLUGINS_TOKEN || GH_AW_GITHUB_TOKEN || GITHUB_TOKEN.
  isolated:
    type: boolean
    required: false
    description: >
      If true, clears primitive directories before unpacking the APM bundle.
      Use when you want a clean agent environment. Default: false.

apm-packages:
  packages: ${{ github.aw.import-inputs.packages }}
  github-token: ${{ github.aw.import-inputs.github-token }}
  isolated: ${{ github.aw.import-inputs.isolated }}
---

<!--
## APM Packages

The following APM packages are configured for this workflow:
- **Packages**: `${{ github.aw.import-inputs.packages }}`

These packages are installed and packed by a dedicated `apm` job that runs before the agent.
The packed bundle is uploaded as a workflow artifact and automatically restored at the start of
the agent job, making all package skills and tools available to the AI agent.

### How it works

1. **Pack job** (`apm`): Microsoft APM action installs the packages and packs them into a
   `.tar.gz` bundle, which is uploaded as the `apm` workflow artifact.
2. **Restore** (agent job): The bundle is downloaded and unpacked into the agent environment
   before the AI model runs, providing it with all configured skills and tools.

### Package format

Packages use the format `owner/repo` or `owner/repo/path/to/skill`:
- `microsoft/apm-sample-package` — organization/repository
- `github/awesome-copilot/skills/review-and-refactor` — organization/repository/path

### Authentication

By default, packages are fetched using the cascading token fallback:
`GH_AW_PLUGINS_TOKEN || GH_AW_GITHUB_TOKEN || GITHUB_TOKEN`

For private packages requiring a specific token, pass `github-token` in the `with:` block.
For cross-organization private packages, use a GitHub App via `imports.apm-packages` in your
main workflow frontmatter with the `github-app` configuration option.
-->
