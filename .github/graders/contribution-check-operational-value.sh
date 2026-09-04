#!/usr/bin/env bash

set -euo pipefail

export LC_ALL=C

REPOSITORY=github/gh-aw
WORKFLOW_NAME="Contribution Check"
SOURCE_PATH=".github/workflows/contribution-check.md"
ADOPTION_COMMIT="134fc10e16a282117bfb386199c1d775a0a8f288"
ADOPTED_AT="2026-09-04T13:32:06Z"
MATURATION_SECONDS=86400
REPORT_PREFIX="[Contribution Check Report]"

fail() {
    printf 'error: %s\n' "$*" >&2
    exit 1
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

definition() {
    cat <<JSON
{
  "schemaVersion": 4,
  "grader": "operational-value",
  "repository": "$REPOSITORY",
  "workflowName": "$WORKFLOW_NAME",
  "sourcePath": "$SOURCE_PATH",
  "adoption": {
    "commit": "$ADOPTION_COMMIT",
    "adoptedAt": "$ADOPTED_AT"
  },
  "operationalValue": "The contribution-check run produces a repository-visible report issue for its assigned contribution-review opportunity.",
  "evidence": {
    "opportunity": "One workflow run's contribution-review batch.",
    "assignment": "The run ID is the stable opportunity key; a null case is reconstructed from the run repository, workflow, ID, and creation time.",
    "accepted": "An issue in the configured repository with the frozen report title prefix and the exact run URL in its body, created by the evidence cutoff.",
    "repositories": ["$REPOSITORY"],
    "collection": "Read all issues created since the run with issues:read through gh api and match title, exact run URL, and creation time.",
    "maturation": "Twenty-four hours after run creation; evidenceCutoff is the earlier of evidenceAt and maturesAt.",
    "zeroRule": "A valid, available search with no matching report issue scores 0.",
    "missingRule": "Invalid requests or unavailable GitHub issue evidence score null; absence in a successful search is zero."
  },
  "primaryMetric": {
    "id": "report-issue-attainment",
    "formula": "1 when the accepted report issue exists by evidenceCutoff; 0 when the complete search finds none; null when evidence is unavailable or malformed.",
    "direction": "higher_is_better"
  },
  "diagnosticMetrics": [],
  "baseline": {
    "mode": "attainment-only",
    "value": null,
    "evidenceCutoff": null,
    "provenance": []
  },
  "validationExamples": {
    "targetAttained": {"valid": true, "reportFound": true},
    "targetMissed": {"valid": true, "reportFound": false},
    "missing": {"valid": false},
    "malformed": {"valid": "yes", "reportFound": true}
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
        if .valid != true or (.reportFound | type != "boolean") then null
        elif .reportFound then 1
        else 0
        end'
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

grade_run() {
    local request run_id repository workflow created_at evidence_at
    local matures_at evidence_cutoff evidence_epoch matures_epoch case_json
    local run_url issues_json report_found evidence value issue_url

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
        printf '%s\n' '{"value":null,"opportunityKey":"invalid-request","case":{"invalidRequest":true},"evidenceCutoff":"1970-01-01T00:00:00Z","maturesAt":"1970-01-01T00:00:00Z","provenance":[],"diagnostics":{"missingReason":"invalid request"}}'
        return
    fi

    run_id=$(printf '%s\n' "$request" | jq -r '.run.id')
    repository=$(printf '%s\n' "$request" | jq -r '.run.repository')
    workflow=$(printf '%s\n' "$request" | jq -r '.run.workflow')
    created_at=$(printf '%s\n' "$request" | jq -r '.run.createdAt')
    evidence_at=$(printf '%s\n' "$request" | jq -r '.evidenceAt')

    if ! created_at=$(normalize_timestamp "$created_at") \
        || ! evidence_at=$(normalize_timestamp "$evidence_at"); then
        printf '%s\n' '{"value":null,"opportunityKey":"invalid-timestamp","case":{"invalidTimestamp":true},"evidenceCutoff":"1970-01-01T00:00:00Z","maturesAt":"1970-01-01T00:00:00Z","provenance":[],"diagnostics":{"missingReason":"invalid timestamp"}}'
        return
    fi

    matures_at=$(add_seconds "$created_at" "$MATURATION_SECONDS")
    evidence_epoch=$(timestamp_epoch "$evidence_at")
    matures_epoch=$(timestamp_epoch "$matures_at")
    if (( evidence_epoch < matures_epoch )); then
        evidence_cutoff=$evidence_at
    else
        evidence_cutoff=$matures_at
    fi

    case_json=$(printf '%s\n' "$request" | jq -c '.case')
    if [[ $case_json == null ]]; then
        case_json=$(jq -cn \
            --arg runId "$run_id" \
            --arg repository "$repository" \
            --arg workflow "$workflow" \
            --arg createdAt "$created_at" \
            '{runId: $runId, repository: $repository, workflow: $workflow, createdAt: $createdAt}')
    elif ! printf '%s\n' "$case_json" | jq -e \
        --arg runId "$run_id" --arg repository "$repository" --arg workflow "$workflow" '
        .runId == $runId and .repository == $repository and .workflow == $workflow
        and (.createdAt | type == "string")
    ' >/dev/null 2>&1; then
        emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" "invalid-case"
        return
    fi

    if [[ $repository != "$REPOSITORY" || $workflow != "$WORKFLOW_NAME" ]]; then
        emit_null "run:$run_id" "$case_json" "$evidence_cutoff" "$matures_at" \
            "run repository or workflow does not match the frozen contract"
        return
    fi

    run_url="https://github.com/$repository/actions/runs/$run_id"
    if ! issues_json=$(gh api --paginate \
        "repos/$repository/issues?state=all&since=$created_at&per_page=100" 2>/dev/null); then
        emit_null "run:$run_id" "$case_json" "$evidence_cutoff" "$matures_at" "issue-evidence-unavailable"
        return
    fi

    report_found=$(printf '%s\n' "$issues_json" | jq -s \
        --arg prefix "$REPORT_PREFIX" \
        --arg runUrl "$run_url" \
        --arg cutoff "$evidence_cutoff" '
        [.[]
         | .[]
         | select((.pull_request // null) == null)
         | select((.title // "") | startswith($prefix))
         | select((.body // "") | contains($runUrl))
         | select((.created_at // "") <= $cutoff)]
        | length > 0')
    evidence=$(jq -cn --argjson reportFound "$report_found" '{valid: true, reportFound: $reportFound}')
    value=$(printf '%s\n' "$evidence" | metric)

    issue_url=$(printf '%s\n' "$issues_json" | jq -r \
        --arg prefix "$REPORT_PREFIX" --arg runUrl "$run_url" --arg cutoff "$evidence_cutoff" '
        [.[]
         | .[]
         | select((.pull_request // null) == null)
         | select((.title // "") | startswith($prefix))
         | select((.body // "") | contains($runUrl))
         | select((.created_at // "") <= $cutoff)
         | .html_url][0] // empty')
    if [[ -n $issue_url ]]; then
        jq -cn --argjson value "$value" --arg opportunityKey "run:$run_id" \
            --argjson case "$case_json" --arg evidenceCutoff "$evidence_cutoff" \
            --arg maturesAt "$matures_at" --arg repository "$repository" \
            --arg issueUrl "$issue_url" \
            '{value: $value, opportunityKey: $opportunityKey, case: $case,
              evidenceCutoff: $evidenceCutoff, maturesAt: $maturesAt,
              provenance: [{repository: $repository, kind: "issue", ref: $issueUrl}],
              diagnostics: {reportFound: true}}'
    else
        jq -cn --argjson value "$value" --arg opportunityKey "run:$run_id" \
            --argjson case "$case_json" --arg evidenceCutoff "$evidence_cutoff" \
            --arg maturesAt "$matures_at" \
            '{value: $value, opportunityKey: $opportunityKey, case: $case,
              evidenceCutoff: $evidenceCutoff, maturesAt: $maturesAt,
              provenance: [], diagnostics: {reportFound: false}}'
    fi
}

case ${1:-} in
    --definition) definition ;;
    --metric) metric ;;
    --grade-run) grade_run ;;
    *) fail "usage: $0 --definition|--metric|--grade-run" ;;
esac
