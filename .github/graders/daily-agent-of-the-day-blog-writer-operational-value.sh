#!/usr/bin/env bash

set -euo pipefail

export LC_ALL=C

REPOSITORY=github/gh-aw
WORKFLOW_NAME="Daily Agent of the Day Blog Writer"
TRACKER_ID="daily-agent-of-the-day-blog-writer"
MATURATION_SECONDS=604800

tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/daily-agent-of-the-day-blog-writer-operational-value.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM

definition() {
    cat <<'JSON'
{
  "schemaVersion": 4, "grader": "operational-value",
  "repository": "github/gh-aw", "workflowName": "Daily Agent of the Day Blog Writer",
  "sourcePath": ".github/workflows/daily-agent-of-the-day-blog-writer.md",
  "adoption": {"commit": "15bc81bcf06ee72df0508c7f72e90ed26d943691", "adoptedAt": "2026-05-13T16:50:09Z"},
  "operationalValue": "Publish the run's assigned daily Agent-of-the-Day blog slot as a merged pull request.",
  "evidence": {
    "opportunity": "The weekday blog slot for the UTC calendar date the run started, derived from run.createdAt.",
    "assignment": "Key: blog-day:<UTC date of run.createdAt>. Duplicate runs for the same calendar date (reruns or manual dispatch) repeat the same key.",
    "accepted": "A merged github/gh-aw pull request whose body contains the gh-aw-tracker-id marker for this workflow and the exact run id of this run is remediation evidence; issues, comments, and traces are excluded.",
    "repositories": ["github/gh-aw"],
    "collection": "With pull-requests:read and issues:read, GitHub Search REST enumerates merged pull requests tagged with this workflow's tracker-id marker; each candidate's body is matched against the run's numeric id.",
    "maturation": "Seven days, matching the workflow's create-pull-request expires:7d safe-output; the adoption run's pull request (#31981) merged the same day it was opened.",
    "zeroRule": "No merged pull request tagged with this run's id by the capped cutoff scores 0.",
    "missingRule": "Invalid requests or unavailable pull-request search evidence score null; an unassignable date never occurs for a completed run."
  },
  "primaryMetric": {"id": "assigned-blog-slot-publication", "formula": "1 when a merged pull request tagged with this run's id exists by evidenceCutoff; otherwise 0.", "direction": "higher_is_better"},
  "diagnosticMetrics": [
    {"id": "merge-latency-health", "name": "Merge latency health", "formula": "clamp(1 - (mergedAt-createdAt)/604800, 0, 1) when merged by evidenceCutoff.", "direction": "higher_is_better", "aggregation": "mean"}
  ],
  "baseline": {"mode": "attainment-only", "value": null, "evidenceCutoff": null, "provenance": []},
  "validationExamples": {
    "targetAttained": {"valid":true,"merged":true},
    "targetMissed": {"valid":true,"merged":false},
    "missing": {"valid":false},
    "malformed": {"valid":"yes","merged":"true"}
  }
}
JSON
}

metric() {
    local evidence
    evidence=$(cat)
    if ! printf '%s\n' "$evidence" | jq -e . >/dev/null 2>&1; then
        printf 'null\n'
        return
    fi
    printf '%s\n' "$evidence" | jq '
      if .valid != true or (.merged | type != "boolean") then null
      elif .merged then 1
      else 0
      end'
}

normalize_timestamp() {
    jq -nr --arg timestamp "$1" '
        ($timestamp | sub("\\.[0-9]+Z$"; "Z")) as $normalized
        | if ($normalized | test("^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$"))
            and (try (($normalized | fromdateiso8601 | todateiso8601) == $normalized) catch false)
        then $normalized else error("invalid timestamp") end
    ' 2>/dev/null
}

timestamp_epoch() {
    jq -nr --arg timestamp "$1" '$timestamp | fromdateiso8601'
}

add_seconds() {
    jq -nr --arg timestamp "$1" --argjson seconds "$2" \
        '$timestamp | fromdateiso8601 + $seconds | todateiso8601'
}

date_of() {
    jq -nr --arg timestamp "$1" '$timestamp | fromdateiso8601 | strftime("%Y-%m-%d")'
}

github_api() {
    gh api "$@" 2>"$tmp_dir/gh-api-error"
}

