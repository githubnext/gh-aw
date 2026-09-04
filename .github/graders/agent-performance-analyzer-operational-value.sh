#!/usr/bin/env bash

set -euo pipefail

REPOSITORY=github/gh-aw
WORKFLOW_NAME="Agent Performance Analyzer - Meta-Orchestrator"
WORKFLOW_ID=agent-performance-analyzer
MATURATION_SECONDS=2592000

tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/agent-performance-analyzer-operational-value.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM

definition() {
    cat <<'JSON'
{
  "schemaVersion": 4,
  "grader": "operational-value",
  "repository": "github/gh-aw",
  "workflowName": "Agent Performance Analyzer - Meta-Orchestrator",
  "sourcePath": ".github/workflows/agent-performance-analyzer.md",
  "adoption": {
    "commit": "1d2348547791a4aa269c8c3341c905de444146e0",
    "adoptedAt": "2025-12-20T02:17:58Z"
  },
  "operationalValue": "Implement the improvement recommendations escalated by the assigned performance-analysis run.",
  "evidence": {
    "opportunity": "Improvement issues produced by this workflow run for critical agent or systemic findings.",
    "assignment": "Search issue bodies for the immutable marker `id: <run ID>, workflow_id: agent-performance-analyzer`; all matching issues created from run creation through the evidence cutoff are one run-level recommendation set. The persisted set is reused, and duplicate runs retain their own run key.",
    "accepted": "An assigned issue is attained only when GitHub records a pull request linked as closing that issue with a non-null mergedAt at or before the cutoff.",
    "repositories": ["github/gh-aw"],
    "collection": "With issues:read and pull-requests:read, search marker-attributed issues and query each issue's closing pull-request references through GitHub GraphQL.",
    "maturation": "Thirty days after run.createdAt; the cutoff is the earlier of evidenceAt and that timestamp.",
    "zeroRule": "A non-empty assigned recommendation set with no linked pull request merged for an issue contributes zero for that issue.",
    "missingRule": "Invalid requests, unavailable search or linkage evidence, malformed cases, and runs with no assigned improvement issue score null; unavailable evidence is never zero."
  },
  "primaryMetric": {
    "id": "recommendation-implementation-rate",
    "formula": "Number of assigned improvement issues with one or more linked pull requests merged by evidenceCutoff divided by assigned improvement issues; null when the assigned set is empty or evidence is unavailable.",
    "direction": "higher_is_better"
  },
  "baseline": {
    "mode": "attainment-only",
    "value": null,
    "evidenceCutoff": null,
    "provenance": []
  },
  "validationExamples": {
    "targetAttained": {"valid": true, "assignedIssues": 2, "implementedIssues": 2},
    "targetMissed": {"valid": true, "assignedIssues": 2, "implementedIssues": 0},
    "missing": {"valid": false},
    "malformed": {"valid": "true", "assignedIssues": "2", "implementedIssues": 0}
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
          or ([.assignedIssues, .implementedIssues] | all(.[]; type == "number" and floor == . and . >= 0) | not)
          or .assignedIssues == 0
          or .implementedIssues > .assignedIssues
        then null
        else .implementedIssues / .assignedIssues
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

github_api() {
    gh api "$@" 2>"$tmp_dir/gh-api-error"
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

assign_case() {
    local repository=$1 run_id=$2 created_at=$3 evidence_cutoff=$4 issues

    issues=$(github_api search/issues \
        -f "q=repo:$repository is:issue $run_id" \
        -f per_page=100) || return 1
    printf '%s\n' "$issues" | jq -ce \
        --arg marker "id: $run_id, workflow_id: $WORKFLOW_ID" \
        --arg start "$created_at" \
        --arg end "$evidence_cutoff" \
        --arg runId "$run_id" '
            .items
            | if type != "array" then error("missing issue results") else . end
            | map(select(
                (.body | type == "string" and contains($marker))
                and (.created_at | type == "string" and fromdateiso8601 >= ($start | fromdateiso8601)
                     and fromdateiso8601 <= ($end | fromdateiso8601))
              ) | .number)
            | unique
            | {runId: $runId, issues: .}' ||
        return 1
}

valid_case() {
    local case_json=$1 run_id=$2

    printf '%s\n' "$case_json" | jq -e --arg runId "$run_id" '
        .runId == $runId
        and (.issues | type == "array")
        and (all(.issues[]; type == "number" and floor == . and . > 0))
        and (.issues | unique | length == length)
    ' >/dev/null
}

issue_implemented() {
    local repository=$1 issue_number=$2 evidence_cutoff=$3 owner name result status

    owner=${repository%%/*}
    name=${repository#*/}
    result=$(github_api graphql \
        -f query='query($owner: String!, $name: String!, $number: Int!) {
          repository(owner: $owner, name: $name) {
            issue(number: $number) {
              closedByPullRequestsReferences(first: 100) {
                nodes { number mergedAt }
              }
            }
          }
        }' \
        -f owner="$owner" -f name="$name" -F number="$issue_number") || return 2

    if printf '%s\n' "$result" | jq -e --arg cutoff "$evidence_cutoff" '
        .data.repository.issue.closedByPullRequestsReferences.nodes
        | if type != "array" then error("missing pull request references") else . end
        | any(.[]; (.mergedAt | type == "string")
            and (.mergedAt | fromdateiso8601) <= ($cutoff | fromdateiso8601))
    '; then
        return 0
    else
        status=$?
    fi
    if [[ $status -eq 1 ]]; then
        return 1
    fi
    return 2
}

grade_run() {
    local request run_id repository workflow created_at evidence_at matures_at evidence_cutoff
    local case_json opportunity_key issue_number implemented=0 assigned=0 value provenance='[]' status

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
        emit_null "invalid-request" '{"invalidRequest":true}' \
            "1970-01-01T00:00:00Z" "1970-01-01T00:00:00Z" "invalid request"
        return
    fi

    run_id=$(printf '%s\n' "$request" | jq -r '.run.id')
    repository=$(printf '%s\n' "$request" | jq -r '.run.repository')
    workflow=$(printf '%s\n' "$request" | jq -r '.run.workflow')
    created_at=$(printf '%s\n' "$request" | jq -r '.run.createdAt')
    evidence_at=$(printf '%s\n' "$request" | jq -r '.evidenceAt')
    if ! created_at=$(normalize_timestamp "$created_at") \
        || ! evidence_at=$(normalize_timestamp "$evidence_at"); then
        emit_null "run:$run_id" '{"invalidTimestamp":true}' \
            "1970-01-01T00:00:00Z" "1970-01-01T00:00:00Z" "invalid timestamp"
        return
    fi

    matures_at=$(add_seconds "$created_at" "$MATURATION_SECONDS")
    evidence_cutoff=$(jq -nr --arg evidenceAt "$evidence_at" --arg maturesAt "$matures_at" '
        if ($evidenceAt | fromdateiso8601) < ($maturesAt | fromdateiso8601)
        then $evidenceAt else $maturesAt end')
    if [[ $repository != "$REPOSITORY" || $workflow != "$WORKFLOW_NAME" ]]; then
        emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" \
            "run repository or workflow does not match the frozen contract"
        return
    fi

    case_json=$(printf '%s\n' "$request" | jq -c '.case')
    if [[ $case_json == null ]]; then
        if ! case_json=$(assign_case "$repository" "$run_id" "$created_at" "$evidence_cutoff"); then
            emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" \
                "assignment-unavailable"
            return
        fi
    elif ! valid_case "$case_json" "$run_id"; then
        emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" \
            "invalid-case"
        return
    fi

    opportunity_key="run:$run_id:recommendation-implementation"
    assigned=$(printf '%s\n' "$case_json" | jq '.issues | length')
    if (( assigned == 0 )); then
        emit_null "$opportunity_key" "$case_json" "$evidence_cutoff" "$matures_at" \
            "no-assigned-improvement-issues"
        return
    fi

    while IFS= read -r issue_number; do
        if issue_implemented "$repository" "$issue_number" "$evidence_cutoff" >/dev/null; then
            implemented=$((implemented + 1))
        else
            status=$?
            if [[ $status -ne 1 ]]; then
                emit_null "$opportunity_key" "$case_json" "$evidence_cutoff" "$matures_at" \
                    "pull-request-linkage-unavailable"
                return
            fi
        fi
        provenance=$(printf '%s\n' "$provenance" | jq -c --arg repository "$repository" \
            --arg ref "$issue_number" '. + [{repository: $repository, kind: "issue", ref: $ref}]')
    done < <(printf '%s\n' "$case_json" | jq -r '.issues[]')

    value=$(jq -cn --argjson assignedIssues "$assigned" --argjson implementedIssues "$implemented" \
        '{valid: true, assignedIssues: $assignedIssues, implementedIssues: $implementedIssues}' | metric)
    jq -cn \
        --argjson value "$value" \
        --arg opportunityKey "$opportunity_key" \
        --argjson case "$case_json" \
        --arg evidenceCutoff "$evidence_cutoff" \
        --arg maturesAt "$matures_at" \
        --argjson provenance "$provenance" \
        '{value: $value, opportunityKey: $opportunityKey, case: $case,
          evidenceCutoff: $evidenceCutoff, maturesAt: $maturesAt,
          provenance: $provenance, diagnostics: {}}'
}

case ${1:-} in
    --definition) definition ;;
    --metric) metric ;;
    --grade-run) grade_run ;;
    *)
        printf 'usage: %s --definition|--metric|--grade-run\n' "$0" >&2
        exit 1
        ;;
esac
