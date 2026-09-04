#!/usr/bin/env bash

set -euo pipefail

export LC_ALL=C

REPOSITORY=github/gh-aw
WORKFLOW_NAME=Archie
MATURATION_SECONDS=604800

tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/archie-operational-value.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM

definition() {
    cat <<'JSON'
{
  "schemaVersion": 4,
  "grader": "operational-value",
  "repository": "github/gh-aw",
  "workflowName": "Archie",
  "sourcePath": ".github/workflows/archie.md",
  "adoption": {"commit": "c5e2550db37fb8c8b99af299e0fa362216334fe8", "adoptedAt": "2025-11-04T19:45:30Z"},
  "operationalValue": "For the assigned issue or pull request, post a GitHub-renderable Mermaid diagram in its discussion.",
  "evidence": {
    "opportunity": "The issue or pull request that triggered the Archie workflow run.",
    "assignment": "Use the recorded issue or pull-request case; when absent, recover the subject from an Archie footer that contains the immutable run URL. Duplicate runs retain the same issue:<number> or pull-request:<number> key.",
    "accepted": "An issue comment authored by github-actions[bot] on the assigned subject that contains both a Mermaid fenced block and Archie’s footer linking to this exact workflow run.",
    "repositories": ["github/gh-aw"],
    "collection": "With issues:read, pull-requests:read, and actions:read, read subject comments at the capped cutoff; search issue comments by the exact run URL only to reconstruct a missing historical assignment.",
    "maturation": "Seven days after run creation, providing a stable observation window for delayed safe-output publication.",
    "zeroRule": "A known assigned subject with no accepted diagram comment by the cutoff scores 0.",
    "missingRule": "Unavailable comment search or retrieval, or a run whose subject cannot be reconstructed, scores null."
  },
  "primaryMetric": {
    "id": "diagram-posted-to-assigned-subject",
    "formula": "1 when accepted evidence contains a Mermaid fenced diagram comment for the assigned subject and exact run URL; 0 when the assigned subject is available but no such comment exists.",
    "direction": "higher_is_better"
  },
  "baseline": {"mode": "attainment-only", "value": null, "evidenceCutoff": null, "provenance": []},
  "validationExamples": {
    "targetAttained": {"valid": true, "diagramPosted": true},
    "targetMissed": {"valid": true, "diagramPosted": false},
    "missing": {"valid": false},
    "malformed": {"valid": "true", "diagramPosted": true}
  }
}
JSON
}

metric() {
    jq 'if .valid != true or (.diagramPosted | type) != "boolean" then null elif .diagramPosted then 1 else 0 end'
}

normalize_timestamp() {
    jq -nr --arg timestamp "$1" '
        ($timestamp | sub("\\.[0-9]+Z$"; "Z")) as $normalized
        | if ($normalized | test("^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$"))
            and (try (($normalized | fromdateiso8601 | todateiso8601) == $normalized) catch false)
          then $normalized
          else error("invalid timestamp")
          end
    ' 2>/dev/null
}

add_seconds() {
    jq -nr --arg timestamp "$1" --argjson seconds "$2" \
        '$timestamp | fromdateiso8601 + $seconds | todateiso8601'
}

github_api() {
    gh api "$@" 2>"$tmp_dir/gh-api-error"
}

emit_null() {
    jq -cn --arg opportunityKey "$1" --argjson case "$2" --arg evidenceCutoff "$3" \
        --arg maturesAt "$4" --arg reason "$5" \
        '{value: null, opportunityKey: $opportunityKey, case: $case,
          evidenceCutoff: $evidenceCutoff, maturesAt: $maturesAt,
          provenance: [], diagnostics: {missingReason: $reason}}'
}

case_from_event() {
    printf '%s\n' "$1" | jq -c '
        if (.issue.number? | type) == "number" then {kind: "issue", number: .issue.number}
        elif (.pull_request.number? | type) == "number" then {kind: "pull-request", number: .pull_request.number}
        else empty end'
}

valid_case() {
    printf '%s\n' "$1" | jq -e '
        type == "object" and (.kind == "issue" or .kind == "pull-request")
        and (.number | type == "number" and floor == . and . > 0)' >/dev/null
}

run_marker() {
    printf 'https://github.com/%s/actions/runs/%s' "$1" "$2"
}

