#!/usr/bin/env bash

set -euo pipefail

export LC_ALL=C

REPOSITORY=github/gh-aw
WORKFLOW_NAME="Approach Validator"
MATURATION_SECONDS=2592000

tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/approach-validator-operational-value.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM

definition() {
    cat <<'JSON'
{
  "schemaVersion": 4,
  "grader": "operational-value",
  "repository": "github/gh-aw",
  "workflowName": "Approach Validator",
  "sourcePath": ".github/workflows/approach-validator.md",
  "adoption": {
    "commit": "9e7ad8ff23ed9d9ed2dfe23a9818a46292a08f44",
    "adoptedAt": "2026-04-08T18:09:46Z"
  },
  "operationalValue": "For an approach submitted for validation, obtain an explicit human approval or rejection of its validation report.",
  "evidence": {
    "opportunity": "The issue or pull request carrying the workflow's Approach Validation Report.",
    "assignment": "Use the report comment containing both the run ID URL and the report heading; key is item:<issue-or-pr-number>. Repeated runs retain that key.",
    "accepted": "A non-bot +1 or -1 reaction on that report comment created no later than the evidence cutoff.",
    "repositories": ["github/gh-aw"],
    "collection": "With issues:read, locate the report comment through the repository issue-comments API and collect its reaction records through the comment-reactions API.",
    "maturation": "Thirty days after run creation, matching the report artifact retention period.",
    "zeroRule": "A located report comment with no qualifying human approval or rejection reaction by the cutoff scores 0.",
    "missingRule": "Unavailable, deleted, ambiguous, or malformed report-comment or reaction evidence scores null."
  },
  "primaryMetric": {
    "id": "human-approach-decision",
    "formula": "1 when the assigned report has at least one qualifying +1 or -1 reaction by the cutoff; 0 when its evidence is available and no such reaction exists.",
    "direction": "higher_is_better"
  },
  "baseline": {
    "mode": "attainment-only",
    "value": null,
    "evidenceCutoff": null,
    "provenance": []
  },
  "validationExamples": {
    "targetAttained": {"valid": true, "decision": true},
    "targetMissed": {"valid": true, "decision": false},
    "missing": {"valid": false},
    "malformed": {"valid": "true", "decision": true}
  }
}
JSON
}

metric() {
    jq '
      if .valid != true or (.decision | type) != "boolean" then null
      elif .decision then 1
      else 0
      end'
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

emit_null() {
    local opportunity_key=$1 case_json=$2 evidence_cutoff=$3 matures_at=$4 reason=$5
    jq -cn \
        --arg opportunityKey "$opportunity_key" \
        --argjson case "$case_json" \
        --arg evidenceCutoff "$evidence_cutoff" \
        --arg maturesAt "$matures_at" \
        --arg reason "$reason" \
        '{value: null, opportunityKey: $opportunityKey, case: $case,
          evidenceCutoff: $evidenceCutoff, maturesAt: $maturesAt,
          provenance: [], diagnostics: {missingReason: $reason}}'
}

find_case() {
    local repository=$1 run_id=$2 cutoff=$3 comments
    comments=$(gh api --paginate "repos/$repository/issues/comments?per_page=100" 2>"$tmp_dir/gh-error") || return 1
    printf '%s\n' "$comments" | jq -sc \
        --arg runID "$run_id" \
        --arg cutoff "$cutoff" '
        [ .[][]
          | select((.body | type == "string")
                   and (.body | contains("Approach Validation Report"))
                   and (.body | contains("/actions/runs/" + $runID))
                   and (.created_at | type == "string" and . <= $cutoff))
          | {commentId: .id, itemNumber: (.issue_url | capture("/issues/(?<number>[0-9]+)$").number | tonumber)}
        ] | unique_by(.commentId)
          | if length == 1 then .[0] else empty end'
}

has_human_decision() {
    local repository=$1 comment_id=$2 cutoff=$3 reactions status
    reactions=$(gh api --paginate "repos/$repository/issues/comments/$comment_id/reactions?per_page=100" \
        2>"$tmp_dir/gh-error") || return 2
    if printf '%s\n' "$reactions" | jq -esc --arg cutoff "$cutoff" '
      any(.[][];
        (.content == "+1" or .content == "-1")
        and (.user.type == "User")
        and (.created_at | type == "string" and . <= $cutoff)
      )' >/dev/null; then
        return 0
    else
        status=$?
    fi
    [[ $status -eq 1 ]] && return 1
    return 2
}

