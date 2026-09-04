#!/usr/bin/env bash

set -euo pipefail

REPOSITORY=github/gh-aw
WORKFLOW_NAME="Daily Cache Strategy Analyzer"
MATURATION_SECONDS=2592000

tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/daily-cache-strategy-operational-value.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM

definition() {
    cat <<'JSON'
{
  "schemaVersion": 4, "grader": "operational-value",
  "repository": "github/gh-aw", "workflowName": "Daily Cache Strategy Analyzer",
  "sourcePath": ".github/workflows/daily-cache-strategy-analyzer.md",
  "adoption": {"commit": "0b31555f2cb0c44c6096b05df65a10fef57a64dc", "adoptedAt": "2026-09-04T15:25:17Z"},
  "operationalValue": "Resolve the cache-strategy findings assigned to the run.",
  "evidence": {
    "opportunity": "One or more cache-strategy issues created from the run's detected cache-memory findings.",
    "assignment": "Issues whose title starts [cache-strategy] and whose body contains the run's GitHub Actions URL. Key: cache-findings:<sorted issue numbers>; repeated keys are retained.",
    "accepted": "The linked issue is closed with state_reason completed by the evidence cutoff; issue creation, discussion reports, traces, and agent judgments are excluded.",
    "repositories": ["github/gh-aw"],
    "collection": "With issues:read, GitHub issue search reconstructs linked issues and the Issues API supplies each issue's state and state_reason.",
    "maturation": "Thirty days after run creation: the workflow permits seven days for safe issue output, followed by a fixed remediation observation window.",
    "zeroRule": "A linked issue that remains open or closes for a reason other than completed scores 0.",
    "missingRule": "Unavailable search or issue evidence, an invalid case, and runs with no linked remediation issue score null, never 0."
  },
  "primaryMetric": {"id": "cache-finding-remediation", "formula": "completedLinkedIssues / linkedIssues, where completed means state=closed and state_reason=completed at the capped cutoff.", "direction": "higher_is_better"},
  "baseline": {"mode": "attainment-only", "value": null, "evidenceCutoff": null, "provenance": []},
  "validationExamples": {
    "targetAttained": {"valid":true,"linkedIssues":2,"completedIssues":2},
    "targetMissed": {"valid":true,"linkedIssues":2,"completedIssues":0},
    "missing": {"valid":false},
    "malformed": {"valid":true,"linkedIssues":"2","completedIssues":2}
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
      if .valid != true
        or ([.linkedIssues,.completedIssues] | all(.[]; type == "number" and floor == .) | not)
        or .linkedIssues <= 0 or .completedIssues < 0 or .completedIssues > .linkedIssues
      then null
      else .completedIssues / .linkedIssues
      end'
}

normalize_timestamp() {
    jq -nr --arg timestamp "$1" '
      ($timestamp | sub("\\.[0-9]+Z$"; "Z")) as $normalized
      | if ($normalized | test("^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$"))
          and (try (($normalized | fromdateiso8601 | todateiso8601) == $normalized) catch false)
        then $normalized else error("invalid timestamp") end' 2>/dev/null
}

timestamp_epoch() {
    jq -nr --arg timestamp "$1" '$timestamp | fromdateiso8601'
}

add_seconds() {
    jq -nr --arg timestamp "$1" --argjson seconds "$2" \
        '$timestamp | fromdateiso8601 + $seconds | todateiso8601'
}

emit_null() {
    local key=$1 case_json=$2 cutoff=$3 maturity=$4 reason=$5
    jq -cn --arg opportunityKey "$key" --argjson case "$case_json" \
        --arg evidenceCutoff "$cutoff" --arg maturesAt "$maturity" --arg reason "$reason" \
        '{value:null,opportunityKey:$opportunityKey,case:$case,evidenceCutoff:$evidenceCutoff,
          maturesAt:$maturesAt,provenance:[],diagnostics:{missingReason:$reason}}'
}

valid_case() {
    printf '%s\n' "$1" | jq -e '
      type == "object" and (.issues | type == "array" and length > 0)
      and all(.issues[]; type == "number" and floor == . and . > 0)
      and (.issues | unique | length == length)' >/dev/null
}

reconstruct_case() {
    local repository=$1 run_id=$2 created_at=$3 cutoff=$4 results issues
    results=$(gh api -X GET search/issues -f "q=repo:$repository is:issue in:body actions/runs/$run_id" \
        -f per_page=100 2>"$tmp_dir/search-error") || return 1
    issues=$(printf '%s\n' "$results" | jq -c --arg prefix '[cache-strategy] ' \
        --arg runURL "https://github.com/$repository/actions/runs/$run_id" \
        --arg createdAt "$created_at" --arg cutoff "$cutoff" '
          [.items[]? | select((.title | startswith($prefix)) and ((.body // "") | contains($runURL))
            and (.created_at >= $createdAt) and (.created_at <= $cutoff)) | .number] | unique | sort')
    [[ $issues != '[]' ]] || return 2
    jq -cn --argjson issues "$issues" '{issues:$issues}'
}

grade_run() {
    local request repository workflow run_id created_at evidence_at matures_at cutoff
    local case_json issue_numbers issue_number issue completed=0 issue_count=0 evidence value key reconstruction_status

    request=$(cat)
    if ! printf '%s\n' "$request" | jq -e '
      .schemaVersion == 1 and (.run.id | type == "string" and length > 0)
      and (.run.repository | type == "string") and (.run.workflow | type == "string")
      and (.run.createdAt | type == "string") and (.evidenceAt | type == "string")
      and (.case == null or (.case | type == "object"))' >/dev/null 2>&1; then
        emit_null invalid-request '{"invalidRequest":true}' 1970-01-01T00:00:00Z 1970-01-01T00:00:00Z "invalid request"
        return
    fi
    repository=$(printf '%s\n' "$request" | jq -r '.run.repository')
    workflow=$(printf '%s\n' "$request" | jq -r '.run.workflow')
    run_id=$(printf '%s\n' "$request" | jq -r '.run.id')
    created_at=$(normalize_timestamp "$(printf '%s\n' "$request" | jq -r '.run.createdAt')") \
        || { emit_null "run:$run_id" '{"invalidTimestamp":true}' 1970-01-01T00:00:00Z 1970-01-01T00:00:00Z "invalid timestamp"; return; }
    evidence_at=$(normalize_timestamp "$(printf '%s\n' "$request" | jq -r '.evidenceAt')") \
        || { emit_null "run:$run_id" '{"invalidTimestamp":true}' 1970-01-01T00:00:00Z 1970-01-01T00:00:00Z "invalid timestamp"; return; }
    matures_at=$(add_seconds "$created_at" "$MATURATION_SECONDS")
    if (( $(timestamp_epoch "$evidence_at") < $(timestamp_epoch "$matures_at") )); then
        cutoff=$evidence_at
    else
        cutoff=$matures_at
    fi
    if [[ $repository != "$REPOSITORY" || $workflow != "$WORKFLOW_NAME" ]]; then
        emit_null "run:$run_id" '{"assignmentMissing":true}' "$cutoff" "$matures_at" "run does not match contract"
        return
    fi
    case_json=$(printf '%s\n' "$request" | jq -c '.case')
    if [[ $case_json == null ]]; then
        if case_json=$(reconstruct_case "$repository" "$run_id" "$created_at" "$cutoff"); then
            :
        else
            reconstruction_status=$?
            if [[ $reconstruction_status -eq 2 ]]; then
                emit_null "run:$run_id" '{"issues":[]}' "$cutoff" "$matures_at" "no linked remediation issue"
            else
                emit_null "run:$run_id" '{"assignmentMissing":true}' "$cutoff" "$matures_at" "assignment evidence unavailable"
            fi
            return
        fi
    elif ! valid_case "$case_json"; then
        emit_null "run:$run_id" '{"assignmentMissing":true}' "$cutoff" "$matures_at" "invalid case"
        return
    fi
    issue_numbers=$(printf '%s\n' "$case_json" | jq -r '.issues[]')
    while IFS= read -r issue_number; do
        issue=$(gh api "repos/$repository/issues/$issue_number" 2>"$tmp_dir/issue-error") || {
            emit_null "run:$run_id" "$case_json" "$cutoff" "$matures_at" "issue evidence unavailable"
            return
        }
        if ! printf '%s\n' "$issue" | jq -e '
          (.number | type == "number") and (.state | type == "string")
          and (.state == "open" or .state == "closed")
          and (.closed_at == null or (.closed_at | type == "string"))
          and (.state != "closed" or (.state_reason | type == "string"))' >/dev/null; then
            emit_null "run:$run_id" "$case_json" "$cutoff" "$matures_at" "malformed issue evidence"
            return
        fi
        if [[ $(printf '%s\n' "$issue" | jq -r --arg cutoff "$cutoff" \
            '.state == "closed" and .state_reason == "completed" and .closed_at != null and .closed_at <= $cutoff') == true ]]; then
            completed=$((completed + 1))
        fi
        issue_count=$((issue_count + 1))
    done <<EOF
$issue_numbers
EOF
    evidence=$(jq -cn --argjson linkedIssues "$issue_count" --argjson completedIssues "$completed" \
        '{valid:true,linkedIssues:$linkedIssues,completedIssues:$completedIssues}')
    value=$(printf '%s\n' "$evidence" | metric)
    key="cache-findings:$(printf '%s\n' "$case_json" | jq -r '.issues | sort | join(",")')"
    jq -cn --argjson value "$value" --arg opportunityKey "$key" --argjson case "$case_json" \
        --arg evidenceCutoff "$cutoff" --arg maturesAt "$matures_at" --arg repository "$repository" \
        '{value:$value,opportunityKey:$opportunityKey,case:$case,evidenceCutoff:$evidenceCutoff,
          maturesAt:$maturesAt,provenance:[$case.issues[] | {repository:$repository,kind:"issue",ref:(tostring)}]}'
}

case ${1:-} in
    --definition) definition ;;
    --metric) metric ;;
    --grade-run) grade_run ;;
    *) printf 'usage: %s --definition|--metric|--grade-run\n' "$0" >&2; exit 1 ;;
esac