# Searches for merged pull requests tagged with this workflow's tracker-id marker, then
# matches the run id against each candidate's body. Returns the merged_at timestamp of the
# matching pull request and its number, or nothing (empty output) when no match is found.
find_merged_evidence() {
    local repository=$1 run_id=$2 results

    results=$(github_api -X GET search/issues \
        -f "q=repo:$repository is:pr is:merged \"gh-aw-tracker-id: $TRACKER_ID\"" \
        -f sort=created -f order=desc -f per_page=100) || return 1

    printf '%s\n' "$results" | jq -c --arg runId "$run_id" '
        [.items[]
         | select(.number | type == "number")
         | select((.body // "") | test("id: " + $runId + ","; "l"))
         | {number: .number, mergedAt: .pull_request.merged_at}]
        | sort_by(.mergedAt)[0] // empty' || return 1
}

emit_null() {
    local opportunity_key=$1 case_json=$2 evidence_cutoff=$3 matures_at=$4 reason=$5
    jq -cn \
        --arg opportunityKey "$opportunity_key" --argjson case "$case_json" \
        --arg evidenceCutoff "$evidence_cutoff" --arg maturesAt "$matures_at" --arg reason "$reason" \
        '{value: null, opportunityKey: $opportunityKey, case: $case,
          evidenceCutoff: $evidenceCutoff, maturesAt: $maturesAt,
          provenance: [], diagnostics: {missingReason: $reason}}'
}

grade_run() {
    local request run_id repository workflow created_at evidence_at matures_at evidence_cutoff
    local evidence_epoch matures_epoch case_json blog_date opportunity_key
    local evidence_json merged pr_number merged_at value diagnostics merge_latency

    request=$(cat)
    if ! printf '%s\n' "$request" | jq -e '
        .schemaVersion == 1 and (.run.id | type == "string" and length > 0)
        and (.run.repository | type == "string") and (.run.workflow | type == "string")
        and (.run.createdAt | type == "string") and (.evidenceAt | type == "string")
        and (.case == null or (.case | type == "object"))' >/dev/null 2>&1; then
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
    if ! [[ $run_id =~ ^[0-9]+$ ]]; then
        emit_null "run:$run_id" '{"invalidRunId":true}' "1970-01-01T00:00:00Z" "1970-01-01T00:00:00Z" "invalid run id"
        return
    fi

    matures_at=$(add_seconds "$created_at" "$MATURATION_SECONDS")
    evidence_epoch=$(timestamp_epoch "$evidence_at")
    matures_epoch=$(timestamp_epoch "$matures_at")
    if (( evidence_epoch < matures_epoch )); then evidence_cutoff=$evidence_at; else evidence_cutoff=$matures_at; fi
    if [[ $repository != "$REPOSITORY" || $workflow != "$WORKFLOW_NAME" ]]; then
        emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" "run repository or workflow does not match the frozen contract"
        return
    fi

    case_json=$(printf '%s\n' "$request" | jq -c '.case')
    if [[ $case_json == null ]]; then
        blog_date=$(date_of "$created_at")
        case_json=$(jq -cn --arg date "$blog_date" '{date: $date}')
    elif ! printf '%s\n' "$case_json" | jq -e '
        .date | type == "string" and test("^[0-9]{4}-[0-9]{2}-[0-9]{2}$")' >/dev/null; then
        emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" "invalid-case"
        return
    else
        blog_date=$(printf '%s\n' "$case_json" | jq -r '.date')
    fi

    opportunity_key="blog-day:$blog_date"
    if ! evidence_json=$(find_merged_evidence "$repository" "$run_id"); then
        emit_null "$opportunity_key" "$case_json" "$evidence_cutoff" "$matures_at" "pull-request-search-unavailable"
        return
    fi

    merged=false
    pr_number=""
    merged_at=""
    if [[ -n $evidence_json ]]; then
        merged_at=$(printf '%s\n' "$evidence_json" | jq -r '.mergedAt // empty')
        pr_number=$(printf '%s\n' "$evidence_json" | jq -r '.number')
        if [[ -n $merged_at ]] && merged_at=$(normalize_timestamp "$merged_at"); then
            if (( $(timestamp_epoch "$merged_at") <= $(timestamp_epoch "$evidence_cutoff") )); then
                merged=true
            fi
        fi
    fi

    value=$(jq -cn --argjson merged "$merged" '{valid: true, merged: $merged}' | metric)
    diagnostics='{}'
    if [[ $merged == true ]]; then
        merge_latency=$(jq -cn \
            --argjson createdEpoch "$(timestamp_epoch "$created_at")" \
            --argjson mergedEpoch "$(timestamp_epoch "$merged_at")" \
            --argjson window "$MATURATION_SECONDS" \
            '(1 - (($mergedEpoch - $createdEpoch) / $window)) as $v
             | if $v < 0 then 0 elif $v > 1 then 1 else $v end')
        diagnostics=$(jq -cn --argjson latency "$merge_latency" '{"merge-latency-health": $latency}')
    fi

    if [[ $merged == true ]]; then
        jq -cn --argjson value "$value" --arg opportunityKey "$opportunity_key" --argjson case "$case_json" \
            --arg evidenceCutoff "$evidence_cutoff" --arg maturesAt "$matures_at" --arg repository "$repository" \
            --arg pr "$pr_number" --argjson diagnostics "$diagnostics" \
            '{value: $value, opportunityKey: $opportunityKey, case: $case,
              evidenceCutoff: $evidenceCutoff, maturesAt: $maturesAt,
              provenance: [{repository: $repository, kind: "pull-request", ref: $pr}],
              diagnostics: $diagnostics}'
    else
        jq -cn --argjson value "$value" --arg opportunityKey "$opportunity_key" --argjson case "$case_json" \
            --arg evidenceCutoff "$evidence_cutoff" --arg maturesAt "$matures_at" --arg repository "$repository" \
            --arg trackerId "$TRACKER_ID" --arg runId "$run_id" --argjson diagnostics "$diagnostics" \
            '{value: $value, opportunityKey: $opportunityKey, case: $case,
              evidenceCutoff: $evidenceCutoff, maturesAt: $maturesAt,
              provenance: [{repository: $repository, kind: "pull-request-search", ref: ("tracker-id:" + $trackerId + "/run:" + $runId)}],
              diagnostics: $diagnostics}'
    fi
}

case ${1:-} in
    --definition) definition ;;
    --metric) metric ;;
    --grade-run) grade_run ;;
    *) printf 'usage: %s --definition|--metric|--grade-run\n' "$0" >&2; exit 1 ;;
esac
