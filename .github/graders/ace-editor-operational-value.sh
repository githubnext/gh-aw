#!/usr/bin/env bash

set -euo pipefail

export LC_ALL=C

REPOSITORY=github/gh-aw
WORKFLOW_NAME="ACE Editor Session"
MATURATION_SECONDS=86400
tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/ace-editor-operational-value.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM

definition() {
    cat <<'JSON'
{
  "schemaVersion": 4,
  "grader": "operational-value",
  "repository": "github/gh-aw",
  "workflowName": "ACE Editor Session",
  "sourcePath": ".github/workflows/ace-editor.md",
  "adoption": {
    "commit": "82239c030d6a1ef6ec8b87a80a1346eeef211f8d",
    "adoptedAt": "2026-09-03T23:02:13Z"
  },
  "operationalValue": "For an eligible /ace pull-request comment, attain a posted ACE session-link comment on that pull request.",
  "evidence": {
    "opportunity": "A pull request comment containing the /ace command.",
    "assignment": "Use the pull-request number from the archived case or event; when absent, select the latest /ace pull-request comment at or before run creation. Key: pull-request:<number>; duplicate runs retain the key.",
    "accepted": "A pull-request issue comment created after the run and no later than the capped evidence cutoff whose body contains the exact https://ace.com/session/github-gh-aw-pr<number> URL.",
    "repositories": ["github/gh-aw"],
    "collection": "Read pull-request comments with issues:read through the capped cutoff.",
    "maturation": "Twenty-four hours after run creation; evidenceCutoff is the earlier of evidenceAt and maturesAt.",
    "zeroRule": "A successfully assigned pull request with no matching ACE link comment at the cutoff scores zero.",
    "missingRule": "Unavailable or ambiguous assignment, invalid input, or unavailable comment evidence scores null."
  },
  "primaryMetric": {
    "id": "ace-session-link-posted",
    "formula": "1 when the accepted ACE link comment exists by the cutoff, otherwise 0",
    "direction": "higher_is_better"
  },
  "baseline": {
    "mode": "attainment-only",
    "value": null,
    "evidenceCutoff": null,
    "provenance": []
  },
  "validationExamples": {
    "targetAttained": {"valid": true, "commentFound": true},
    "targetMissed": {"valid": true, "commentFound": false},
    "missing": {"valid": false},
    "malformed": {"valid": "yes", "commentFound": "yes"}
  }
}
JSON
}

metric() {
    jq '
      if .valid != true or (.commentFound | type) != "boolean" then null
      elif .commentFound then 1 else 0 end
    '
}

normalize_timestamp() {
    jq -nr --arg timestamp "$1" '
      ($timestamp | sub("\\.[0-9]+Z$"; "Z")) as $normalized
      | if ($normalized | test("^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$"))
        and (try (($normalized | fromdateiso8601 | todateiso8601) == $normalized) catch false)
        then $normalized else error("invalid timestamp") end
    ' 2>/dev/null
}

add_seconds() {
    jq -nr --arg timestamp "$1" --argjson seconds "$2" \
        '$timestamp | fromdateiso8601 + $seconds | todateiso8601'
}

timestamp_epoch() {
    jq -nr --arg timestamp "$1" '$timestamp | fromdateiso8601'
}

github_api() {
    gh api "$@" 2>"$tmp_dir/gh-api-error"
}

comment_matches() {
    local comments=$1 pull_request=$2 cutoff=$3
    printf '%s\n' "$comments" | jq -e \
        --arg url "https://ace.com/session/github-gh-aw-pr${pull_request}" \
        --arg cutoff "$cutoff" '
        any(.[]?;
          (.created_at | type == "string" and . <= $cutoff)
          and (.body | type == "string" and contains($url))
        )' >/dev/null
}

find_case_from_request() {
    jq -r '
      [
        .case.pullRequest,
        .case.pull_request.number,
        .case.issue.number,
        .case.prNumber,
        .event.pullRequest,
        .event.pull_request.number,
        .event.issue.number,
        .event.payload.issue.number,
        .event.payload.pull_request.number
      ]
      | map(select(type == "number" and floor == .))
      | .[0] // empty
    '
}

