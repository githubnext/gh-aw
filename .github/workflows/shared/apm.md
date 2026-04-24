---
# APM (Agent Package Manager) - Shared Workflow
# Install Microsoft APM packages in your agentic workflow.
#
# This shared workflow creates a dedicated "apm" job (depending on activation) that
# packs packages using microsoft/apm-action and caches the bundle. The agent job
# then restores and unpacks the bundle as pre-agent-steps.
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

jobs:
  apm:
    runs-on: ubuntu-slim
    needs: [activation]
    permissions: {}
    steps:
      - name: Checkout workflow lock files
        uses: actions/checkout@v6.0.2
        with:
          sparse-checkout: |
            .github/workflows
          sparse-checkout-cone-mode: false
          persist-credentials: false
      - name: Restore APM bundle from cache
        id: apm_cache
        uses: actions/cache/restore@v5.0.5
        with:
          path: /tmp/gh-aw/apm-workspace
          key: apm-${{ needs.activation.outputs.engine_id }}-${{ hashFiles('.github/workflows/*.lock.yml') }}
      - name: Prepare APM package list
        id: apm_prep
        if: steps.apm_cache.outputs.cache-hit != 'true'
        env:
          AW_APM_PACKAGES: '${{ github.aw.import-inputs.packages }}'
        run: |
          DEPS=$(echo "$AW_APM_PACKAGES" | jq -r '.[] | "- " + .')
          {
            echo "deps<<APMDEPS"
            printf '%s\n' "$DEPS"
            echo "APMDEPS"
          } >> "$GITHUB_OUTPUT"
      - name: Pack APM packages
        id: apm_pack
        if: steps.apm_cache.outputs.cache-hit != 'true'
        uses: microsoft/apm-action@v1.4.1
        env:
          GITHUB_TOKEN: ${{ secrets.GH_AW_PLUGINS_TOKEN || secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}
        with:
          dependencies: ${{ steps.apm_prep.outputs.deps }}
          isolated: 'true'
          pack: 'true'
          archive: 'true'
          target: all
          working-directory: /tmp/gh-aw/apm-workspace
      - name: Save APM bundle to cache
        if: steps.apm_cache.outputs.cache-hit != 'true' && success()
        uses: actions/cache/save@v5.0.5
        with:
          path: /tmp/gh-aw/apm-workspace
          key: ${{ steps.apm_cache.outputs.cache-primary-key }}

pre-agent-steps:
  - name: Restore APM bundle from cache
    uses: actions/cache/restore@v5.0.5
    with:
      path: /tmp/gh-aw/apm-workspace
      key: apm-${{ needs.activation.outputs.engine_id }}-${{ hashFiles('.github/workflows/*.lock.yml') }}
      fail-on-cache-miss: true
  - name: Find APM bundle path
    id: apm_bundle
    run: echo "path=$(find /tmp/gh-aw/apm-workspace -name '*.tar.gz' | head -1)" >> "$GITHUB_OUTPUT"
  - name: Restore APM packages
    uses: microsoft/apm-action@v1.4.1
    with:
      bundle: ${{ steps.apm_bundle.outputs.path }}
---

<!--
## APM Packages

These packages are installed via a dedicated "apm" job that packs and caches a bundle,
which the agent job then restores and unpacks as pre-agent-steps.

### How it works

1. **Pack** (`apm` job): checks for a cached bundle keyed by lock file hash + engine ID.
   On a cache miss, `microsoft/apm-action` installs packages and creates a bundle archive,
   which is saved to the cache for reuse.
2. **Unpack** (agent job pre-agent-steps): the bundle is restored from cache and unpacked
   via `microsoft/apm-action` in restore mode, making all skills and tools available to the AI agent.

### Cache key

The cache key is `apm-{engine_id}-{hash_of_lock_files}`, derived from:
- `needs.activation.outputs.engine_id` — the AI engine identifier (e.g. `copilot`, `claude`)
- `hashFiles('.github/workflows/*.lock.yml')` — hash of all compiled workflow lock files

This ensures the bundle is refreshed whenever the workflow configuration changes or the
engine changes, while being reused across runs with identical configuration.

### Package format

Packages use the format `owner/repo` or `owner/repo/path/to/skill`:
- `microsoft/apm-sample-package` — organization/repository
- `github/awesome-copilot/skills/review-and-refactor` — organization/repository/path

### Authentication

Packages are fetched using the cascading token fallback:
`GH_AW_PLUGINS_TOKEN || GH_AW_GITHUB_TOKEN || GITHUB_TOKEN`
-->
