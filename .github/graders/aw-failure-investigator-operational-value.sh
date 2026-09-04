#!/usr/bin/env bash

set -euo pipefail

export LC_ALL=C

REPOSITORY="github/gh-aw"
WORKFLOW_NAME="[aw] Failure Investigator (6h)"
WORKFLOW_ID="aw-failure-investigator"
MATURATION_SECONDS=604800
LOOKBACK_SECONDS=21600

tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/aw-failure-investigator-operational-value.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM

definition() {
    cat <<'JSON'
{
  "schemaVersion": 4,
  "grader": "operational-value",
  "repository": "github/gh-aw",
  "workflowName": "[aw] Failure Investigator (6h)",
  "sourcePath": ".github/workflows/aw-failure-investigator.md",
  "adoption": {
    "commit": "134fc10e16a282117bfb386199c1d775a0a8f288",
    "adoptedAt": "2026-09-04T13:32:06Z"
  },
  "operationalValue": "Make each qualifying agentic-workflow failure findable in repository issue tracking with a focused evidence-backed record.",
  "evidence": {
    "opportunity": "A failed agentic-workflow run in the six-hour lookback window before an investigator run.",
    "assignment": "Select each failed, timed-out, or startup-failed agentic-workflow run once by failure run ID; repeated investigator runs preserve the same key.",
    "accepted": "An issue in github/gh-aw created by the evidence cutoff that contains the exact failed run ID and is labeled agentic-workflows.",
    "repositories": ["github/gh-aw"],
    "collection": "Read Actions runs and repository issues through GH_TOKEN; filter all timestamps at the frozen cutoff and exclude the investigator workflow itself.",
    "maturation": "Seven days after the investigator run; evidenceCutoff is the earlier of evidenceAt and maturesAt.",
    "zeroRule": "An eligible failed run with no accepted tracking issue scores zero.",
    "missingRule": "Unavailable Actions or issue evidence, malformed assignments, and truncated API responses score null; no eligible failure is null."
  },
  "primaryMetric": {
    "id": "failure-tracking-attainment",
    "formula": "1 when the assigned failed run has an accepted labeled issue by the cutoff, otherwise 0; no eligible failure or unavailable evidence is null.",
    "direction": "higher_is_better"
  },
  "baseline": {
    "mode": "attainment-only",
    "value": null,
    "evidenceCutoff": null,
    "provenance": []
  },
  "validationExamples": {
    "targetAttained": {"valid": true, "eligible": true, "covered": true},
    "targetMissed": {"valid": true, "eligible": true, "covered": false},
    "missing": {"valid": false},
    "malformed": {"valid": true, "eligible": "yes", "covered": true}
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
      if (.valid != true)
        or (.eligible | type) != "boolean"
        or (.covered | type) != "boolean"
        or .eligible == false then null
      elif .covered then 1
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

run_is_failure() {
    jq -e '
      (.conclusion == "failure" or .conclusion == "timed_out" or .conclusion == "startup_failure")
      and (.path | type == "string" and endswith(".lock.yml"))
    ' >/dev/null
}

find_case() {
    local repository=$1 created_at=$2 run_id=$3
    local start_at runs
    start_at=$(jq -nr --arg timestamp "$created_at" \
        '$timestamp | fromdateiso8601 - 21600 | todateiso8601')
    runs=$(github_api --paginate --slurp "repos/$repository/actions/runs?per_page=100&created=$start_at..$created_at") || return 1
    printf '%s\n' "$runs" | jq -c --arg investigator "$WORKFLOW_ID" --arg runId "$run_id" '
      [.[].workflow_runs[]
       | select((.id|tostring) != $runId)
       | select(.name != "[aw] Failure Investigator (6h)")
       | select(.conclusion == "failure" or .conclusion == "timed_out" or .conclusion == "startup_failure")
       | select((.path|type == "string") and endswith(".lock.yml"))
       | {failureRunId: .id, failureWorkflow: .name, failurePath: .path, failureSha: .head_sha}]
      | sort_by(.failureRunId)
      | .[0] // {eligible: false}
    '
}

issue_covers_run() {
    local repository=$1 failure_run_id=$2 cutoff=$3
    local issues
    issues=$(github_api --paginate --slurp "repos/$repository/issues?state=all&labels=agentic-workflows&per_page=100") || return 1
    printf '%s\n' "$issues" | jq -r --arg runId "$failure_run_id" --arg cutoff "$cutoff" '
      [.[].[]] |
      any(.[]?;
        (.pull_request == null)
        and (.created_at | type == "string" and . <= $cutoff)
        and (((.title // "") + "\n" + (.body // "")) | test("(^|[^0-9])" + $runId + "([^0-9]|$)"))
      )
    '
}

emit_null() {
    local key=$1 case_json=$2 cutoff=$3 matures_at=$4 reason=$5
    jq -cn --arg opportunityKey "$key" --argjson case "$case_json" \
        --arg evidenceCutoff "$cutoff" --arg maturesAt "$matures_at" --arg reason "$reason" '
      {value: null, opportunityKey: $opportunityKey, case: $case,
       evidenceCutoff: $evidenceCutoff, maturesAt: $maturesAt, provenance: [],
       diagnostics: {missingReason: $reason}}
    '
}

grade_run() {
    local request run_id repository workflow created_at evidence_at
    local matures_at evidence_cutoff case_json failure_run_id key covered

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
    if ! created_at=$(normalize_timestamp "$created_at") || ! evidence_at=$(normalize_timestamp "$evidence_at"); then
        printf '%s\n' '{"value":null,"opportunityKey":"invalid-timestamp","case":{"invalidTimestamp":true},"evidenceCutoff":"1970-01-01T00:00:00Z","maturesAt":"1970-01-01T00:00:00Z","provenance":[]}'
        return
    fi
    matures_at=$(add_seconds "$created_at" "$MATURATION_SECONDS")
    if (( $(timestamp_epoch "$evidence_at") < $(timestamp_epoch "$matures_at") )); then
        evidence_cutoff=$evidence_at
    else
        evidence_cutoff=$matures_at
    fi

    if [[ $repository != "$REPOSITORY" || $workflow != "$WORKFLOW_NAME" ]]; then
        emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" "repository or workflow does not match contract"
        return
    fi

    case_json=$(printf '%s\n' "$request" | jq -c '.case')
    if [[ $case_json == null ]]; then
        if ! case_json=$(find_case "$repository" "$created_at" "$run_id"); then
            emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" "failure assignment unavailable"
            return
        fi
    fi
    if ! printf '%s\n' "$case_json" | jq -e '
      (.eligible == false)
      or ((.failureRunId | type == "number" or type == "string")
          and ((.failureRunId | tostring) | test("^[0-9]+$")))
    ' >/dev/null 2>&1; then
        emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" "invalid failure assignment"
        return
    fi
    if [[ $(printf '%s\n' "$case_json" | jq -r '.eligible // true') == false ]]; then
        jq -cn --arg opportunityKey "no-failure-window:$run_id" --argjson case "$case_json" \
            --arg evidenceCutoff "$evidence_cutoff" --arg maturesAt "$matures_at" '
          {value: null, opportunityKey: $opportunityKey, case: $case,
           evidenceCutoff: $evidenceCutoff, maturesAt: $maturesAt, provenance: [],
           diagnostics: {eligible: false}}
        '
        return
    fi

    failure_run_id=$(printf '%s\n' "$case_json" | jq -r '.failureRunId | tostring')
    key="failure-run:$failure_run_id"
    if ! covered=$(issue_covers_run "$repository" "$failure_run_id" "$evidence_cutoff"); then
        emit_null "$key" "$case_json" "$evidence_cutoff" "$matures_at" "issue evidence unavailable"
        return
    fi
    jq -cn --arg opportunityKey "$key" --argjson case "$case_json" \
        --arg evidenceCutoff "$evidence_cutoff" --arg maturesAt "$matures_at" \
        --arg repository "$repository" --arg runId "$failure_run_id" \
        --argjson value "$([[ $covered == true ]] && printf 1 || printf 0)" '
      {value: $value, opportunityKey: $opportunityKey, case: $case,
       evidenceCutoff: $evidenceCutoff, maturesAt: $maturesAt,
       provenance: [{repository: $repository, kind: "issue-tracking", ref: $runId}]}
    '
}

case ${1:-} in
    --definition) definition ;;
    --metric) metric ;;
    --grade-run) grade_run ;;
    *) printf 'usage: %s --definition|--metric|--grade-run\n' "$0" >&2; exit 1 ;;
esac
