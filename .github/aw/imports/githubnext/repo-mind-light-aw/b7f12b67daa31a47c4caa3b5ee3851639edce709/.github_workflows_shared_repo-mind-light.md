---
# Repo Mind Light - Shared Workflow
# Reusable Repo Mind Light integration for gh-aw workflows.
#
# This repository is meant to be understandable by both humans and agents.
# The comments in this file are intentionally detailed because agents may read
# this file directly when deciding how to compose a consumer workflow.
#
# This shared workflow bundles four concerns behind one local import:
# 1. incremental index preparation in a dedicated job
# 2. artifact handoff into the agent job
# 3. MCP server startup, readiness checks, and cleanup
# 4. prompt guidance telling the agent to use Repo Mind Light first
#
# Operational summary:
# - Consumer workflows provide a Repo Mind Light config via `config.yaml`.
# - This shared workflow writes that config to `.repo-mind-light.config.yml`.
# - The prep job restores or refreshes `.index-store` and uploads it as an
#   artifact for the agent job.
# - The agent job downloads the artifact, starts Repo Mind Light on port 8000,
#   waits for preload readiness, then exposes only the `query` MCP tool.
#
# Retention and cache behavior:
# - Uploaded index artifacts use `retention-days: 1`.
# - Cache entries are immutable and are only saved when the structured
#   `repo-mind-light index --result-json ...` output reports at least one
#   refreshed source.
# - Cache eviction after that is controlled by GitHub Actions cache policy.
#
# Consumer responsibilities:
# - Provide `COPILOT_GITHUB_TOKEN`. Repo Mind Light itself uses Copilot-backed
#   model access for embeddings and final answer synthesis, regardless of which
#   outer agent engine the consumer workflow uses.
# - Provide GitHub read permissions suitable for the configured indexing scope.
# - Use a runner with Docker, `jq`, `curl`, and the ability to bind port 8000.
# - Keep event-specific gating policy in the consumer workflow and pass it through
#   the `run-if` input when Repo Mind Light should only run for selected events.
# - Set `sandbox.mcp.env.MCP_GATEWAY_TOOL_TIMEOUT` in each consumer workflow for
#   the MCP gateway read timeout. `tools.timeout` and `tools.startup-timeout`
#   control agent-side tool execution and startup waits.
# - Mention Repo Mind Light explicitly in the consumer workflow's task prompt
#   when it should be a required evidence source. This import injects generic
#   tool guidance, but workflow-specific instructions make usage more reliable.
#
# Recommended `config.yaml` fields for most workflows:
#
#   slug: owner/repo                           # required
#   store_path: /var/lib/repo-mind-light/index
#   refresh_if_older_than: 1d                 # always | never | 3600 | 15m | 6h | 1d
#   conversations:
#     keep_count: 1000
#     issue_state: all                        # open | closed | all | none | null
#     pr_state: all                           # open | closed | merged | all | none | null
#     discussion_state: all                   # all | none | null
#     issue_labels: null                      # optional any-of filter
#     pr_labels: null                         # optional any-of filter
#     discussion_categories: null             # optional any-of filter
#     ignore_bot_authored: true
#   wiki: null                                # opt-in; set to {} or filters to index GitHub wiki pages
#   query:
#     preload_query_sources_on_startup: true
#     graph_rag_zero:
#       top_k_initial: 1000
#       max_chunks: 200
#       relative_dropoff_threshold: 0.1
#       knn_neighbors: 5
#       max_cluster_size: 10
#     code_search:
#       enabled: true
#       limit: 50
#
# Wiki indexing is disabled by default. `wiki: {}` enables GitHub wiki indexing
# with Repo Mind Light's default path filters for common text formats and skips
# sidebar/footer pages. Use `wiki.include_paths` and `wiki.exclude_paths` glob
# lists to narrow which wiki pages are cloned, embedded, and made queryable.
#
# Query guidance:
# - Use task-provided context to form a focused query before broad repository
#   exploration or final decisions.
# - Good inputs are exact errors, feature names, ownership hints, downloaded
#   issue data, candidate themes, or requests for similar regressions and
#   concrete relevant code areas.
# - `preload_query_sources_on_startup: true` reduces first-query latency by
#   warming preloadable sources after server startup.
# - The `query` tool is for repository-context retrieval, not arbitrary remote
#   browsing.
# - Consumer prompts should say when Repo Mind Light is required, what evidence
#   to ask for, and how to handle no useful matches.

