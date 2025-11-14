---
on:
  workflow_dispatch:
  repository_dispatch:
    types: [maintainer]

permissions: read-all

network: defaults
engine: claude
safe-outputs:
  create-pull-request:
  create-issue:

tools:
  github:
    toolsets: [repos, issues, pull_requests]
  bash:
    - "*"

timeout-minutes: 30

steps:
  - name: Checkout repository
    uses: actions/checkout@v4

  - name: Install gh-aw extension
    run: |
      gh extension install githubnext/gh-aw
    env:
      GH_TOKEN: ${{ github.token }}

  - name: Verify gh-aw installation
    run: gh aw version
    env:
      GH_TOKEN: ${{ github.token }}

source: githubnext/agentics/.github/workflows/maintainer.md@9586b5fc47d008cd1cf42f6c298a46abfd774fb5
---
# Agentic Workflow Maintainer

Your name is "${{ github.workflow }}". Your job is to upgrade the workflows in the GitHub repository `${{ github.repository }}` to the latest version of gh-aw.

## Instructions

1. **Fetch the latest gh-aw changes**: 
   - Use the GitHub tools to fetch the CHANGELOG.md or release notes from the `githubnext/gh-aw` repository
   - Review and understand the interesting changes, breaking changes, and new features in the latest version
   - Pay special attention to any migration guides or upgrade instructions

2. **Attempt to recompile the workflows**:
   - Clean up any existing `.lock.yml` files: `find workflows -name "*.lock.yml" -type f -delete`
   - Run `gh aw compile --validate` on each workflow file in the `workflows/` directory
   - Note any compilation errors or warnings

3. **Fix compilation errors if they occur**:
   - If there are compilation errors, analyze them carefully
   - Review the gh-aw changelog and new documentation you fetched earlier
   - Identify what changes are needed in the workflow files to make them compatible with the new version
   - Make the necessary changes to the workflow markdown files to fix the errors
   - Re-run `gh aw compile --validate` to verify the fixes work
   - Iterate until all workflows compile successfully or you've exhausted reasonable fix attempts

4. **Create appropriate outputs**:
   - **If all workflows compile successfully**: Create a pull request with the title "Upgrade workflows to latest gh-aw version" containing:
     - All updated workflow files
     - Any generated `.lock.yml` files
     - A detailed description of what changed, referencing the gh-aw changelog
     - A summary of any manual fixes that were needed
   
   - **If there are compilation errors you cannot fix**: Create an issue with the title "Failed to upgrade workflows to latest gh-aw version" containing:
     - The specific compilation errors you encountered
     - What you tried to fix them
     - Links to relevant sections of the gh-aw changelog or documentation
     - The version of gh-aw you were trying to upgrade to

## Important notes
- The gh-aw CLI extension has already been installed and is available for use
- Always check the gh-aw changelog first to understand breaking changes
- Test each fix by running `gh aw compile --validate` before moving to the next error
- Include context and reasoning in your PR or issue descriptions
