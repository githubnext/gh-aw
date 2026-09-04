#!/usr/bin/env bash

set -euo pipefail

REPOSITORY=github/gh-aw
WORKFLOW_NAME="Daily Assign Issue To User"
MATURATION_SECONDS=86400
ASSIGNMENT_ACTOR="github-actions[bot]"

tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/daily-assign-issue-operational-value.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM

definition() {
    cat <<'JSON'
{
  "schemaVersion": 4,
  "grader": "operational-value",
  "repository": "github/gh-aw",
  "workflowName": "Daily Assign Issue To User",
  "sourcePath": ".github/workflows/daily-assign-issue-to-user.md",
  "adoption": {"commit": "4e2c72975191a91f5e9cb8ac4a06cc12d3a03575", "adoptedAt": "2025-12-04T02:26:50Z"},
  "operationalValue": "Retain a named contributor's ownership of each issue assigned by the run through its one-day observation horizon.",
  "evidence": {
    "opportunity": "Each issue assigned by github-actions[bot] during the run's one-day assignment window.",
    "assignment": "Issue assignment events by github-actions[bot] from run creation through maturity; key is assignment-batch:<run ID>, so reruns retain the same key.",
    "accepted": "Repository issue-event history establishes the assigned login and whether it remained assigned at the capped cutoff. Workflow traces, comments, and agent judgment are excluded.",
    "repositories": ["github/gh-aw"],
    "collection": "With issues:read, search issues updated on the assignment dates and retrieve every candidate's issue events. More than 1,000 search results or event pages makes evidence unavailable.",
    "maturation": "One day after run creation, matching the daily cadence and the explicit assignment action.",
    "zeroRule": "A reconstructed assignment batch with no assigned contributor retained at cutoff scores 0.",
    "missingRule": "Missing, incomplete, ambiguous, or unavailable issue-event evidence, including no reconstructable assignment batch, scores null."
  },
  "primaryMetric": {
    "id": "retained-assignment-share",
    "formula": "retainedAssignments / assignedIssues when a nonempty complete assignment batch is available; an issue is retained only when its github-actions[bot] assignee remains assigned at cutoff.",
    "direction": "higher_is_better"
  },
  "diagnosticMetrics": [
    {"id": "assignment-batch-capacity", "name": "Assignment batch capacity", "formula": "min(1, assignedIssues / 5) for a complete reconstructed batch.", "direction": "higher_is_better", "aggregation": "latest"}
  ],
  "baseline": {"mode": "attainment-only", "value": null, "evidenceCutoff": null, "provenance": []},
  "validationExamples": {
    "targetAttained": {"valid": true, "assignedIssues": 2, "retainedAssignments": 2},
    "targetMissed": {"valid": true, "assignedIssues": 2, "retainedAssignments": 0},
    "missing": {"valid": false},
    "malformed": {"valid": true, "assignedIssues": "2", "retainedAssignments": 1}
  }
}
JSON
}

metric() {
    jq '
      .assignedIssues as $assigned | .retainedAssignments as $retained
      | if .valid != true
        or ([$assigned, $retained] | all(.[]; type == "number" and floor == .) | not)
        or $assigned <= 0
        or $retained < 0
        or $retained > $assigned
      then null
      else $retained / $assigned
      end'
}

