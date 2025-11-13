#!/usr/bin/env bash
set -euo pipefail

# Bug Bash Preflight Test Script
# Purpose: Exercise the logic the workflow will use to classify and emit update-project safe outputs
#          before running the full agentic workflow. This avoids noisy failures and validates token scope.
#
# Requirements:
#   - Environment variable PROJECT_GITHUB_TOKEN (preferred) or fallback GITHUB_TOKEN
#   - jq installed
#   - curl available
#
# Usage:
#   export PROJECT_GITHUB_TOKEN=ghp_yourPAT   # PAT with repo + project scopes
#   ./scripts/bug-bash-preflight.sh mnkiefer repo-with-bugs 15
#
# Arguments:
#   $1 = owner (e.g., mnkiefer)
#   $2 = repo  (e.g., repo-with-bugs)
#   $3 = max items (optional, default 15)
#
# Output:
#   - Summary of token scopes (best-effort via response headers)
#   - Sample open issues with basic metadata
#   - Classification preview table
#   - Generated safe-output JSON lines (NOT written) and optional file append if GH_AW_SAFE_OUTPUTS set

OWNER=${1:-}
REPO=${2:-}
MAX=${3:-15}

if [[ -z "$OWNER" || -z "$REPO" ]]; then
  echo "Usage: $0 <owner> <repo> [max]" >&2
  exit 1
fi

TOKEN="${PROJECT_GITHUB_TOKEN:-${GITHUB_TOKEN:-}}"
if [[ -z "$TOKEN" ]]; then
  echo "ERROR: PROJECT_GITHUB_TOKEN or GITHUB_TOKEN not set" >&2
  exit 1
fi

echo "== Preflight: Repository $OWNER/$REPO (max=$MAX) ==" >&2

viewer_resp=$(curl -s -H "Authorization: Bearer $TOKEN" https://api.github.com/graphql \
  -d '{"query":"query{viewer{login}}"}') || true
viewer_login=$(echo "$viewer_resp" | jq -r '.data.viewer.login // empty' 2>/dev/null) || true
if [[ -z "$viewer_login" ]]; then
  errors=$(echo "$viewer_resp" | jq -r '.errors[]?.message' 2>/dev/null || true)
  echo "Viewer login: UNKNOWN" >&2
  if [[ -n "$errors" ]]; then
    echo "GraphQL errors:" >&2
    echo "$errors" >&2
  fi
  echo "Suggestion: Ensure PAT has 'repo' and classic project scopes (or project v2 access). Regenerate if expired." >&2
else
  echo "Viewer login: $viewer_login" >&2
fi

