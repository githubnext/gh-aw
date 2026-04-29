---
# APM (Agent Package Manager) - Shared Workflow
# Install Microsoft APM packages in your agentic workflow.
#
# This shared workflow normalises packages, single-app inputs, and apps[] (multi-org
# GitHub App credential groups) into one canonical list of credential groups in an
# "apm-prep" job, then fans the "apm" job out one matrix replica per group. Each
# replica mints its own installation token (when an app-id is set), packs only its
# declared packages with microsoft/apm-action, and uploads a uniquely-named artifact.
# Pre-agent-steps then download all bundles and restore them in one apm-action call.
#
# Source of truth: https://github.com/microsoft/apm/blob/main/.github/workflows/shared/apm.md
# apm-action pin:  microsoft/apm-action@v1.5.0
# To check whether a vendored copy is current, compare these two lines.
#
# Documentation: https://microsoft.github.io/apm/integrations/gh-aw/
#
# Three user-facing forms (all valid, additive):
#
# 1. Public + default-token packages (no App credentials):
#
#    imports:
#      - uses: shared/apm.md
#        with:
#          packages:
#            - microsoft/apm-sample-package
#            - github/awesome-copilot/skills/review-and-refactor
#
# 2. Single GitHub App (one org) -- canonical shorthand:
#
#    imports:
#      - uses: shared/apm.md
#        with:
#          app-id: ${{ vars.APP_ID }}
#          private-key: ${{ secrets.APP_PRIVATE_KEY }}
#          owner: my-org
#          packages:
#            - my-org/my-private-skills
#
# 3. Multiple GitHub Apps (cross-org):
#
#    imports:
#      - uses: shared/apm.md
#        with:
#          packages:
#            - microsoft/apm-sample-package
#          apps:
#            - id: acme
#              app-id: ${{ vars.ACME_APP_ID }}
#              private-key: ${{ secrets.ACME_KEY }}
#              owner: acme-org
#              packages:
#                - acme-org/acme-skills/skills/code-review
#            - app-id: ${{ vars.BETA_APP_ID }}
#              private-key: ${{ secrets.BETA_KEY }}
#              owner: beta-org
#              packages:
#                - beta-org/beta-pkg

import-schema:
  packages:
    type: array
    items:
      type: string
    required: false
    description: >
      Public APM packages or packages reachable via the default token cascade
      (GH_AW_PLUGINS_TOKEN, GH_AW_GITHUB_TOKEN, GITHUB_TOKEN). Optional. At
      least one of `packages`, the single-app inputs, or `apps` must be provided.
      Format: owner/repo or owner/repo/path/to/skill.

  # Single-app convenience form (canonical shorthand for one-org users)
  app-id:
    type: string
    required: false
    description: >
      GitHub App ID. With `private-key`, mints an installation token for the
      packages listed in `packages:`. For multiple orgs, use `apps:` instead.
  private-key:
    type: string
    required: false
    description: >
      PEM private key matching `app-id`. Required when `app-id` is set. Pass via
      a repository or organization secret.
  owner:
    type: string
    required: false
    description: >
      App installation owner. Defaults to the current repository owner when
      omitted. Only used when `app-id` is set.
  repositories:
    type: string
    required: false
    description: >
      Repositories the minted token is scoped to. Comma- or newline-separated.
      Empty defaults to the calling repo or the App installation default scope.
      Note: literal "*" is NOT a wildcard for actions/create-github-app-token;
      leave empty for org-wide access via App installation config.

  # Multi-app form (cross-org)
  apps:
    type: array
    required: false
    description: >
      List of GitHub App credential groups. Each entry mints its own
      installation token and packs its own packages. Use when packages span
      multiple orgs requiring different App installations.
    items:
      type: object
      properties:
        id:
          type: string
          required: false
          description: >
            Stable identifier used for matrix-row and artifact naming.
            Auto-derived from `owner` (slugified) when omitted. Required when
            two entries share the same owner.
        app-id:
          type: string
          required: true
        private-key:
          type: string
          required: true
        owner:
          type: string
          required: false
        repositories:
          type: string
          required: false
        packages:
          type: array
          items:
            type: string
          required: true

