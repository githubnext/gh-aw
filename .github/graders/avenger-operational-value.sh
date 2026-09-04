#!/usr/bin/env bash

set -euo pipefail

export LC_ALL=C

REPOSITORY="github/gh-aw"
WORKFLOW_NAME="Avenger"
SOURCE_PATH=".github/workflows/avenger.md"
ADOPTION_COMMIT="134fc10e16a282117bfb386199c1d775a0a8f288"
ADOPTED_AT="2026-09-04T13:32:06Z"
MATURATION_SECONDS=604800

tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/avenger-operational-value.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM

definition() {
    cat <<'JSON'
{
  "schemaVersion": 4,
  "grader": "operational-value",
  "repository": "github/gh-aw",
  "workflowName": "Avenger",
  "sourcePath": ".github/workflows/avenger.md",
  "adoption": {
    "commit": "134fc10e16a282117bfb386199c1d775a0a8f288",
    "adoptedAt": "2026-09-04T13:32:06Z"
  },
  "operationalValue": "Restore the main branch's CI to a successful completed run after the assigned failure.",
  "evidence": {
    "opportunity": "The latest completed failing CI run on main at or before the Avenger run.",
    "assignment": "Select the latest completed CI failure created no later than the Avenger run; key: ci-run:<database id>. Preserve that key for duplicate runs and reruns.",
    "accepted": "A completed CI run on main with conclusion success, created after the assigned failure and no later than the evidence cutoff.",
    "repositories": ["github/gh-aw"],
    "collection": "Use actions:read through the GitHub Actions workflow-runs API for ci.yml on main.",
    "maturation": "Seven days after the Avenger run was created; cap evidenceCutoff at the earlier of evidenceAt and maturesAt.",
    "zeroRule": "An assigned failure with available history and no qualifying successful CI run by the cutoff scores zero.",
    "missingRule": "Unavailable or malformed assignment or Actions evidence scores null; missing success is zero when the failure history was collected."
  },
  "primaryMetric": {
    "id": "ci-recovery",
    "formula": "1 when a qualifying successful CI run exists by the cutoff, otherwise 0; unavailable evidence is null.",
    "direction": "higher_is_better"
  },
  "baseline": {
    "mode": "attainment-only",
    "value": null,
    "evidenceCutoff": null,
    "provenance": []
  },
  "validationExamples": {
    "targetAttained": {"eligible": true, "recovered": true},
    "targetMissed": {"eligible": true, "recovered": false},
    "missing": {"eligible": false, "recovered": false},
    "malformed": {"eligible": "yes", "recovered": true}
  }
}
JSON
}

metric() {
    jq '
      if (.eligible | type) != "boolean" or .eligible == false
        or (.recovered | type) != "boolean" then null
      elif .recovered then 1 else 0 end'
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

github_api() {
    gh api "$@" 2>"$tmp_dir/gh-api-error"
}

valid_repository() {
    [[ $1 =~ ^[^/]+/[^/]+$ ]]
}

run_list() {
    local repository=$1
    github_api -X GET "repos/$repository/actions/workflows/ci.yml/runs" \
        -f branch=main -f status=completed -f per_page=100
}

assign_case() {
    local repository=$1 created_at=$2 runs
    runs=$(run_list "$repository") || return 1
    printf '%s\n' "$runs" | jq -e --arg createdAt "$created_at" '
      [.workflow_runs[]? | select(
        .conclusion == "failure"
        and (.created_at | type == "string" and . <= $createdAt)
      )] | sort_by(.created_at) | last
      | {ciRunId: .id, failureCreatedAt: .created_at, failureSha: .head_sha}
      | select(
        (.ciRunId | type == "number" and floor == .)
        and (.failureCreatedAt | type == "string")
        and (.failureSha | type == "string" and test("^[0-9a-f]{40}$"))
      )
    ' >/dev/null
    printf '%s\n' "$runs" | jq -c --arg createdAt "$created_at" '
      [.workflow_runs[]? | select(
        .conclusion == "failure"
        and (.created_at | type == "string" and . <= $createdAt)
      )] | sort_by(.created_at) | last
      | {ciRunId: .id, failureCreatedAt: .created_at, failureSha: .head_sha}
    '
}

valid_case() {
    jq -e '
      (.ciRunId | type == "number" and floor == .)
      and (.failureCreatedAt | type == "string")
      and (.failureSha | type == "string" and test("^[0-9a-f]{40}$"))
    ' >/dev/null
}

emit_null() {
    local key=$1 case_json=$2 cutoff=$3 matures=$4 reason=$5
    jq -cn --arg opportunityKey "$key" --argjson case "$case_json" \
        --arg evidenceCutoff "$cutoff" --arg maturesAt "$matures" --arg reason "$reason" \
        '{value:null, opportunityKey:$opportunityKey, case:$case,
          evidenceCutoff:$evidenceCutoff, maturesAt:$maturesAt, provenance:[],
          diagnostics:{missingReason:$reason}}'
}