# Token scope diagnostics (classic PAT scopes appear in x-oauth-scopes header)
scope_headers=$(curl -s -D - -o /dev/null -H "Authorization: token $TOKEN" https://api.github.com/user || true)
token_scopes=$(echo "$scope_headers" | grep -i '^x-oauth-scopes:' | cut -d':' -f2- | sed 's/^ *//')
if [[ -n "$token_scopes" ]]; then
  echo "Token scopes: $token_scopes" >&2
else
  echo "Token scopes: (not reported - token may be fine-grained or missing scopes)" >&2
fi

# Repo existence (GraphQL) - build payload with jq to avoid quoting issues
repo_payload=$(jq -n --arg name "$REPO" --arg owner "$OWNER" '{query:"query($name:String!,$owner:String!){repository(name:$name,owner:$owner){id name isPrivate}}",variables:{name:$name,owner:$owner}}')
repo_check=$(curl -s -H "Authorization: Bearer $TOKEN" https://api.github.com/graphql -d "$repo_payload" || true)
repo_id=$(echo "$repo_check" | jq -r '.data.repository.id // empty' 2>/dev/null) || true
if [[ -z "$repo_id" ]]; then
  echo "ERROR: Repository not accessible with current token." >&2
  # Surface GraphQL errors (if any)
  echo "$repo_check" | jq '.errors // empty' 2>/dev/null >&2 || true
  exit 2
fi
echo "Repo ID: $repo_id" >&2

# Calculate current project name based on date (YYYY - WNN format with spaces)
PROJECT_NAME="Bug Bash $(date +%Y) - W$(date +%V)"
echo "Project name: $PROJECT_NAME" >&2

echo "Fetching open issues (first 100, may truncate)" >&2
issues_json=$(curl -s -H "Authorization: token $TOKEN" \
  "https://api.github.com/repos/$OWNER/$REPO/issues?state=open&per_page=100")

if echo "$issues_json" | jq -e 'type == "array"' >/dev/null 2>&1; then
  total=$(echo "$issues_json" | jq 'length')
else
  echo "ERROR: Issues response not an array." >&2
  echo "$issues_json" | head -c 500 >&2
  exit 3
fi
echo "Total open issues fetched: $total" >&2

classify_issue() {
  local issue_json="$1"
  local num title body len labels comments reactions priority complexity impact classification
  num=$(echo "$issue_json" | jq -r '.number')
  title=$(echo "$issue_json" | jq -r '.title')
  body=$(echo "$issue_json" | jq -r '.body // ""')
  len=${#body}
  labels=$(echo "$issue_json" | jq -r '[.labels[].name] | join(",")')
  comments=$(echo "$issue_json" | jq -r '.comments')
  # reactions requires separate endpoint for full summary; try partial
  reactions_url=$(echo "$issue_json" | jq -r '.reactions.url // empty')
  reactions=0
  if [[ -n "$reactions_url" ]]; then
    reactions=$(curl -s -H "Authorization: token $TOKEN" "$reactions_url" | jq '[.[] | .content] | length' 2>/dev/null || echo 0)
  fi

  # Priority
  if [[ "$labels" =~ (^|,)P0(,|$) || "$labels" =~ (^|,)P1(,|$) || "$labels" =~ (^|,)severity:critical(,|$) ]]; then
    priority="Critical"
  elif [[ $((comments + reactions)) -ge 5 || "$labels" =~ severity:high ]]; then
    priority="High"
  else
    priority="Medium"
  fi

  # Complexity
  if [[ "$labels" =~ architecture || "$labels" =~ security ]]; then
    complexity="Complex"
  elif [[ $len -lt 600 ]]; then
    complexity="Quick Win"
  else
    complexity="Standard"
  fi

  # Impact
  if [[ "$labels" =~ blocker ]]; then
    impact="Blocker"
  else
    # Count component/area labels
    comp_count=$(echo "$labels" | tr ',' '\n' | grep -E '^(area:|component:|module:)' | wc -l | tr -d ' ')
    if [[ $comp_count -ge 2 ]]; then
      impact="Major"
    else
      impact="Minor"
    fi
  fi

  classification="${priority}|${impact}|${complexity}"
  printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\n' "$num" "$priority" "$impact" "$complexity" "$classification" "$comments" "$reactions"
}

echo -e "Number\tPriority\tImpact\tComplexity\tClassification\tComments\tReactions" >&2
safe_outputs=()
count_added=0

while read -r issue; do
  # Filter labels: must have bug/defect/regression
  label_str=$(echo "$issue" | jq -r '[.labels[].name] | join(",")')
  if ! echo "$label_str" | grep -qiE '(^|,)(bug|defect|regression)(,|$)'; then
    continue
  fi
  # Skip in-progress/wip
  if echo "$label_str" | grep -qiE '(^|,)(in-progress|wip|blocked-by-external)(,|$)'; then
    continue
  fi
  line=$(classify_issue "$issue")
  echo "$line" >&2
  if [[ $count_added -lt $MAX ]]; then
    issue_number=$(echo "$issue" | jq -r '.number')
    safe_json=$(jq -n \
      --arg num "$issue_number" \
      --arg prio "$(echo "$line" | cut -f2)" \
      --arg compx "$(echo "$line" | cut -f4)" \
      --arg impact "$(echo "$line" | cut -f3)" \
      --arg project "$PROJECT_NAME" \
      '{type:"update-project", project:$project, content_type:"issue", content_number:($num|tonumber), fields:{Status:"To Do", Priority:$prio, Complexity:$compx, Impact:$impact}}')
    echo "$safe_json" >&2
    safe_outputs+=("$safe_json")
    ((count_added++))
  fi
done < <(echo "$issues_json" | jq -c '.[]')

echo "Added (simulated): $count_added" >&2

if [[ -n "${GH_AW_SAFE_OUTPUTS:-}" && $count_added -gt 0 ]]; then
  echo "Appending to $GH_AW_SAFE_OUTPUTS" >&2
  for s in "${safe_outputs[@]}"; do
    echo "$s" >>"$GH_AW_SAFE_OUTPUTS"
  done
fi

echo "Preflight complete." >&2
