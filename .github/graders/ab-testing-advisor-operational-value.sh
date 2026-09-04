#!/usr/bin/env bash

set -euo pipefail

export LC_ALL=C

REPOSITORY=github/gh-aw
WORKFLOW_NAME="Daily A/B Testing Advisor"
MATURATION_SECONDS=1209600

tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/ab-testing-advisor-operational-value.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM

definition() {
    cat <<'JSON'
{
  "schemaVersion": 4,
  "grader": "operational-value",
  "repository": "github/gh-aw",
  "workflowName": "Daily A/B Testing Advisor",
  "sourcePath": ".github/workflows/ab-testing-advisor.md",
  "adoption": {"commit": "82239c030d6a1ef6ec8b87a80a1346eeef211f8d", "adoptedAt": "2026-09-04T06:02:13Z"},
  "operationalValue": "The assigned campaign's proposed experiment, linked to its campaign issue, and paired eval are adopted in the target workflow.",
  "evidence": {
    "opportunity": "A campaign issue created by this advisor run for one workflow that lacked experiments at assignment.",
    "assignment": "The issue body binds an Advisor run ID to its target workflow and proposed experiment; historical runs reconstruct this case by that run ID.",
    "accepted": "At the capped cutoff, the target workflow contains the proposed experiment with issue: <campaign issue number> and its paired eval ID.",
    "repositories": ["github/gh-aw"],
    "collection": "With issues:read and contents:read, read the campaign issue and the target workflow at the latest default-branch commit no later than the cutoff.",
    "maturation": "Fourteen days after run creation, matching the campaign issue expiry.",
    "zeroRule": "A resolved assignment whose target lacks either the issue-linked experiment or paired eval at the cutoff scores 0.",
    "missingRule": "Unavailable, ambiguous, malformed, or inaccessible assignment or Git evidence scores null, never 0."
  },
  "primaryMetric": {
    "id": "campaign-experiment-adoption",
    "formula": "1 when both the issue-linked proposed experiment and its paired eval are present; 0 when valid evidence shows either is absent; null when evidence is unavailable.",
    "direction": "higher_is_better"
  },
  "baseline": {"mode": "attainment-only", "value": null, "evidenceCutoff": null, "provenance": []},
  "validationExamples": {
    "targetAttained": {"valid": true, "experimentLinked": true, "evalLinked": true},
    "targetMissed": {"valid": true, "experimentLinked": true, "evalLinked": false},
    "missing": {"valid": false},
    "malformed": {"valid": "true", "experimentLinked": true, "evalLinked": true}
  }
}
JSON
}