import-schema:
  config:
    type: object
    required: true
    description: >
      Repo Mind Light configuration object. Put the full YAML content in the
      required `yaml` property. This indirection exists because gh-aw import
      substitution is reliable for object inputs in step env blocks but not for
      raw multiline string inputs embedded directly in frontmatter.
    properties:
      yaml:
        type: string
        required: true
        description: >
          Full Repo Mind Light YAML configuration content. The workflow writes
          this to .repo-mind-light.config.yml in both the prep job and the
          agent job. Typical fields include slug, refresh_if_older_than,
          conversations filters, optional wiki settings, and query settings.
  image:
    type: string
    required: false
    default: ghcr.io/githubnext/repo-mind-light@sha256:bebe617c092bf1ed38754fa91f717e55d03e802d2372f2d72e2edad40442e864
    description: >
      Repo Mind Light container image reference. Override this to use a newer
      release or test a different image intentionally.
  cache-prefix:
    type: string
    required: false
    default: repo-mind-light-index-
    description: >
      Prefix used when deriving refreshed index cache keys. Defaults are safe
      for most consumers.
  cache-restore-key:
    type: string
    required: false
    default: repo-mind-light-index-restore
    description: >
      Primary restore key used for the index cache restore step. Useful when a
      consumer wants a workflow-specific cache namespace.
  container-name:
    type: string
    required: false
    default: repo-mind-light-mcp
    description: >
      Docker container name used for the agent-job Repo Mind Light MCP server.
      Override only if the consumer workflow might collide with another
      container name.
  artifact-name:
    type: string
    required: false
    default: repo-mind-light-index
    description: >
      Base artifact name used for the prepared Repo Mind Light index. gh-aw's
      activation artifact prefix is added automatically to avoid collisions.
  run-if:
    type: string
    required: false
    default: 'true'
    description: >
      GitHub Actions expression used as the job-level condition for the Repo
      Mind Light prep job. Defaults to true. Consumers can pass an event or
      label condition to skip indexing and the dependent agent job when Repo
      Mind Light should not run.

jobs:
  repo-mind-light-prep:
    name: Prepare Repo Mind Light index
    if: ${{ github.aw.import-inputs.run-if }}
    runs-on: ubuntu-latest
    needs: [activation]
    permissions:
      contents: read
      issues: read
      pull-requests: read
    steps:
      - name: Checkout repository
        uses: actions/checkout@v6.0.2

      - name: Write Repo Mind Light config
        env:
          REPO_MIND_LIGHT_CONFIG_MAP: ${{ github.aw.import-inputs.config }}
        run: |
          set -euo pipefail
          if printf '%s' "$REPO_MIND_LIGHT_CONFIG_MAP" | jq -e . >/dev/null 2>&1; then
            config_yaml="$(printf '%s' "$REPO_MIND_LIGHT_CONFIG_MAP" | jq -re '.yaml | select(type == "string")')"
          else
            config_yaml="${REPO_MIND_LIGHT_CONFIG_MAP#map[yaml:}"
            config_yaml="${config_yaml%]}"
          fi
          printf '%s\n' "$config_yaml" > .repo-mind-light.config.yml

      - name: Restore Repo Mind Light index cache
        uses: actions/cache/restore@v5.0.5
        with:
          path: .index-store
          key: ${{ github.aw.import-inputs.cache-restore-key }}
          restore-keys: |
            ${{ github.aw.import-inputs.cache-prefix }}

      - name: Ensure index directory exists
        run: |
          mkdir -p .index-store
          chmod -R a+rwX .index-store

      - name: Pull Repo Mind Light image
        env:
          REPO_MIND_LIGHT_IMAGE: ${{ github.aw.import-inputs.image }}
        run: docker pull "$REPO_MIND_LIGHT_IMAGE"

      - name: Run incremental indexing
        id: incremental-index
        env:
          COPILOT_GITHUB_TOKEN: ${{ secrets.COPILOT_GITHUB_TOKEN }}
          GITHUB_TOKEN: ${{ github.token }}
          REPO_MIND_LIGHT_CONFIG: /tmp/repo-mind-light.config.yml
          REPO_MIND_LIGHT_CACHE_PREFIX: ${{ github.aw.import-inputs.cache-prefix }}
          REPO_MIND_LIGHT_IMAGE: ${{ github.aw.import-inputs.image }}
        run: |
          set -euo pipefail
          result_json_dir="$RUNNER_TEMP/repo-mind-light-index"
          result_json_path="$result_json_dir/result.json"

          test -n "$COPILOT_GITHUB_TOKEN" || {
            echo "COPILOT_GITHUB_TOKEN secret is required" >&2
            exit 1
          }

          rm -rf "$result_json_dir"
          mkdir -p "$result_json_dir"
          chmod -R a+rwX "$result_json_dir"

          docker run --rm \
            -v "$PWD/.index-store:/var/lib/repo-mind-light/index" \
            -v "$result_json_dir:/tmp/repo-mind-light-result" \
            -v "$PWD/.repo-mind-light.config.yml:/tmp/repo-mind-light.config.yml:ro" \
            -e COPILOT_GITHUB_TOKEN \
            -e GITHUB_TOKEN \
            -e REPO_MIND_LIGHT_CONFIG \
            "$REPO_MIND_LIGHT_IMAGE" \
            index --result-json /tmp/repo-mind-light-result/result.json

          test -f "$result_json_path" || {
            echo "Repo Mind Light did not write the structured result JSON" >&2
            exit 1
          }

          refreshed="$(jq -r '
            if (.refreshed | type) == "boolean" then
              .refreshed
            elif (.sources | type) == "array" then
              any(.sources[]; .refreshed == true)
            else
              error("missing refreshed")
            end
          ' "$result_json_path")"
          echo "changed=${refreshed}" >> "$GITHUB_OUTPUT"

          if test "$refreshed" = 'true'; then
            cache_key="$(jq -re --arg prefix "$REPO_MIND_LIGHT_CACHE_PREFIX" '
              if (.last_refreshed_at | type) == "string" then
                .last_refreshed_at
              elif (.sources | type) == "array" then
                [
                  .sources[]
                  | select(.refreshed == true and (.last_refreshed_at | type) == "string")
                  | .last_refreshed_at
                ] | max
              else
                null
              end
              | if type == "string" and length > 0
                then $prefix + (split("T")[0])
                else error("missing last_refreshed_at")
                end
            ' "$result_json_path")"
            echo "cache_key=${cache_key}" >> "$GITHUB_OUTPUT"
          fi

      - name: Save refreshed Repo Mind Light index cache
        if: steps.incremental-index.outputs.changed == 'true'
        uses: actions/cache/save@v5.0.5
        with:
          path: .index-store
          key: ${{ steps.incremental-index.outputs.cache_key }}

      - name: Upload Repo Mind Light index artifact
        uses: actions/upload-artifact@v7.0.1
        with:
          name: ${{ needs.activation.outputs.artifact_prefix }}${{ github.aw.import-inputs.artifact-name }}
          path: .index-store
          if-no-files-found: error
          include-hidden-files: true
          retention-days: '1'