normalize_timestamp() {
    jq -nr --arg timestamp "$1" '
      ($timestamp | sub("\\.[0-9]+Z$"; "Z")) as $normalized
      | if ($normalized | test("^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$"))
          and (try (($normalized | fromdateiso8601 | todateiso8601) == $normalized) catch false)
        then $normalized else error("invalid timestamp") end' 2>/dev/null
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

issue_events() {
    local repository=$1 issue=$2 page=1 response events='[]'

    while :; do
        response=$(github_api "repos/$repository/issues/$issue/events?per_page=100&page=$page") || return 1
        jq -e 'type == "array"' >/dev/null <<<"$response" || return 1
        events=$(jq -cn --argjson prior "$events" --argjson next "$response" '$prior + $next')
        if (( $(jq 'length' <<<"$response") < 100 )); then
            printf '%s\n' "$events"
            return
        fi
        (( page < 10 )) || return 1
        page=$((page + 1))
    done
}

reconstruct_case() {
    local repository=$1 created_at=$2 matures_at=$3 start_day end_day response
    local page=1 total_count candidates='[]' issue events assignments='[]'

    start_day=${created_at%%T*}
    end_day=${matures_at%%T*}
    while :; do
        response=$(github_api -X GET search/issues \
            -f "q=repo:$repository is:issue updated:$start_day..$end_day" \
            -f per_page=100 -f page="$page") || return 1
        jq -e '.total_count | type == "number"' >/dev/null <<<"$response" || return 1
        if (( page == 1 )); then total_count=$(jq -r '.total_count' <<<"$response"); fi
        (( total_count <= 1000 )) || return 1
        candidates=$(jq -cn --argjson prior "$candidates" --argjson next "$response" \
            '$prior + [$next.items[] | select(.pull_request | not) | .number]')
        if (( $(jq '.items | length' <<<"$response") < 100 )); then break; fi
        (( page < 10 )) || return 1
        page=$((page + 1))
    done

    while IFS= read -r issue; do
        events=$(issue_events "$repository" "$issue") || return 1
        assignments=$(jq -cn --argjson prior "$assignments" --argjson events "$events" \
            --arg actor "$ASSIGNMENT_ACTOR" --arg start "$created_at" --arg end "$matures_at" \
            --argjson issue "$issue" '
              $prior + [$events[]
                | select(.event == "assigned" and .actor.login == $actor
                    and (.created_at >= $start and .created_at <= $end)
                    and (.assignee.login | type == "string"))
                | {number: $issue, assignee: .assignee.login, assignedAt: .created_at}]') 
    done < <(jq -r '.[]' <<<"$candidates")

    jq -cn --argjson assignments "$assignments" '
      {issues: ($assignments | unique_by(.number, .assignee, .assignedAt) | sort_by(.assignedAt, .number, .assignee))}'
}

case_is_valid() {
    jq -e '
      .issues | type == "array" and length > 0
      and all(.[];
        (.number | type == "number" and floor == . and . > 0)
        and (.assignee | type == "string" and length > 0)
        and (.assignedAt | type == "string"))' >/dev/null
}

retained_assignments() {
    local repository=$1 case_json=$2 cutoff=$3 issue events retained=0

    while IFS= read -r issue; do
        events=$(issue_events "$repository" "$(jq -r '.number' <<<"$issue")") || return 1
        if jq -e --arg cutoff "$cutoff" --argjson assignment "$issue" '
              [ .[]
                | select((.created_at | type == "string") and .created_at <= $cutoff)
                | select(.assignee.login == $assignment.assignee)
                | select(.event == "assigned" or .event == "unassigned")
                | {event, created_at} ]
              | sort_by(.created_at)
              | last?.event == "assigned"' <<<"$events" >/dev/null; then
            retained=$((retained + 1))
        fi
    done < <(jq -c '.issues[]' <<<"$case_json")
    printf '%s\n' "$retained"
}

grade_run() {
    local request repository workflow run_id created_at evidence_at matures_at evidence_cutoff
    local case_json assigned retained evidence value diagnostics

    request=$(cat)
    if ! jq -e '
      .schemaVersion == 1 and (.run.id | type == "string" and length > 0)
      and (.run.repository | type == "string") and (.run.workflow | type == "string")
      and (.run.createdAt | type == "string") and (.evidenceAt | type == "string")
      and (.case == null or (.case | type == "object"))' >/dev/null <<<"$request"; then
        emit_null invalid-request '{"invalidRequest":true}' 1970-01-01T00:00:00Z 1970-01-01T00:00:00Z "invalid request"
        return
    fi
    repository=$(jq -r '.run.repository' <<<"$request")
    workflow=$(jq -r '.run.workflow' <<<"$request")
    run_id=$(jq -r '.run.id' <<<"$request")
    created_at=$(normalize_timestamp "$(jq -r '.run.createdAt' <<<"$request")") \
        || { emit_null "run:$run_id" '{"invalidTimestamp":true}' 1970-01-01T00:00:00Z 1970-01-01T00:00:00Z "invalid timestamp"; return; }
    evidence_at=$(normalize_timestamp "$(jq -r '.evidenceAt' <<<"$request")") \
        || { emit_null "run:$run_id" '{"invalidTimestamp":true}' 1970-01-01T00:00:00Z 1970-01-01T00:00:00Z "invalid timestamp"; return; }
    matures_at=$(add_seconds "$created_at" "$MATURATION_SECONDS")
    if [[ $evidence_at < $matures_at ]]; then evidence_cutoff=$evidence_at; else evidence_cutoff=$matures_at; fi
    if [[ $repository != "$REPOSITORY" || $workflow != "$WORKFLOW_NAME" ]]; then
        emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" "run does not match contract"
        return
    fi
    case_json=$(jq -c '.case' <<<"$request")
    if [[ $case_json == null ]]; then
        case_json=$(reconstruct_case "$repository" "$created_at" "$evidence_cutoff") \
            || { emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" "assignment-unavailable"; return; }
    fi
    if ! case_is_valid <<<"$case_json"; then
        emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" "assignment-unavailable"
        return
    fi
    assigned=$(jq '.issues | length' <<<"$case_json")
    retained=$(retained_assignments "$repository" "$case_json" "$evidence_cutoff") \
        || { emit_null "assignment-batch:$run_id" "$case_json" "$evidence_cutoff" "$matures_at" "issue-events-unavailable"; return; }
    evidence=$(jq -cn --argjson assignedIssues "$assigned" --argjson retainedAssignments "$retained" \
        '{valid:true, assignedIssues:$assignedIssues, retainedAssignments:$retainedAssignments}')
    value=$(metric <<<"$evidence")
    diagnostics=$(jq -cn --argjson assigned "$assigned" \
        '{"assignment-batch-capacity": ([1, ($assigned / 5)] | min)}')
    jq -cn --argjson value "$value" --arg run_id "$run_id" --argjson case "$case_json" \
        --arg evidenceCutoff "$evidence_cutoff" --arg maturesAt "$matures_at" --arg repository "$repository" \
        --argjson diagnostics "$diagnostics" \
        '{value:$value, opportunityKey:("assignment-batch:" + $run_id), case:$case,
          evidenceCutoff:$evidenceCutoff, maturesAt:$maturesAt,
          provenance:[$case.issues[] | {repository:$repository, kind:"issue-events", ref:(.number|tostring)}],
          diagnostics:$diagnostics}'
}

case ${1:-} in
    --definition) definition ;;
    --metric) metric ;;
    --grade-run) grade_run ;;
    *) printf 'usage: %s --definition|--metric|--grade-run\n' "$0" >&2; exit 1 ;;
esac
