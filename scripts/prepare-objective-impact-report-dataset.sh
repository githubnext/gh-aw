#!/bin/bash
set -euo pipefail

cd "$(dirname "$0")/.."

DATA_DIR=/tmp/gh-aw/agent/objective-impact-report
mkdir -p "$DATA_DIR"

has_data_file() {
  local path="$1"
  [ -s "$path" ]
}

repo="${EXPR_GITHUB_REPOSITORY:-${GITHUB_REPOSITORY:-}}"
if [ -z "$repo" ]; then
  echo "EXPR_GITHUB_REPOSITORY or GITHUB_REPOSITORY is required" >&2
  exit 1
fi

run_id="${GITHUB_RUN_ID:-}"
server_url="${GITHUB_SERVER_URL:-https://github.com}"
window_start=$(date -u -d '180 days ago' '+%Y-%m-%d' 2>/dev/null || date -u -v-180d '+%Y-%m-%d')
generated_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)

cat > "$DATA_DIR/run-context.json" <<EOF
{
  "repository": "${repo}",
  "generated_at": "${generated_at}",
  "window_start": "${window_start}",
  "window_days": 180,
  "run_id": "${run_id}",
  "run_url": "${server_url}/${repo}/actions/runs/${run_id}"
}
EOF

if [ -f .github/objective-mapping.json ]; then
  cp .github/objective-mapping.json "$DATA_DIR/objective-mapping.json"
else
  printf '%s\n' '{}' > "$DATA_DIR/objective-mapping.json"
fi

logs_source="gh-aw-logs"
if has_data_file "$DATA_DIR/workflow-logs.json"; then
  echo "Using cached workflow logs dataset"
else
  if gh aw logs --help >/dev/null 2>&1; then
    if ! gh aw logs --repo "$repo" --start-date -180d --json > "$DATA_DIR/workflow-logs.json"; then
      logs_source="gh-api-fallback"
    fi
  else
    logs_source="gh-api-fallback"
  fi
fi

if [ "$logs_source" = "gh-api-fallback" ]; then
  gh api --paginate "repos/$repo/actions/runs?per_page=100" \
    --jq '.workflow_runs[] | select((.created_at // "") >= "'"$window_start"'T00:00:00Z") | {id, workflow_name: .name, display_title, path, created_at, run_started_at, updated_at, status, conclusion, html_url, event, head_branch, actor: (.actor.login // null), aic: null}' \
    | jq -s '{source:"gh-api-fallback", runs:.}' > "$DATA_DIR/workflow-logs.json"
fi

# Aggregate per-workflow AIC from daily token-audit memory snapshots.
# Each daily snapshot in the memory/token-audit branch covers ~24 hours of runs.
# Summing across all snapshots in the window gives total AIC per workflow.
aic_snapshot_count=0
if has_data_file "$DATA_DIR/aic-by-workflow.json"; then
  echo "Using cached AIC by workflow dataset"
  aic_snapshot_count=$(jq '.snapshot_count // 0' "$DATA_DIR/aic-by-workflow.json" 2>/dev/null || echo 0)