mcp-servers:
  repo-mind:
    url: http://127.0.0.1:8000/mcp
    allowed:
      - query

tools:
  timeout: 600
  startup-timeout: 600

pre-agent-steps:
  - name: Download Repo Mind Light index artifact
    uses: actions/download-artifact@v8.0.1
    with:
      name: ${{ needs.activation.outputs.artifact_prefix }}${{ github.aw.import-inputs.artifact-name }}
      path: .index-store

  - name: Ensure Repo Mind Light directories exist
    run: |
      mkdir -p .index-store /tmp/gh-aw/mcp-logs
      chmod -R a+rwX .index-store /tmp/gh-aw/mcp-logs

  - name: Write Repo Mind Light config
    env:
      REPO_MIND_LIGHT_CONFIG_MAP: ${{ github.aw.import-inputs.config }}
    run: |
      set -euo pipefail
      if printf '%s' "$REPO_MIND_LIGHT_CONFIG_MAP" | jq -e . >/dev/null 2>&1; then
        config_yaml="$(printf '%s' "$REPO_MIND_LIGHT_CONFIG_MAP" | jq -re '.yaml | select(type == "string")')"
      else
        config_yaml="${REPO_MIND_LIGHT_CONFIG_MAP#map[yaml:}"
        config_yaml="${config_yaml%]}"
      fi
      printf '%s\n' "$config_yaml" > .repo-mind-light.config.yml

  - name: Start Repo Mind Light MCP server
    env:
      COPILOT_GITHUB_TOKEN: ${{ secrets.COPILOT_GITHUB_TOKEN }}
      GITHUB_TOKEN: ${{ github.token }}
      REPO_MIND_LIGHT_CONTAINER_NAME: ${{ github.aw.import-inputs.container-name }}
      REPO_MIND_LIGHT_IMAGE: ${{ github.aw.import-inputs.image }}
    run: |
      set -euo pipefail
      docker rm -f "$REPO_MIND_LIGHT_CONTAINER_NAME" >/dev/null 2>&1 || true
      docker pull "$REPO_MIND_LIGHT_IMAGE"

      docker run -d \
        --name "$REPO_MIND_LIGHT_CONTAINER_NAME" \
        -p 8000:8000 \
        -v "$PWD/.index-store:/var/lib/repo-mind-light/index" \
        -v "$PWD/.repo-mind-light.config.yml:/tmp/repo-mind-light.config.yml:ro" \
        -e COPILOT_GITHUB_TOKEN \
        -e GITHUB_TOKEN \
        -e REPO_MIND_LIGHT_CONFIG=/tmp/repo-mind-light.config.yml \
        "$REPO_MIND_LIGHT_IMAGE"

  - name: Capture initial Repo Mind Light logs
    env:
      REPO_MIND_LIGHT_CONTAINER_NAME: ${{ github.aw.import-inputs.container-name }}
    run: |
      docker logs "$REPO_MIND_LIGHT_CONTAINER_NAME" \
        > /tmp/gh-aw/mcp-logs/repo-mind-light-startup.log 2>&1 || true

  - name: Wait for Repo Mind Light preload readiness
    timeout-minutes: 10
    env:
      REPO_MIND_LIGHT_CONTAINER_NAME: ${{ github.aw.import-inputs.container-name }}
    run: |
      set -euo pipefail
      status_file='/tmp/gh-aw/mcp-logs/repo-mind-light-preload-status.json'
      status_url='http://127.0.0.1:8000/preload-status'

      for attempt in $(seq 1 120); do
        if ! docker ps --format '{{.Names}}' | grep -Fxq "$REPO_MIND_LIGHT_CONTAINER_NAME"; then
          echo 'Repo Mind Light MCP container exited before preload completed.' >&2
          docker logs "$REPO_MIND_LIGHT_CONTAINER_NAME" >&2 || true
          exit 1
        fi

        if response="$(curl --silent --fail --max-time 5 "$status_url" 2>/dev/null)"; then
          printf '%s\n' "$response" > "$status_file"
          ready="$(jq -r '.ready' "$status_file")"
          state="$(jq -r '.state' "$status_file")"
          echo "Repo Mind Light preload state: ${state}"

          if test "$ready" = 'true'; then
            echo 'Repo Mind Light preload completed.'
            exit 0
          fi

          if test "$state" = 'failed' || test "$state" = 'cancelled'; then
            echo "Repo Mind Light preload ended in unexpected state: ${state}" >&2
            docker logs "$REPO_MIND_LIGHT_CONTAINER_NAME" >&2 || true
            exit 1
          fi
        fi

        sleep 5
      done

      echo 'Timed out waiting for Repo Mind Light preload readiness.' >&2
      docker logs "$REPO_MIND_LIGHT_CONTAINER_NAME" >&2 || true
      exit 1