grade_run() {
    local request run_id repository workflow created_at evidence_at matures_at evidence_cutoff
    local case_json comment_id item_number opportunity_key decision value

    request=$(cat)
    if ! printf '%s\n' "$request" | jq -e '
      .schemaVersion == 1
      and (.run.id | type == "string" and length > 0)
      and (.run.repository | type == "string")
      and (.run.workflow | type == "string")
      and (.run.createdAt | type == "string")
      and (.evidenceAt | type == "string")
      and (.case == null or (.case | type == "object"))
    ' >/dev/null 2>&1; then
        emit_null "invalid-request" '{"invalidRequest":true}' "1970-01-01T00:00:00Z" "1970-01-01T00:00:00Z" "invalid request"
        return
    fi

    run_id=$(printf '%s\n' "$request" | jq -r '.run.id')
    repository=$(printf '%s\n' "$request" | jq -r '.run.repository')
    workflow=$(printf '%s\n' "$request" | jq -r '.run.workflow')
    created_at=$(printf '%s\n' "$request" | jq -r '.run.createdAt')
    evidence_at=$(printf '%s\n' "$request" | jq -r '.evidenceAt')
    if ! created_at=$(normalize_timestamp "$created_at") || ! evidence_at=$(normalize_timestamp "$evidence_at"); then
        emit_null "invalid-timestamp" '{"invalidTimestamp":true}' "1970-01-01T00:00:00Z" "1970-01-01T00:00:00Z" "invalid timestamp"
        return
    fi
    matures_at=$(add_seconds "$created_at" "$MATURATION_SECONDS")
    if [[ $evidence_at < $matures_at ]]; then evidence_cutoff=$evidence_at; else evidence_cutoff=$matures_at; fi

    if [[ $repository != "$REPOSITORY" || $workflow != "$WORKFLOW_NAME" ]]; then
        emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" "run does not match frozen contract"
        return
    fi

    case_json=$(printf '%s\n' "$request" | jq -c '.case')
    if [[ $case_json == null ]]; then
        if ! case_json=$(find_case "$repository" "$run_id" "$evidence_cutoff") || [[ -z $case_json ]]; then
            emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" "report-comment-assignment-unavailable"
            return
        fi
    elif ! printf '%s\n' "$case_json" | jq -e '
      (.commentId | type == "number" and . > 0 and floor == .)
      and (.itemNumber | type == "number" and . > 0 and floor == .)
    ' >/dev/null; then
        emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" "invalid case"
        return
    fi

    comment_id=$(printf '%s\n' "$case_json" | jq -r '.commentId')
    item_number=$(printf '%s\n' "$case_json" | jq -r '.itemNumber')
    opportunity_key="item:$item_number"
    if has_human_decision "$repository" "$comment_id" "$evidence_cutoff"; then
        decision=true
    else
        local reaction_status=$?
        if [[ $reaction_status -eq 1 ]]; then
            decision=false
        else
        emit_null "$opportunity_key" "$case_json" "$evidence_cutoff" "$matures_at" "reaction-evidence-unavailable"
        return
        fi
    fi
    value=$(jq -cn --argjson decision "$decision" '{valid: true, decision: $decision}' | metric)
    jq -cn \
      --argjson value "$value" --arg opportunityKey "$opportunity_key" --argjson case "$case_json" \
      --arg evidenceCutoff "$evidence_cutoff" --arg maturesAt "$matures_at" \
      --arg repository "$repository" --arg commentID "$comment_id" \
      '{value: $value, opportunityKey: $opportunityKey, case: $case,
        evidenceCutoff: $evidenceCutoff, maturesAt: $maturesAt,
        provenance: [{repository: $repository, kind: "issue-comment", ref: $commentID},
                     {repository: $repository, kind: "comment-reactions", ref: $commentID}],
        diagnostics: {reportCommentLocated: true}}'
}

case ${1:-} in
    --definition) definition ;;
    --metric) metric ;;
    --grade-run) grade_run ;;
    *) printf 'usage: %s --definition|--metric|--grade-run\n' "$0" >&2; exit 1 ;;
esac
