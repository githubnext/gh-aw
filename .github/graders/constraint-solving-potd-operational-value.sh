#!/usr/bin/env bash

set -euo pipefail

export LC_ALL=C

REPOSITORY=github/gh-aw
WORKFLOW_NAME="Constraint Solving — Problem of the Day"
SOURCE_PATH=".github/workflows/constraint-solving-potd.md"
ADOPTION_COMMIT=134fc10e16a282117bfb386199c1d775a0a8f288
ADOPTED_AT=2026-09-04T13:32:06Z
MATURATION_SECONDS=604800
TITLE_PREFIX="🧩 Constraint Solving POTD:"

tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/constraint-solving-potd-operational-value.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM

definition() {
    cat <<'JSON'
{
  "schemaVersion": 4,
  "grader": "operational-value",
  "repository": "github/gh-aw",
  "workflowName": "Constraint Solving — Problem of the Day",
  "sourcePath": ".github/workflows/constraint-solving-potd.md",
  "adoption": {
    "commit": "134fc10e16a282117bfb386199c1d775a0a8f288",
    "adoptedAt": "2026-09-04T13:32:06Z"
  },
  "operationalValue": "For each daily opportunity, attain publication of one qualifying constraint-solving problem discussion.",
  "evidence": {
    "opportunity": "The UTC calendar date of a scheduled or dispatched run.",
    "assignment": "Use case.date when supplied; otherwise reconstruct the UTC date from run.createdAt. The key is discussion-date:<date>; duplicate runs retain the same key and reruns retain their run subject.",
    "accepted": "A GitHub Discussion in github/gh-aw with category Announcements, a title beginning with the frozen prefix, and created_at on the assigned UTC date no later than the evidence cutoff.",
    "repositories": ["github/gh-aw"],
    "collection": "Read the repository Discussions API with discussions:read and filter returned records by category, title, creation date, and cutoff.",
    "maturation": "Seven days after run.createdAt; evidenceCutoff is the earlier of evidenceAt and maturesAt.",
    "zeroRule": "A successfully collected, eligible opportunity with no qualifying discussion at the cutoff scores 0.",
    "missingRule": "Invalid assignment, unavailable API evidence, or malformed discussion data scores null; absence after a successful complete collection is zero."
  },
  "primaryMetric": {
    "id": "discussion-publication",
    "formula": "1 if exactly one or more qualifying discussions exists for the assigned UTC date at the cutoff, otherwise 0",
    "direction": "higher_is_better"
  },
  "diagnosticMetrics": [
    {
      "id": "qualifying-discussion-count",
      "name": "Qualifying discussion count",
      "formula": "Count of qualifying discussions for the assigned date at the cutoff",
      "direction": "higher_is_better",
      "aggregation": "latest"
    }
  ],
  "baseline": {
    "mode": "attainment-only",
    "value": null,
    "evidenceCutoff": null,
    "provenance": []
  },
  "validationExamples": {
    "targetAttained": {"valid": true, "discussionFound": true},
    "targetMissed": {"valid": true, "discussionFound": false},
    "missing": {"valid": false},
    "malformed": {"valid": "yes", "discussionFound": true}
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
      if .valid != true or (.discussionFound | type) != "boolean" then null
      elif .discussionFound then 1 else 0 end'
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

emit_null() {
    local key=$1 case_json=$2 cutoff=$3 matures=$4 reason=$5
    jq -cn --arg opportunityKey "$key" --argjson case "$case_json" \
        --arg evidenceCutoff "$cutoff" --arg maturesAt "$matures" --arg reason "$reason" \
        '{value:null, opportunityKey:$opportunityKey, case:$case,
          evidenceCutoff:$evidenceCutoff, maturesAt:$maturesAt, provenance:[],
          diagnostics: {missingReason:$reason}}'
}

discussion_count() {
    local repository=$1 date=$2 cutoff=$3 response="$tmp_dir/discussions.json"
    if ! gh api --paginate --slurp "repos/$repository/discussions?per_page=100" >"$response" 2>"$tmp_dir/gh-api-error"; then
        return 1
    fi
    jq -er --arg date "$date" --arg cutoff "$cutoff" --arg prefix "$TITLE_PREFIX" '
      [.[]
       | .[]
       | select(.category.name == "Announcements")
       | select((.title | type) == "string" and startswith($prefix))
       | select((.created_at | type) == "string")
       | select((.created_at | startswith($date)) and .created_at <= $cutoff)]
      | length
    ' "$response"
}

grade_run() {
    local request run_id repository workflow run_sha created_at evidence_at
    local created_date case_json opportunity_key matures_at evidence_cutoff
    local evidence_epoch matures_epoch count evidence value

    request=$(cat)
    if ! printf '%s\n' "$request" | jq -e '
      .schemaVersion == 1 and
      (.run.id | type == "string" and length > 0) and
      (.run.repository | type == "string") and (.run.workflow | type == "string") and
      (.run.sha | type == "string" and test("^[0-9a-f]{40}$")) and
      (.run.createdAt | type == "string") and (.evidenceAt | type == "string") and
      (.case == null or (.case | type == "object"))
    ' >/dev/null 2>&1; then
        printf '%s\n' '{"value":null,"opportunityKey":"invalid-request","case":{"invalidRequest":true},"evidenceCutoff":"1970-01-01T00:00:00Z","maturesAt":"1970-01-01T00:00:00Z","provenance":[],"diagnostics":{"missingReason":"invalid request"}}'
        return
    fi

    run_id=$(printf '%s\n' "$request" | jq -r '.run.id')
    repository=$(printf '%s\n' "$request" | jq -r '.run.repository')
    workflow=$(printf '%s\n' "$request" | jq -r '.run.workflow')
    run_sha=$(printf '%s\n' "$request" | jq -r '.run.sha')
    created_at=$(printf '%s\n' "$request" | jq -r '.run.createdAt')
    evidence_at=$(printf '%s\n' "$request" | jq -r '.evidenceAt')
    if ! created_at=$(normalize_timestamp "$created_at") || ! evidence_at=$(normalize_timestamp "$evidence_at"); then
        printf '%s\n' '{"value":null,"opportunityKey":"invalid-timestamp","case":{"invalidTimestamp":true},"evidenceCutoff":"1970-01-01T00:00:00Z","maturesAt":"1970-01-01T00:00:00Z","provenance":[],"diagnostics":{"missingReason":"invalid timestamp"}}'
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
        created_date=${created_at%%T*}
        case_json=$(jq -cn --arg date "$created_date" '{date:$date}')
    elif ! printf '%s\n' "$case_json" | jq -e '.date | type == "string" and test("^[0-9]{4}-[0-9]{2}-[0-9]{2}$")' >/dev/null; then
        emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" "invalid-case"
        return
    fi
    created_date=$(printf '%s\n' "$case_json" | jq -r '.date')
    opportunity_key="discussion-date:$created_date"

    if ! count=$(discussion_count "$repository" "$created_date" "$evidence_cutoff"); then
        emit_null "$opportunity_key" "$case_json" "$evidence_cutoff" "$matures_at" "discussion-evidence-unavailable"
        return
    fi
    evidence=$(jq -cn --argjson count "$count" '{valid:true, discussionFound:($count > 0)}')
    value=$(printf '%s\n' "$evidence" | metric)
    jq -cn --argjson value "$value" --arg opportunityKey "$opportunity_key" \
        --argjson case "$case_json" --arg evidenceCutoff "$evidence_cutoff" \
        --arg maturesAt "$matures_at" --arg repository "$repository" \
        --arg runId "$run_id" --arg sha "$run_sha" --argjson count "$count" \
        '{value:$value, opportunityKey:$opportunityKey, case:$case,
          evidenceCutoff:$evidenceCutoff, maturesAt:$maturesAt,
          provenance:[{repository:$repository, kind:"discussion", ref:($runId + "/" + $sha)}],
          diagnostics: {"qualifying-discussion-count":$count}}'
}

case ${1:-} in
    --definition) definition ;;
    --metric) metric ;;
    --grade-run) grade_run ;;
    *) printf 'usage: %s --definition|--metric|--grade-run\n' "$0" >&2; exit 1 ;;
esac