post-steps:
  - name: Capture Repo Mind Light logs
    if: always()
    env:
      REPO_MIND_LIGHT_CONTAINER_NAME: ${{ github.aw.import-inputs.container-name }}
    run: |
      mkdir -p /tmp/gh-aw/mcp-logs
      if docker ps -a --format '{{.Names}}' | grep -Fxq "$REPO_MIND_LIGHT_CONTAINER_NAME"; then
        docker logs "$REPO_MIND_LIGHT_CONTAINER_NAME" \
          > /tmp/gh-aw/mcp-logs/repo-mind-light.log 2>&1 || true
      fi

  - name: Stop Repo Mind Light MCP server
    if: always()
    env:
      REPO_MIND_LIGHT_CONTAINER_NAME: ${{ github.aw.import-inputs.container-name }}
    run: docker rm -f "$REPO_MIND_LIGHT_CONTAINER_NAME" || true
---

# Repo Mind Light

Use Repo Mind Light for repository-context retrieval before broad repo exploration or final decisions.

This import exposes a repository-scoped `query` tool through the `repo-mind` MCP server.

The server reads from an index built from the repository configured in `config.yaml`.

## Required Usage Pattern

1. Use task-provided context, such as an issue body, error message, downloaded data, or candidate theme, to form a focused query.
2. Make one focused `query` request before broad repository exploration or final decisions.
3. If the first result is sufficient, do not issue more Repo Mind Light queries.
4. Make at most one follow-up `query` only when the first result leaves a specific gap that matters to the task.

## Notes

- Repo Mind Light is available at the `repo-mind` MCP server with the `query` tool.
- Startup and runtime logs are written under `/tmp/gh-aw/mcp-logs/` for debugging.
- Use Repo Mind Light when you need repository-local context such as similar incidents, relevant code areas, ownership hints, or related implementation history.
- Prefer focused, discriminating queries over broad prompts.
- Consumer workflows should still mention Repo Mind Light explicitly in their own task instructions when they require it as an evidence source; this shared prompt guidance is a baseline, not a substitute for workflow-specific direction.
- This shared import enables Repo Mind Light for the whole workflow by default. If a consumer needs event-specific gating, pass a GitHub Actions expression through the `run-if` input.