jobs:
  apm-prep:
    runs-on: ubuntu-slim
    needs: [activation]
    permissions: {}
    outputs:
      matrix: ${{ steps.compute.outputs.matrix }}
    steps:
      # SECURITY (S3): never echo $groups, $matrix, or any matrix.group.* value
      # in any apm-prep step. private-key is a real secret string here.
      - name: Compute APM credential-group matrix
        id: compute
        env:
          AW_APM_PACKAGES: '${{ github.aw.import-inputs.packages }}'
          AW_APM_APPS: '${{ github.aw.import-inputs.apps }}'
          AW_APM_LEGACY_APP_ID: ${{ github.aw.import-inputs.app-id }}
          AW_APM_LEGACY_PRIVATE_KEY: ${{ github.aw.import-inputs.private-key }}
          AW_APM_LEGACY_OWNER: ${{ github.aw.import-inputs.owner }}
          AW_APM_LEGACY_REPOS: ${{ github.aw.import-inputs.repositories }}
        run: |
          set -euo pipefail
          packages_json=${AW_APM_PACKAGES:-null}
          apps_json=${AW_APM_APPS:-null}
          legacy_id=${AW_APM_LEGACY_APP_ID:-}

          groups=$(jq -nc \
            --argjson packages "$packages_json" \
            --argjson apps "$apps_json" \
            --arg legacy_id "$legacy_id" \
            --arg legacy_pk "${AW_APM_LEGACY_PRIVATE_KEY:-}" \
            --arg legacy_owner "${AW_APM_LEGACY_OWNER:-}" \
            --arg legacy_repos "${AW_APM_LEGACY_REPOS:-}" \
            'def slug(s): s | gsub("[^a-zA-Z0-9-]"; "-") | ascii_downcase | .[0:32];
             def with_id(g):
               g + (if (g.id // "") == "" then {id: ("auto-" + slug(g.owner // "default"))} else {} end);
             [
               (if (($packages // []) | length) > 0 and $legacy_id == ""
                  then [{id:"default",("app-id"):"",("private-key"):"",owner:"",repositories:"",packages:$packages}]
                  else [] end),
               (if $legacy_id != ""
                  then [with_id({id:"legacy",("app-id"):$legacy_id,("private-key"):$legacy_pk,owner:$legacy_owner,repositories:$legacy_repos,packages:($packages // [])})]
                  else [] end),
               (($apps // []) | map(with_id(.)))
             ] | add // []')

          count=$(echo "$groups" | jq 'length')
          if [ "$count" = "0" ]; then
            echo "::error::shared/apm.md import provided no packages. Add packages: <list>, single-app inputs (app-id + private-key), or apps: <list> in the with: block."
            exit 1
          fi

          dups=$(echo "$groups" | jq -r '[.[].id] | group_by(.) | map(select(length > 1) | first) | join(", ")')
          if [ -n "$dups" ]; then
            echo "::error::duplicate apm group ids after auto-derivation: $dups. Set apps[].id explicitly when two entries share the same owner."
            exit 1
          fi

          while IFS= read -r id; do
            if ! echo "$id" | grep -Eq '^[a-z0-9-]{1,32}$'; then
              echo "::error::invalid apm group id: '$id' (lowercase alphanumeric and dashes, 1-32 chars). Set apps[].id explicitly."
              exit 1
            fi
          done < <(echo "$groups" | jq -r '.[].id')

          # SAFE: emit only id + package-count to logs. Never $groups in full.
          {
            echo "matrix={\"group\":$groups}"
          } >> "$GITHUB_OUTPUT"
          printf "::notice::APM matrix: %d credential group(s)\n" "$count"
          echo "$groups" | jq -r '.[] | "  - " + .id + " (" + (.packages | length | tostring) + " package(s))"'

  apm:
    runs-on: ubuntu-slim
    needs: [activation, apm-prep]
    permissions: {}
    strategy:
      fail-fast: false
      matrix: ${{ fromJSON(needs.apm-prep.outputs.matrix) }}
    steps:
      - name: Mint installation token
        id: token
        if: ${{ matrix.group.app-id != '' }}
        uses: actions/create-github-app-token@v3.1.1
        with:
          app-id: ${{ matrix.group.app-id }}
          private-key: ${{ matrix.group.private-key }}
          owner: ${{ matrix.group.owner != '' && matrix.group.owner || github.repository_owner }}
          repositories: ${{ matrix.group.repositories }}
      - name: Render package list
        id: list
        env:
          AW_PKG: ${{ toJSON(matrix.group.packages) }}
        run: |
          DEPS=$(echo "$AW_PKG" | jq -r '.[] | "- " + .')
          {
            echo "deps<<APMDEPS"
            printf '%s\n' "$DEPS"
            echo "APMDEPS"
          } >> "$GITHUB_OUTPUT"
      - name: Pack APM packages
        id: pack
        uses: microsoft/apm-action@v1.5.0
        env:
          GITHUB_TOKEN: ${{ steps.token.outputs.token || secrets.GH_AW_PLUGINS_TOKEN || secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}
        with:
          dependencies: ${{ steps.list.outputs.deps }}
          isolated: 'true'
          pack: 'true'
          archive: 'true'
          target: all
          working-directory: /tmp/gh-aw/apm-workspace
      - name: Rename bundle to group-scoped filename
        id: rename-bundle
        env:
          BUNDLE_SRC: ${{ steps.pack.outputs.bundle-path }}
          GROUP_ID: ${{ matrix.group.id }}
          ARTIFACT_PREFIX: ${{ needs.activation.outputs.artifact_prefix }}
        run: |
          dst="$(dirname "$BUNDLE_SRC")/${ARTIFACT_PREFIX}apm-${GROUP_ID}.tar.gz"
          mv "$BUNDLE_SRC" "$dst"
          echo "bundle-path=$dst" >> "$GITHUB_OUTPUT"
      - name: Upload APM bundle artifact
        if: success()
        uses: actions/upload-artifact@v7.0.1
        with:
          name: ${{ needs.activation.outputs.artifact_prefix }}apm-${{ matrix.group.id }}
          path: ${{ steps.rename-bundle.outputs.bundle-path }}
          retention-days: '1'

pre-agent-steps:
  - name: Download APM bundle artifacts (all groups)
    uses: actions/download-artifact@v8.0.1
    with:
      pattern: ${{ needs.activation.outputs.artifact_prefix }}apm-*
      path: /tmp/gh-aw/apm-bundles
      merge-multiple: false
  - name: Validate downloaded bundles match matrix manifest
    env:
      EXPECTED_MATRIX: ${{ needs.apm-prep.outputs.matrix }}
      ARTIFACT_PREFIX: ${{ needs.activation.outputs.artifact_prefix }}
    run: |
      set -euo pipefail
      mapfile -t expected_names < <(echo "$EXPECTED_MATRIX" | jq -r --arg prefix "$ARTIFACT_PREFIX" '.group | map($prefix + "apm-" + .id) | sort | .[]')
      missing=()
      for name in "${expected_names[@]}"; do
        if ! find /tmp/gh-aw/apm-bundles -maxdepth 2 -name "${name}.tar.gz" | grep -q .; then
          missing+=("$name")
        fi
      done
      if [ ${#missing[@]} -gt 0 ]; then
        echo "::error::missing APM bundles (group did not pack successfully): ${missing[*]}"
        exit 1
      fi
      actual_count=$(find /tmp/gh-aw/apm-bundles -maxdepth 2 -name "${ARTIFACT_PREFIX}apm-*.tar.gz" | wc -l)
      expected_count=${#expected_names[@]}
      if [ "$actual_count" -gt "$expected_count" ]; then
        echo "::error::unexpected APM bundles found: $actual_count file(s) but only $expected_count expected (possible collision)"
        exit 1
      fi
  - name: Build bundle list
    id: bundles
    run: |
      set -euo pipefail
      mapfile -t list < <(find /tmp/gh-aw/apm-bundles -name '*.tar.gz' | sort)
      [ ${#list[@]} -gt 0 ] || { echo '::error::no apm bundles found'; exit 1; }
      printf '%s\n' "${list[@]}" > /tmp/gh-aw/apm-bundle-list.txt
  - name: Restore APM packages (all bundles)
    uses: microsoft/apm-action@v1.5.0
    with:
      bundles-file: /tmp/gh-aw/apm-bundle-list.txt
---

<!--
## APM Packages

This shared workflow installs APM packages in a dedicated `apm` job that runs
in parallel one matrix replica per credential group, packs each group's packages
with `microsoft/apm-action`, and uploads a per-group bundle artifact. The agent
job's pre-agent-steps then download all bundles and restore them in a single
`apm-action` invocation (using the `bundles-file:` input shipped in
`microsoft/apm-action@v1.5.0`).

### How it works

1. **Normalise** (`apm-prep` job): a small jq script merges `packages:`, the
   single-app top-level inputs, and `apps[]` into one canonical list of
   credential groups. Each group has an `id`, optional App credentials, and a
   `packages` list. The matrix size is the number of groups.
2. **Pack per group** (`apm` job, matrix fan-out): each replica conditionally
   mints an installation token (only if `app-id` is set), packs only its declared
   packages, and uploads `apm-<group-id>` as an artifact.
3. **Restore** (agent pre-agent-steps): all `apm-*` artifacts are downloaded,
   validated against the matrix manifest (defends against same-run artifact-name
   collision attacks), and restored in one call via the `bundles-file:` input
   on `microsoft/apm-action@v1.5.0`.

### Authentication

Three forms, additive:

- No App credentials: packages fetched via `GH_AW_PLUGINS_TOKEN || GH_AW_GITHUB_TOKEN || GITHUB_TOKEN`.
- Single App (top-level `app-id` + `private-key` + `owner` + `repositories`):
  one installation token mints for one credential group; canonical shorthand for
  one-org users.
- Multi App (`apps:` array): each entry mints its own installation token and
  packs only its declared packages, enabling cross-org scenarios where each org
  requires a different App installation.
-->