metric() {
    jq '
        if .valid != true
          or ([.experimentLinked, .evalLinked] | all(type == "boolean") | not)
        then null
        elif .experimentLinked and .evalLinked then 1
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

assignment_from_issue() {
    local issue_json=$1 run_id=$2

    printf '%s\n' "$issue_json" | jq -cer --arg runID "$run_id" '
        .number as $issue
        | .title as $title
        | .body as $body
        | select(($issue | type) == "number" and $issue > 0 and ($issue | floor) == $issue)
        | select(($title | type) == "string" and ($title | startswith("Experiment campaign for ")))
        | select(($body | type) == "string"
            and ($body | contains("**Advisor run ID**: `" + $runID + "`")))
        | ($body | capture("\\*\\*Workflow file\\*\\*: `\\.github/workflows/(?<workflow>[A-Za-z0-9._-]+)\\.md`").workflow) as $workflow
        | ($body | capture("(?s)experiments:\\n  (?<experiment>[A-Za-z0-9_-]+):.*?\\n    issue: " + ($issue | tostring) + "(?:\\n|$)").experiment) as $experiment
        | ($body | capture("eval:(?<eval>[A-Za-z0-9_-]+)").eval) as $eval
        | {issue: $issue, workflow: $workflow, experiment: $experiment, eval: $eval}
    '
}

reconstruct_case() {
    local repository=$1 run_id=$2 search_json issue_number issue_json

    search_json=$(github_api -X GET search/issues \
        -f q="repo:$repository is:issue in:body \"Advisor run ID\" \"$run_id\"" -f per_page=100) || return 1
    issue_number=$(printf '%s\n' "$search_json" | jq -er '
        [.items[]? | select(.number | type == "number") | .number] | unique
        | if length == 1 then .[0] else error("ambiguous or missing assignment") end
    ') || return 1
    issue_json=$(github_api "repos/$repository/issues/$issue_number") || return 1
    assignment_from_issue "$issue_json" "$run_id"
}

latest_commit_at_cutoff() {
    local repository=$1 cutoff=$2 repository_json default_branch commits_json

    repository_json=$(github_api "repos/$repository") || return 1
    default_branch=$(printf '%s\n' "$repository_json" | jq -er '.default_branch | select(type == "string" and length > 0)') \
        || return 1
    commits_json=$(github_api -X GET "repos/$repository/commits" \
        -f sha="$default_branch" -f until="$cutoff" -f per_page=1) || return 1
    printf '%s\n' "$commits_json" | jq -er '.[0].sha | select(type == "string" and test("^[0-9a-f]{40}$"))'
}

target_evidence() {
    local repository=$1 cutoff_commit=$2 case_json=$3 target_path target_file
    local workflow experiment eval issue

    workflow=$(printf '%s\n' "$case_json" | jq -r '.workflow')
    experiment=$(printf '%s\n' "$case_json" | jq -r '.experiment')
    eval=$(printf '%s\n' "$case_json" | jq -r '.eval')
    issue=$(printf '%s\n' "$case_json" | jq -r '.issue')
    target_path=".github/workflows/$workflow.md"
    target_file="$tmp_dir/target-workflow.md"

    github_api -H "Accept: application/vnd.github.raw+json" \
        "repos/$repository/contents/$target_path?ref=$cutoff_commit" >"$target_file" || return 1

    if awk -v experiment="$experiment" -v issue="$issue" '
        $0 == "  " experiment ":" { in_experiment = 1; next }
        in_experiment && /^  [A-Za-z0-9_-]+:$/ { in_experiment = 0 }
        in_experiment && $0 == "    issue: " issue { found = 1 }
        END { exit(found ? 0 : 1) }
    ' "$target_file"; then
        local experiment_linked=0
    else
        local experiment_linked=1
    fi
    if grep -Fqx "  - id: $eval" "$target_file"; then
        local eval_linked=0
    else
        local eval_linked=1
    fi
    jq -cn \
        --argjson experimentLinked "$([[ $experiment_linked -eq 0 ]] && echo true || echo false)" \
        --argjson evalLinked "$([[ $eval_linked -eq 0 ]] && echo true || echo false)" \
        '{valid: true, experimentLinked: $experimentLinked, evalLinked: $evalLinked}'
}

grade_run() {
    local request run_id repository workflow created_at evidence_at matures_at evidence_cutoff
    local evidence_epoch matures_epoch case_json opportunity_key cutoff_commit evidence value

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
        if ! case_json=$(reconstruct_case "$repository" "$run_id"); then
            emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" "assignment-unavailable"
            return
        fi
    elif ! printf '%s\n' "$case_json" | jq -e '
        (.issue | type == "number" and .issue > 0 and floor == .issue)
        and (.workflow | type == "string" and test("^[A-Za-z0-9._-]+$"))
        and (.experiment | type == "string" and test("^[A-Za-z0-9_-]+$"))
        and (.eval | type == "string" and test("^[A-Za-z0-9_-]+$"))
    ' >/dev/null; then
        emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" "invalid-case"
        return
    fi

    opportunity_key=$(printf '%s\n' "$case_json" | jq -r '"campaign-issue:" + (.issue | tostring)')
    if ! cutoff_commit=$(latest_commit_at_cutoff "$repository" "$evidence_cutoff"); then
        emit_null "$opportunity_key" "$case_json" "$evidence_cutoff" "$matures_at" "cutoff-commit-unavailable"
        return
    fi
    if ! evidence=$(target_evidence "$repository" "$cutoff_commit" "$case_json"); then
        emit_null "$opportunity_key" "$case_json" "$evidence_cutoff" "$matures_at" "target-workflow-unavailable"
        return
    fi
    value=$(printf '%s\n' "$evidence" | metric)
    jq -cn \
        --argjson value "$value" \
        --arg opportunityKey "$opportunity_key" \
        --argjson case "$case_json" \
        --arg evidenceCutoff "$evidence_cutoff" \
        --arg maturesAt "$matures_at" \
        --arg repository "$repository" \
        --arg cutoffCommit "$cutoff_commit" \
        --arg targetPath ".github/workflows/$(printf '%s\n' "$case_json" | jq -r '.workflow').md" \
        '{value: $value, opportunityKey: $opportunityKey, case: $case,
          evidenceCutoff: $evidenceCutoff, maturesAt: $maturesAt,
          provenance: [{repository: $repository, kind: "git-commit", ref: $cutoffCommit},
                       {repository: $repository, kind: "workflow-source", ref: ($targetPath + "@" + $cutoffCommit)}],
          diagnostics: {}}'
}

case ${1:-} in
    --definition) definition ;;
    --metric) metric ;;
    --grade-run) grade_run ;;
    *) printf 'usage: %s --definition|--metric|--grade-run\n' "$0" >&2; exit 1 ;;
esac
