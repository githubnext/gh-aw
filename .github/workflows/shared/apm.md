---
# APM (Agent Package Manager) - Shared Workflow
# Install Microsoft APM packages in your agentic workflow.
#
# This shared workflow installs APM packages as pre-steps in the agent job before
# the AI model runs. It is fully self-contained — no special compiler support needed.
# It packs the packages locally, stages the bundle, then unpacks it into the workspace.
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

steps:
  - name: Prepare APM package list
    id: apm_prep
    run: |
      PACKAGES='${{ github.aw.import-inputs.packages }}'
      DEPS=$(echo "$PACKAGES" | jq -r '.[] | "- " + .')
      {
        echo "deps<<APMDEPS"
        printf '%s\n' "$DEPS"
        echo "APMDEPS"
      } >> "$GITHUB_OUTPUT"
  - name: Pack APM packages
    id: apm_pack
    uses: microsoft/apm-action@v1.4.1
    env:
      GITHUB_TOKEN: ${{ secrets.GH_AW_PLUGINS_TOKEN || secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}
    with:
      dependencies: ${{ steps.apm_prep.outputs.deps }}
      isolated: 'false'
      pack: 'true'
      archive: 'true'
      target: all
      working-directory: /tmp/gh-aw/apm-workspace
      apm-version: v0.8.6
  - name: Stage APM bundle for restore
    env:
      APM_BUNDLE_PATH: ${{ steps.apm_pack.outputs.bundle-path }}
    run: |
      mkdir -p "${RUNNER_TEMP}/gh-aw/apm-bundle"
      cp "$APM_BUNDLE_PATH" "${RUNNER_TEMP}/gh-aw/apm-bundle/"
  - name: Restore APM packages
    uses: actions/github-script@v8
    env:
      APM_BUNDLE_DIR: ${{ runner.temp }}/gh-aw/apm-bundle
    with:
      script: |
        const { setupGlobals } = require(`${process.env.RUNNER_TEMP}/gh-aw/actions/setup_globals.cjs`);
        setupGlobals(core, github, context, exec, io);
        const { main } = require(`${process.env.RUNNER_TEMP}/gh-aw/actions/apm_unpack.cjs`);
        await main();
---

<!--
## APM Packages

These packages are installed as pre-steps in the agent job before the AI model runs.

### How it works

1. **Prepare**: Converts the JSON array of packages to YAML dependency format.
2. **Pack**: `microsoft/apm-action` installs packages and creates a local bundle archive.
3. **Stage**: The bundle is copied to the expected restore location.
4. **Restore**: `apm_unpack.cjs` unpacks the bundle into the workspace, making all
   skills and tools available to the AI agent.

### Package format

Packages use the format `owner/repo` or `owner/repo/path/to/skill`:
- `microsoft/apm-sample-package` — organization/repository
- `github/awesome-copilot/skills/review-and-refactor` — organization/repository/path

### Authentication

Packages are fetched using the cascading token fallback:
`GH_AW_PLUGINS_TOKEN || GH_AW_GITHUB_TOKEN || GITHUB_TOKEN`
-->