else
  echo "Fetching token-audit memory snapshots for AIC aggregation..."
  if git fetch origin "memory/token-audit:refs/remotes/origin/memory/token-audit" --no-tags 2>/dev/null; then
    mapfile -t snapshot_files < <(
      git ls-tree --name-only origin/memory/token-audit \
        | grep -E '^[0-9]{4}-[0-9]{2}-[0-9]{2}\.json$' \
        | awk -F. '{print $1}' \
        | awk -v ws="$window_start" '$0 >= ws' \
        | sed 's/$/.json/'
    )
    aic_snapshot_count="${#snapshot_files[@]}"
    echo "Found $aic_snapshot_count token-audit snapshots in the window"

    if [ "$aic_snapshot_count" -gt 0 ]; then
      {
        for f in "${snapshot_files[@]}"; do
          if content=$(git show "origin/memory/token-audit:$f" 2>/dev/null); then
            echo "$content"
          else
            echo "⚠ Failed to retrieve snapshot: $f" >&2
            echo 'null'
          fi
        done
      } | jq -s \
          --arg window_start "$window_start" \
          --arg generated_at "$generated_at" \
          --argjson snapshot_count "$aic_snapshot_count" '
        [.[].workflows[]? | {workflow_name, total_aic: (.total_aic // 0), run_count: (.run_count // 0)}]
        | sort_by(.workflow_name)
        | group_by(.workflow_name)
        | map({
            workflow_name: .[0].workflow_name,
            total_aic: (map(.total_aic) | add // 0),
            run_count: (map(.run_count) | add // 0)
          })
        | sort_by(-.total_aic)
        | {
            source: "token-audit-memory",
            window_start: $window_start,
            generated_at: $generated_at,
            snapshot_count: $snapshot_count,
            total_aic: (map(.total_aic) | add // 0),
            workflows: .
          }
      ' > "$DATA_DIR/aic-by-workflow.json"
    else
      printf '{"source":"token-audit-memory","window_start":"%s","snapshot_count":0,"total_aic":0,"workflows":[]}\n' \
        "$window_start" > "$DATA_DIR/aic-by-workflow.json"
    fi
  else
    printf '{"source":"none","window_start":"%s","snapshot_count":0,"total_aic":0,"workflows":[]}\n' \
      "$window_start" > "$DATA_DIR/aic-by-workflow.json"
    echo "⚠ Could not fetch memory/token-audit branch (does the branch exist? are credentials configured?); AIC by workflow data unavailable" >&2
  fi
fi

if has_data_file "$DATA_DIR/merged-prs.json"; then
  echo "Using cached merged PR dataset"
else
  gh pr list \
    --repo "$repo" \
    --state merged \
    --search "merged:>=$window_start" \
    --limit 500 \
    --json number,title,url,mergedAt,closedAt,body,labels \
    > "$DATA_DIR/merged-prs.json" || printf '%s\n' '[]' > "$DATA_DIR/merged-prs.json"
fi

if has_data_file "$DATA_DIR/closed-unmerged-prs.json"; then
  echo "Using cached closed-unmerged PR dataset"
else
  gh pr list \
    --repo "$repo" \
    --state closed \
    --search "closed:>=$window_start is:unmerged" \
    --limit 500 \
    --json number,title,url,mergedAt,closedAt,body,labels \
    > "$DATA_DIR/closed-unmerged-prs.json" || printf '%s\n' '[]' > "$DATA_DIR/closed-unmerged-prs.json"
fi

jq '
  def linked_issue_numbers:
    [scan("(?i)(?:close[sd]?|fix(?:e[sd])?|resolve[sd]?)\\s+#([0-9]+)")[]? | tonumber] | unique;
  map(. + {linked_issue_numbers: ((.body // "") | linked_issue_numbers)})
' "$DATA_DIR/merged-prs.json" > "$DATA_DIR/merged-prs-linked.json"

jq '
  def linked_issue_numbers:
    [scan("(?i)(?:close[sd]?|fix(?:e[sd])?|resolve[sd]?)\\s+#([0-9]+)")[]? | tonumber] | unique;
  map(. + {linked_issue_numbers: ((.body // "") | linked_issue_numbers)})
' "$DATA_DIR/closed-unmerged-prs.json" > "$DATA_DIR/closed-unmerged-prs-linked.json"

jq -n \
  --arg generated_at "$generated_at" \
  --arg repository "$repo" \
  --arg window_start "$window_start" \
  --arg workflow_logs_source "$logs_source" \
  --argjson aic_snapshot_count "$aic_snapshot_count" \
  --slurpfile workflow_logs "$DATA_DIR/workflow-logs.json" \
  --slurpfile merged "$DATA_DIR/merged-prs-linked.json" \
  --slurpfile closed "$DATA_DIR/closed-unmerged-prs-linked.json" \
  --slurpfile mapping "$DATA_DIR/objective-mapping.json" \
  --slurpfile aic_by_workflow "$DATA_DIR/aic-by-workflow.json" '
  {
    generated_at: $generated_at,
    repository: $repository,
    window_start: $window_start,
    workflow_logs_source: $workflow_logs_source,
    workflow_run_count: (($workflow_logs[0].runs // $workflow_logs[0].runs // []) | length),
    merged_pr_count: (($merged[0] // []) | length),
    merged_prs_with_linked_issue: (($merged[0] // []) | map(select((.linked_issue_numbers | length) > 0)) | length),
    closed_unmerged_pr_count: (($closed[0] // []) | length),
    closed_unmerged_prs_with_linked_issue: (($closed[0] // []) | map(select((.linked_issue_numbers | length) > 0)) | length),
    objective_mapping_present: ((($mapping[0] // {}) | type) == "object" and ((($mapping[0] // {}) | keys | length) > 0)),
    aic_by_workflow_source: ($aic_by_workflow[0].source // "none"),
    aic_by_workflow_snapshot_count: $aic_snapshot_count,
    aic_by_workflow_total: ($aic_by_workflow[0].total_aic // 0),
    safe_output_precompute_note: "Safe-output issue resolution may still require live lookups unless workflow log data already contains the needed identifiers.",
    required_live_fallbacks: [
      "safe-output issue state or label gaps not present in precomputed files",
      "root-issue label fetches for traced linked issues"
    ]
  }
' > "$DATA_DIR/dataset-manifest.json"

node scripts/prepare-objective-impact-safe-output-evaluations.cjs