grade_run() {
    local request run_id repository workflow created_at evidence_at
    local matures_at evidence_cutoff case_json runs failure_id failure_at cutoff_epoch
    local recovered=false success_id success_at

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
        printf '%s\n' '{"value":null,"opportunityKey":"invalid-request","case":{"invalidRequest":true},"evidenceCutoff":"1970-01-01T00:00:00Z","maturesAt":"1970-01-01T00:00:00Z","provenance":[]}'
        return
    fi

    run_id=$(printf '%s\n' "$request" | jq -r '.run.id')
    repository=$(printf '%s\n' "$request" | jq -r '.run.repository')
    workflow=$(printf '%s\n' "$request" | jq -r '.run.workflow')
    created_at=$(printf '%s\n' "$request" | jq -r '.run.createdAt')
    evidence_at=$(printf '%s\n' "$request" | jq -r '.evidenceAt')
    if ! created_at=$(normalize_timestamp "$created_at") \
      || ! evidence_at=$(normalize_timestamp "$evidence_at"); then
        printf '%s\n' '{"value":null,"opportunityKey":"invalid-timestamp","case":{"invalidTimestamp":true},"evidenceCutoff":"1970-01-01T00:00:00Z","maturesAt":"1970-01-01T00:00:00Z","provenance":[]}'
        return
    fi

    matures_at=$(add_seconds "$created_at" "$MATURATION_SECONDS")
    if (( $(timestamp_epoch "$evidence_at") < $(timestamp_epoch "$matures_at") )); then
        evidence_cutoff=$evidence_at
    else
        evidence_cutoff=$matures_at
    fi
    if ! valid_repository "$repository" || [[ $repository != "$REPOSITORY" || $workflow != "$WORKFLOW_NAME" ]]; then
        emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" "contract mismatch"
        return
    fi

    case_json=$(printf '%s\n' "$request" | jq -c '.case')
    if [[ $case_json == null ]]; then
        case_json=$(assign_case "$repository" "$created_at") || {
            emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" "assignment unavailable"
            return
        }
    elif ! printf '%s\n' "$case_json" | valid_case; then
        emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" "invalid case"
        return
    fi

    failure_id=$(printf '%s\n' "$case_json" | jq -r '.ciRunId')
    failure_at=$(printf '%s\n' "$case_json" | jq -r '.failureCreatedAt')
    if ! failure_at=$(normalize_timestamp "$failure_at"); then
        emit_null "ci-run:$failure_id" "$case_json" "$evidence_cutoff" "$matures_at" "invalid case timestamp"
        return
    fi
    cutoff_epoch=$(timestamp_epoch "$evidence_cutoff")
    runs=$(run_list "$repository") || {
        emit_null "ci-run:$failure_id" "$case_json" "$evidence_cutoff" "$matures_at" "evidence unavailable"
        return
    }
    success_id=$(printf '%s\n' "$runs" | jq -r --arg failureAt "$failure_at" --argjson cutoff "$cutoff_epoch" '
      [.workflow_runs[]? | select(.conclusion == "success"
        and (.created_at | type == "string")
        and (.created_at > $failureAt)
        and ((.created_at | fromdateiso8601) <= $cutoff)
      )] | sort_by(.created_at) | .[0].id // empty')
    if [[ -n $success_id ]]; then
        recovered=true
        success_at=$(printf '%s\n' "$runs" | jq -r --arg id "$success_id" '.workflow_runs[] | select((.id|tostring) == $id) | .created_at' | head -n 1)
    fi
    printf '%s\n' "$case_json" | jq -c \
      --argjson recovered "$recovered" \
      --arg evidenceCutoff "$evidence_cutoff" --arg maturesAt "$matures_at" \
      --arg repository "$repository" --arg failureId "$failure_id" \
      --arg successId "$success_id" --arg successAt "${success_at:-}" '
      {eligible:true, recovered:$recovered} as $evidence
      | {value: (if $recovered then 1 else 0 end),
         opportunityKey: ("ci-run:" + $failureId), case: .,
         evidenceCutoff:$evidenceCutoff, maturesAt:$maturesAt,
         provenance: ([{repository:$repository,kind:"ci-run",ref:$failureId}]
           + (if $successId == "" then [] else [{repository:$repository,kind:"ci-run",ref:$successId}] end)),
         diagnostics:{}}'
}

case ${1:-} in
    --definition) definition ;;
    --metric) metric ;;
    --grade-run) grade_run ;;
    *) printf 'usage: %s --definition|--metric|--grade-run\n' "$0" >&2; exit 1 ;;
esac
