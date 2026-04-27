---
# APM (Agent Package Manager) - Shared Workflow
# Install Microsoft APM packages in your agentic workflow.
#
# This shared workflow creates a dedicated "apm" job (depending on activation) that
# packs packages using microsoft/apm-action, caches the bundle for cross-run reuse,
# and uploads it as an artifact for reliable same-run access.
# The agent job restores from cache (preferred) or downloads the artifact as a fallback.
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
        uses: microsoft/apm-action@v1.4.2
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
      - name: Find APM bundle path
        id: apm_bundle_path
        run: |
          bundle=$(find /tmp/gh-aw/apm-workspace -name '*.tar.gz' | head -1)
          if [ -z "$bundle" ]; then
            echo "::error::APM bundle not found in /tmp/gh-aw/apm-workspace"
            exit 1
          fi
          echo "path=$bundle" >> "$GITHUB_OUTPUT"
      - name: Upload APM bundle artifact
        if: success()
        uses: actions/upload-artifact@v7.0.1
        with:
          name: ${{ needs.activation.outputs.artifact_prefix }}apm
          path: ${{ steps.apm_bundle_path.outputs.path }}
          retention-days: '1'

pre-agent-steps:
  - name: Restore APM bundle from cache
    id: apm_cache_restore
    uses: actions/cache/restore@v5.0.5
    with:
      path: /tmp/gh-aw/apm-workspace
      key: apm-${{ needs.activation.outputs.engine_id }}-${{ hashFiles('.github/workflows/*.lock.yml') }}
  - name: Download APM bundle artifact
    if: steps.apm_cache_restore.outputs.cache-hit != 'true'
    uses: actions/download-artifact@v8.0.1
    with:
      name: ${{ needs.activation.outputs.artifact_prefix }}apm
      path: /tmp/gh-aw/apm-bundle
  - name: Find APM bundle path
    id: apm_bundle
    run: |
      bundle=$(find /tmp/gh-aw/apm-workspace /tmp/gh-aw/apm-bundle -name '*.tar.gz' 2>/dev/null | head -1)
      if [ -z "$bundle" ]; then
        echo "::error::APM bundle not found in /tmp/gh-aw/apm-workspace or /tmp/gh-aw/apm-bundle"
        exit 1
      fi
      echo "path=$bundle" >> "$GITHUB_OUTPUT"
  - name: Restore APM packages
    uses: microsoft/apm-action@v1.4.2
    with:
      bundle: ${{ steps.apm_bundle.outputs.path }}
---

<!--
## APM Packages

These packages are installed via a dedicated "apm" job that packs the bundle, saves it to
cache for cross-run reuse, and uploads it as an artifact for reliable same-run access.
The agent job restores from cache (preferred) or falls back to downloading the artifact.

### How it works

1. **Pack** (`apm` job): checks for a cached bundle keyed by lock file hash + engine ID.
   On a cache miss, `microsoft/apm-action` installs packages and creates a bundle archive,
   which is saved to the cache and uploaded as an artifact.
   On a cache hit, the cached bundle is uploaded directly as an artifact.
2. **Unpack** (agent job pre-agent-steps): the bundle is restored from cache (preferred for
   speed and cross-run reuse) or downloaded from the artifact (reliable same-run fallback),
   then unpacked via `microsoft/apm-action` in restore mode.

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