find_case_by_marker() {
    local repository=$1 marker=$2 cutoff=$3 results candidates candidate number
    results=$(github_api -X GET search/issues -f q="repo:$repository in:comments \"$marker\"" -f per_page=100) || return 1
    candidates=$(printf '%s\n' "$results" | jq -cer '
        [.items[] | select((.pull_request? | type) == "object") | {kind: "pull-request", number}]
        + [.items[] | select((.pull_request? | type) != "object") | {kind: "issue", number}]
        | unique') || return 1
    while IFS= read -r candidate; do
        number=$(printf '%s\n' "$candidate" | jq -r '.number')
        if accepted_comment_exists "$repository" "$number" "$marker" "$cutoff"; then
            printf '%s\n' "$candidate"
            return
        fi
    done <<EOF
$(printf '%s\n' "$candidates" | jq -c '.[]')
EOF
    return 1
}

accepted_comment_exists() {
    local repository=$1 number=$2 marker=$3 cutoff=$4 comments
    comments=$(github_api "repos/$repository/issues/$number/comments?per_page=100") || return 1
    printf '%s\n' "$comments" | jq -e --arg marker "$marker" --arg cutoff "$cutoff" '
        any(.[]; .user.login == "github-actions[bot]"
          and (.created_at | type == "string" and . <= $cutoff)
          and (.body | type == "string" and contains($marker) and test("```mermaid[[:space:]]")))'
}

grade_run() {
    local request run_id repository workflow created_at evidence_at matures_at evidence_cutoff
    local case_json event_json marker number evidence value

    request=$(cat)
    if ! printf '%s\n' "$request" | jq -e '
        .schemaVersion == 1 and (.run.id | type == "string" and length > 0)
        and (.run.repository | type == "string") and (.run.workflow | type == "string")
        and (.run.createdAt | type == "string") and (.evidenceAt | type == "string")
        and (.case == null or (.case | type == "object"))
        and (.event == null or (.event | type == "object"))' >/dev/null 2>&1; then
        emit_null "invalid-request" '{"invalidRequest":true}' "1970-01-01T00:00:00Z" "1970-01-01T00:00:00Z" "invalid request"
        return
    fi

    run_id=$(printf '%s\n' "$request" | jq -r '.run.id')
    repository=$(printf '%s\n' "$request" | jq -r '.run.repository')
    workflow=$(printf '%s\n' "$request" | jq -r '.run.workflow')
    created_at=$(printf '%s\n' "$request" | jq -r '.run.createdAt')
    evidence_at=$(printf '%s\n' "$request" | jq -r '.evidenceAt')
    if ! created_at=$(normalize_timestamp "$created_at") || ! evidence_at=$(normalize_timestamp "$evidence_at"); then
        emit_null "run:$run_id" '{"invalidTimestamp":true}' "1970-01-01T00:00:00Z" "1970-01-01T00:00:00Z" "invalid timestamp"
        return
    fi
    matures_at=$(add_seconds "$created_at" "$MATURATION_SECONDS")
    evidence_cutoff=$(jq -nr --arg evidenceAt "$evidence_at" --arg maturesAt "$matures_at" '
        if ($evidenceAt | fromdateiso8601) < ($maturesAt | fromdateiso8601) then $evidenceAt else $maturesAt end')

    if [[ $repository != "$REPOSITORY" || $workflow != "$WORKFLOW_NAME" ]]; then
        emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" "run repository or workflow does not match the frozen contract"
        return
    fi

    case_json=$(printf '%s\n' "$request" | jq -c '.case')
    if [[ $case_json == null ]]; then
        event_json=$(printf '%s\n' "$request" | jq -c '.event')
        case_json=$(case_from_event "$event_json" || true)
    fi
    marker=$(run_marker "$repository" "$run_id")
    if [[ -z ${case_json:-} ]]; then
        if ! case_json=$(find_case_by_marker "$repository" "$marker" "$evidence_cutoff"); then
            emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" "assignment-unavailable"
            return
        fi
    elif ! valid_case "$case_json"; then
        emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" "invalid-case"
        return
    fi

    number=$(printf '%s\n' "$case_json" | jq -r '.number')
    if ! accepted_comment_exists "$repository" "$number" "$marker" "$evidence_cutoff"; then
        if [[ -s $tmp_dir/gh-api-error ]]; then
            emit_null "$(printf '%s\n' "$case_json" | jq -r '.kind + ":" + (.number | tostring)')" "$case_json" "$evidence_cutoff" "$matures_at" "comments-unavailable"
            return
        fi
        evidence='{"valid":true,"diagramPosted":false}'
    else
        evidence='{"valid":true,"diagramPosted":true}'
    fi
    value=$(printf '%s\n' "$evidence" | metric)
    jq -cn --argjson value "$value" --argjson case "$case_json" --arg evidenceCutoff "$evidence_cutoff" \
        --arg maturesAt "$matures_at" --arg repository "$repository" --arg marker "$marker" '
        {value: $value, opportunityKey: ($case.kind + ":" + ($case.number | tostring)), case: $case,
         evidenceCutoff: $evidenceCutoff, maturesAt: $maturesAt,
         provenance: [{repository: $repository, kind: "issue-comment-search", ref: $marker}],
         diagnostics: {}}'
}

case ${1:-} in
    --definition) definition ;;
    --metric) metric ;;
    --grade-run) grade_run ;;
    *) printf 'usage: %s --definition|--metric|--grade-run\n' "$0" >&2; exit 1 ;;
esac
