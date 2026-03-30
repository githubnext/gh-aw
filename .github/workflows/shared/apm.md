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
# Usage:
#   imports:
#     - uses: shared/apm.md
#       with:
#         packages:
#           - microsoft/apm-sample-package
#           - github/awesome-copilot/skills/review-and-refactor

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

dependencies:
  packages: ${{ github.aw.import-inputs.packages }}
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

For private packages requiring a specific token, use `dependencies.github-token` directly
in your main workflow frontmatter:

```yaml
dependencies:
  packages:
    - myorg/private-skills
  github-token: YOUR_GITHUB_TOKEN_SECRET
```

For cross-organization private packages using a GitHub App, configure `dependencies.github-app`
in your main workflow frontmatter.
-->
