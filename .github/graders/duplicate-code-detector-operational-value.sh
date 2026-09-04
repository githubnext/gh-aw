#!/usr/bin/env bash

set -euo pipefail

REPOSITORY=github/gh-aw
WORKFLOW_NAME="Duplicate Code Detector"
MATURATION_SECONDS=1209600

tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/duplicate-code-detector-operational-value.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM

definition() {
    cat <<'JSON'
{
  "schemaVersion": 4, "grader": "operational-value",
  "repository": "github/gh-aw", "workflowName": "Duplicate Code Detector",
  "sourcePath": ".github/workflows/duplicate-code-detector.md",
  "adoption": {"commit": "c3c00d297e070cd7f766d2c87bb809301912e87b", "adoptedAt": "2025-10-07T06:29:50Z"},
  "operationalValue": "Resolve the duplicate-code reports assigned to a detector run through merged pull requests.",
  "evidence": {
    "opportunity": "The distinct duplicate-code reports whose generated body binds them to the detector run ID, or for legacy runs its analyzed commit.",
    "assignment": "Run ID is primary and analyzed commit SHA is the historical fallback; only '[duplicate-code] Duplicate Code:' issues qualify. The run key repeats for reruns.",
    "accepted": "GitHub issue bodies proving assignment and their closing pull requests' merged state and time. Agent output, traces, issue closure alone, and report volume are excluded.",
    "repositories": ["github/gh-aw"],
    "collection": "With issues:read and pull-requests:read, search issue bodies at the capped cutoff and query each qualifying issue's closing pull requests.",
    "maturation": "Fourteen days. Comparable pre-adoption duplicate-report evidence does not exist because this workflow introduced the report format.",
    "zeroRule": "A qualifying report without a closing pull request merged by the cutoff contributes 0.",
    "missingRule": "No qualifying report, incomplete search results, unavailable issue evidence, or malformed evidence scores null, never 0."
  },
  "primaryMetric": {"id": "duplicate-report-remediation-rate", "formula": "mergedReports / qualifyingReports, where a report is attained only when a closing pull request merged no later than the capped cutoff.", "direction": "higher_is_better"},
  "baseline": {"mode": "attainment-only", "value": null, "evidenceCutoff": null, "provenance": []},
  "validationExamples": {
    "targetAttained": {"valid":true,"qualifyingReports":2,"mergedReports":2},
    "targetMissed": {"valid":true,"qualifyingReports":2,"mergedReports":0},
    "missing": {"valid":false},
    "malformed": {"valid":"yes","qualifyingReports":"2","mergedReports":2}
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
        or ([.qualifyingReports,.mergedReports] | all(.[]; type == "number" and floor == .) | not)
        or .qualifyingReports <= 0 or .mergedReports < 0 or .mergedReports > .qualifyingReports
      then null else .mergedReports / .qualifyingReports end'
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
    local opportunity_key=$1 case_json=$2 evidence_cutoff=$3 matures_at=$4 reason=$5
    jq -cn --arg opportunityKey "$opportunity_key" --argjson case "$case_json" \
        --arg evidenceCutoff "$evidence_cutoff" --arg maturesAt "$matures_at" --arg reason "$reason" \
        '{value:null, opportunityKey:$opportunityKey, case:$case, evidenceCutoff:$evidenceCutoff,
          maturesAt:$maturesAt, provenance:[], diagnostics:{missingReason:$reason}}'
}

find_reports() {
    local run_id=$1 sha=$2 cutoff=$3 response
    response=$(gh api -X GET search/issues \
        -f q="repo:$REPOSITORY is:issue in:body \"id: $run_id\"" -f per_page=100 2>"$tmp_dir/search-error") \
        || return 1
    if ! printf '%s\n' "$response" | jq -e '.incomplete_results == false and (.items | type == "array")' >/dev/null; then
        return 1
    fi
    if [[ $(printf '%s\n' "$response" | jq '[.items[] | select(.title | startswith("[duplicate-code] Duplicate Code:"))] | length') -eq 0 ]]; then
        response=$(gh api -X GET search/issues \
            -f q="repo:$REPOSITORY is:issue in:body \"Analysis of commit\" \"$sha\"" -f per_page=100 \
            2>"$tmp_dir/search-fallback-error") || return 1
    fi
    printf '%s\n' "$response" | jq -cer --arg cutoff "$cutoff" '
      select(.incomplete_results == false)
      | [.items[]
         | select(.title | startswith("[duplicate-code] Duplicate Code:"))
         | select(.created_at <= $cutoff)
         | {number, createdAt:.created_at}]
      | unique_by(.number)'
}

report_merged_by_cutoff() {
    local number=$1 cutoff=$2 response
    response=$(gh api graphql -F owner=github -F name=gh-aw -F number="$number" \
        -f query='query($owner: String!, $name: String!, $number: Int!) {
          repository(owner: $owner, name: $name) {
            issue(number: $number) {
              closedByPullRequestsReferences(first: 20) {
                nodes { mergedAt }
              }
            }
          }
        }' 2>"$tmp_dir/issue-$number-error") || return 1
    printf '%s\n' "$response" | jq -e --arg cutoff "$cutoff" '
      [.data.repository.issue.closedByPullRequestsReferences.nodes[]?.mergedAt
       | select(type == "string" and . <= $cutoff)] | length > 0' >/dev/null
}

grade_run() {
    local request run_id repository workflow run_sha created_at evidence_at matures_at evidence_cutoff
    local evidence_epoch matures_epoch case_json reports report_numbers merged_reports evidence value provenance
    request=$(cat)
    if ! printf '%s\n' "$request" | jq -e '
      .schemaVersion == 1 and (.run.id | type == "string" and length > 0)
      and (.run.repository | type == "string") and (.run.workflow | type == "string")
      and (.run.sha | type == "string" and test("^[0-9a-f]{40}$"))
      and (.run.createdAt | type == "string") and (.evidenceAt | type == "string")
      and (.case == null or (.case | type == "object"))' >/dev/null 2>&1; then
        emit_null invalid-request '{"invalidRequest":true}' 1970-01-01T00:00:00Z 1970-01-01T00:00:00Z "invalid request"
        return
    fi
    run_id=$(printf '%s\n' "$request" | jq -r '.run.id')
    repository=$(printf '%s\n' "$request" | jq -r '.run.repository')
    workflow=$(printf '%s\n' "$request" | jq -r '.run.workflow')
    run_sha=$(printf '%s\n' "$request" | jq -r '.run.sha')
    created_at=$(printf '%s\n' "$request" | jq -r '.run.createdAt')
    evidence_at=$(printf '%s\n' "$request" | jq -r '.evidenceAt')
    if ! created_at=$(normalize_timestamp "$created_at") || ! evidence_at=$(normalize_timestamp "$evidence_at"); then
        emit_null invalid-timestamp '{"invalidTimestamp":true}' 1970-01-01T00:00:00Z 1970-01-01T00:00:00Z "invalid timestamp"
        return
    fi
    matures_at=$(add_seconds "$created_at" "$MATURATION_SECONDS")
    evidence_epoch=$(timestamp_epoch "$evidence_at")
    matures_epoch=$(timestamp_epoch "$matures_at")
    if (( evidence_epoch < matures_epoch )); then evidence_cutoff=$evidence_at; else evidence_cutoff=$matures_at; fi
    case_json=$(jq -cn --arg runId "$run_id" --arg subjectSha "$run_sha" \
        '{runId:$runId, subjectSha:$subjectSha}')
    if [[ $repository != "$REPOSITORY" || $workflow != "$WORKFLOW_NAME" ]]; then
        emit_null "run:$run_id" "$case_json" "$evidence_cutoff" "$matures_at" "run repository or workflow does not match the frozen contract"
        return
    fi
    if ! reports=$(find_reports "$run_id" "$run_sha" "$evidence_cutoff"); then
        emit_null "run:$run_id" "$case_json" "$evidence_cutoff" "$matures_at" "report-search-unavailable"
        return
    fi
    report_numbers=$(printf '%s\n' "$reports" | jq -c '[.[].number]')
    if [[ $report_numbers == '[]' ]]; then
        emit_null "run:$run_id" "$case_json" "$evidence_cutoff" "$matures_at" "no-qualifying-reports"
        return
    fi
    merged_reports=0
    while IFS= read -r number; do
        if report_merged_by_cutoff "$number" "$evidence_cutoff"; then merged_reports=$((merged_reports + 1)); fi
    done < <(printf '%s\n' "$report_numbers" | jq -r '.[]')
    evidence=$(jq -cn --argjson qualifyingReports "$(printf '%s\n' "$report_numbers" | jq length)" \
        --argjson mergedReports "$merged_reports" '{valid:true, qualifyingReports:$qualifyingReports, mergedReports:$mergedReports}')
    value=$(printf '%s\n' "$evidence" | metric)
    provenance=$(printf '%s\n' "$report_numbers" | jq -c --arg repository "$repository" \
        '[.[] | {repository:$repository, kind:"issue", ref:(tostring)}]')
    jq -cn --argjson value "$value" --arg opportunityKey "run:$run_id" --argjson case "$case_json" \
        --arg evidenceCutoff "$evidence_cutoff" --arg maturesAt "$matures_at" --argjson provenance "$provenance" \
        '{value:$value, opportunityKey:$opportunityKey, case:$case, evidenceCutoff:$evidenceCutoff,
          maturesAt:$maturesAt, provenance:$provenance}'
}

case ${1:-} in
    --definition) definition ;;
    --metric) metric ;;
    --grade-run) grade_run ;;
    *) printf 'usage: %s --definition|--metric|--grade-run\n' "$0" >&2; exit 1 ;;
esac