reconstruct_case() {
    local repository=$1 created_at=$2
    local issues number comments candidate='' candidate_time='' selected_number='' selected_time=''
    if ! issues=$(github_api --paginate "repos/$repository/issues?state=all&per_page=100"); then
        return 1
    fi
    while IFS= read -r number; do
        [[ -n $number ]] || continue
        if ! comments=$(github_api --paginate "repos/$repository/issues/$number/comments?per_page=100"); then
            return 1
        fi
        while IFS=$'\t' read -r candidate_time candidate; do
            if [[ -n $candidate ]] && { [[ -z $selected_time ]] || $candidate_time > "$selected_time"; }; then
                selected_time=$candidate_time
                selected_number=$candidate
            fi
        done < <(printf '%s\n' "$comments" | jq -r --arg cutoff "$created_at" --arg number "$number" '
          .[] | select(.body | type == "string" and test("(^|[[:space:]])/ace([[:space:]]|$)"))
          | select(.created_at <= $cutoff) | [.created_at, $number] | @tsv')
    done < <(printf '%s\n' "$issues" | jq -r '.[] | select(.pull_request != null) | .number')
    [[ -n ${selected_number:-} ]] || return 1
    printf '%s\n' "$selected_number"
}

emit_null() {
    local key=$1 case_json=$2 cutoff=$3 matures=$4 reason=$5
    jq -cn --arg key "$key" --argjson case "$case_json" --arg cutoff "$cutoff" \
        --arg matures "$matures" --arg reason "$reason" \
        '{value:null,opportunityKey:$key,case:$case,evidenceCutoff:$cutoff,maturesAt:$matures,
          provenance:[],diagnostics:{missingReason:$reason}}'
}

grade_run() {
    local request run_id repository workflow created_at evidence_at matures_at cutoff
    local run_epoch matures_epoch pull_request case_json comments evidence value
    request=$(cat)
    if ! printf '%s\n' "$request" | jq -e '
      .schemaVersion == 1 and (.run.id | type == "string" and length > 0)
      and .run.repository == "github/gh-aw" and .run.workflow == "ACE Editor Session"
      and (.run.createdAt | type == "string") and (.evidenceAt | type == "string")
      and (.case == null or (.case | type == "object"))
    ' >/dev/null 2>&1; then
        printf '%s\n' '{"value":null,"opportunityKey":"invalid-request","case":{"invalidRequest":true},"evidenceCutoff":"1970-01-01T00:00:00Z","maturesAt":"1970-01-01T00:00:00Z","provenance":[]}'
        return
    fi
    run_id=$(printf '%s\n' "$request" | jq -r '.run.id')
    repository=$REPOSITORY
    workflow=$WORKFLOW_NAME
    created_at=$(printf '%s\n' "$request" | jq -r '.run.createdAt')
    evidence_at=$(printf '%s\n' "$request" | jq -r '.evidenceAt')
    if ! created_at=$(normalize_timestamp "$created_at") || ! evidence_at=$(normalize_timestamp "$evidence_at"); then
        printf '%s\n' '{"value":null,"opportunityKey":"invalid-timestamp","case":{"invalidTimestamp":true},"evidenceCutoff":"1970-01-01T00:00:00Z","maturesAt":"1970-01-01T00:00:00Z","provenance":[]}'
        return
    fi
    matures_at=$(add_seconds "$created_at" "$MATURATION_SECONDS")
    run_epoch=$(timestamp_epoch "$evidence_at")
    matures_epoch=$(timestamp_epoch "$matures_at")
    if (( run_epoch < matures_epoch )); then cutoff=$evidence_at; else cutoff=$matures_at; fi
    pull_request=$(printf '%s\n' "$request" | find_case_from_request)
    if [[ -z $pull_request ]]; then
        if ! pull_request=$(reconstruct_case "$repository" "$created_at"); then
            emit_null "run:$run_id" '{"assignmentMissing":true}' "$cutoff" "$matures_at" "assignment-unavailable"
            return
        fi
        case_json=$(jq -cn --argjson number "$pull_request" '{pullRequest:$number,source:"reconstructed"}')
    else
        case_json=$(jq -cn --argjson number "$pull_request" '{pullRequest:$number,source:"archived-case"}')
    fi
    if ! comments=$(github_api --paginate "repos/$repository/issues/$pull_request/comments?per_page=100"); then
        emit_null "pull-request:$pull_request" "$case_json" "$cutoff" "$matures_at" "comment-evidence-unavailable"
        return
    fi
    evidence=$(jq -cn --argjson valid true --argjson commentFound false '{valid:$valid,commentFound:$commentFound}')
    if comment_matches "$comments" "$pull_request" "$cutoff"; then
        evidence=$(jq -cn '{valid:true,commentFound:true}')
    fi
    value=$(printf '%s\n' "$evidence" | metric)
    jq -cn --argjson value "$value" --arg key "pull-request:$pull_request" \
        --argjson case "$case_json" --arg cutoff "$cutoff" --arg matures "$matures_at" \
        --arg repository "$repository" --arg pullRequest "$pull_request" --arg runId "$run_id" \
        '{value:$value,opportunityKey:$key,case:$case,evidenceCutoff:$cutoff,maturesAt:$matures,
          provenance:[{repository:$repository,kind:"issue-comment",ref:($pullRequest+"@"+$cutoff)}],
          diagnostics:{runId:$runId}}'
}

case ${1:-} in
    --definition) definition ;;
    --metric) metric ;;
    --grade-run) grade_run ;;
    *) printf 'usage: %s --definition|--metric|--grade-run\n' "$0" >&2; exit 1 ;;
esac
