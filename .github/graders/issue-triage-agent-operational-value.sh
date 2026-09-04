#!/usr/bin/env bash

set -euo pipefail

export LC_ALL=C

REPOSITORY=github/gh-aw
WORKFLOW_NAME="Issue Triage Agent"
MATURATION_SECONDS=604800
ALLOWED_LABELS='["bug","feature","enhancement","documentation","question","help-wanted","good-first-issue"]'

tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/issue-triage-agent-operational-value.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM

definition() {
    cat <<'JSON'
{
  "schemaVersion": 4,
  "grader": "operational-value",
  "repository": "github/gh-aw",
  "workflowName": "Issue Triage Agent",
  "sourcePath": ".github/workflows/issue-triage-agent.md",
  "adoption": {
    "commit": "b5d1b90b2e1ab16c567d0838ee8824a7aa5ba6dc",
    "adoptedAt": "2025-11-25T05:59:16Z"
  },
  "operationalValue": "Retain an allowed category label on each issue that this run demonstrably triaged.",
  "evidence": {
    "opportunity": "The issues with this run's Issue Triaged comment, which names the allowed category it assigned.",
    "assignment": "Search issue comments for the exact run URL and Issue Triaged heading, then use the sorted issue-number/label pairs. Key: issues:<comma-separated issue numbers>; repeated keys remain repeated.",
    "accepted": "The linked Issue Triaged comment and immutable issue label events showing whether its named allowed label remains applied by the cutoff.",
    "repositories": ["github/gh-aw"],
    "collection": "With issues:read, search comments by run URL, read matching issue comments, and replay issue label events through the capped cutoff. A run without a uniquely attributable comment has no reconstructable case.",
    "maturation": "Seven days after run creation, measuring whether the triage category persists beyond immediate application.",
    "zeroRule": "A reconstructable nonempty case with none of its named allowed labels applied at the cutoff scores 0.",
    "missingRule": "Unavailable, malformed, ambiguous, or absent run-linked comment or issue-event evidence scores null; it is never treated as zero."
  },
  "primaryMetric": {
    "id": "retained-triage-category-share",
    "formula": "retainedIssueCount / assignedIssueCount when the reconstructed case is nonempty and valid; 1 when every named allowed label is applied at cutoff, 0 when none is.",
    "direction": "higher_is_better"
  },
  "baseline": {
    "mode": "attainment-only",
    "value": null,
    "evidenceCutoff": null,
    "provenance": []
  },
  "validationExamples": {
    "targetAttained": {"valid": true, "assignedIssueCount": 2, "retainedIssueCount": 2},
    "targetMissed": {"valid": true, "assignedIssueCount": 2, "retainedIssueCount": 0},
    "missing": {"valid": false},
    "malformed": {"valid": "yes", "assignedIssueCount": "2"}
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
        or ([.assignedIssueCount, .retainedIssueCount] | all(.[]; type == "number" and floor == .) | not)
        or .assignedIssueCount <= 0
        or .retainedIssueCount < 0
        or .retainedIssueCount > .assignedIssueCount
      then null
      else .retainedIssueCount / .assignedIssueCount
      end'
}

normalize_timestamp() {
    jq -nr --arg timestamp "$1" '
        ($timestamp | sub("\\.[0-9]+Z$"; "Z")) as $normalized
        | if ($normalized | test("^[0-9]{4}-[0-9]{2}-[0-9]{2}:[0-9]{2}:[0-9]{2}Z$"))
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

find_case() {
    local repository=$1 run_id=$2 cutoff=$3
    local search_file="$tmp_dir/search.json" comments_file="$tmp_dir/comments.json"
    local query issue_number case_items='[]'

    query="repo:$repository \"https://github.com/$repository/actions/runs/$run_id\" in:comments"
    if ! github_api --paginate -X GET search/issues -f q="$query" -f per_page=100 >"$search_file"; then
        return 1
    fi
    jq -s 'add // []' "$search_file" | jq -e 'type == "object" and (.items | type == "array")' >/dev/null || return 1

    while IFS= read -r issue_number; do
        if ! github_api --paginate "repos/$repository/issues/$issue_number/comments?per_page=100" >"$comments_file"; then
            return 1
        fi
        case_items=$(jq -cn \
            --arg runURL "https://github.com/$repository/actions/runs/$run_id" \
            --arg cutoff "$cutoff" \
            --argjson allowed "$ALLOWED_LABELS" \
            --argjson existing "$case_items" \
            --slurpfile pages "$comments_file" '
              ($pages | add // []) as $comments
              | [$comments[]
                 | select(.body | type == "string")
                 | select(.created_at | type == "string" and . <= $cutoff)
                 | select(.body | contains("### 🏷️ Issue Triaged") and contains($runURL))
                 | .body as $body
                 | (try ($body | capture("categorized this issue as \\*\\*(?<label>[^*]+)\\*\\*").label) catch empty) as $label
                 | select($allowed | index($label))
                 | {issue: '"$issue_number"', label: $label, commentId: .id, commentCreatedAt: .created_at}]
                | if length == 1 then $existing + . else $existing end') || return 1
    done < <(jq -r '.items[].number' <(jq -s 'add // []' "$search_file"))

    printf '%s\n' "$case_items" | jq -ce '
        if length > 0 and length == (unique_by(.issue) | length)
          then {issues: sort_by(.issue)}
          else empty
          end'
}

label_retained() {
    local repository=$1 issue_number=$2 label=$3 cutoff=$4 events_file="$tmp_dir/events-$issue_number.json"

    if ! github_api --paginate "repos/$repository/issues/$issue_number/events?per_page=100" >"$events_file"; then
        return 1
    fi
    jq -s --arg label "$label" --arg cutoff "$cutoff" '
        (add // [])
        | map(select(.created_at | type == "string" and . <= $cutoff)
              | select((.event == "labeled" or .event == "unlabeled")
                       and (.label.name? == $label)))
        | sort_by(.created_at)
        | if length == 0 then null else .[-1].event == "labeled" end' "$events_file"
}

grade_run() {
    local request run_id repository workflow created_at evidence_at
    local matures_at evidence_cutoff evidence_epoch matures_epoch
    local case_json opportunity_key item issue_number label retained retained_count=0
    local evidence value provenance='[]'

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
        if ! case_json=$(find_case "$repository" "$run_id" "$evidence_cutoff"); then
            emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" "assignment-unavailable"
            return
        fi
    elif ! printf '%s\n' "$case_json" | jq -e --argjson allowed "$ALLOWED_LABELS" '
        (.issues | type == "array" and length > 0)
        and all(.issues[];
            (.issue | type == "number" and . > 0 and floor == .)
            and (.label | type == "string")
            and (.label as $label | $allowed | index($label))
            and (.commentId | type == "number" and . > 0)
            and (.commentCreatedAt | type == "string"))' >/dev/null; then
        emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" "invalid-case"
        return
    fi

    opportunity_key=$(printf '%s\n' "$case_json" | jq -r '"issues:" + ([.issues[].issue | tostring] | sort | join(","))')
    while IFS= read -r item; do
        issue_number=$(printf '%s\n' "$item" | jq -r '.issue')
        label=$(printf '%s\n' "$item" | jq -r '.label')
        if ! retained=$(label_retained "$repository" "$issue_number" "$label" "$evidence_cutoff"); then
            emit_null "$opportunity_key" "$case_json" "$evidence_cutoff" "$matures_at" "issue-events-unavailable"
            return
        fi
        [[ $retained == true ]] && retained_count=$((retained_count + 1))
        provenance=$(jq -cn \
            --argjson existing "$provenance" \
            --arg repository "$repository" \
            --arg issue "$issue_number" \
            --arg label "$label" \
            --arg comment "$(printf '%s\n' "$item" | jq -r '.commentId')" \
            '$existing + [
              {repository: $repository, kind: "issue-comment", ref: $comment},
              {repository: $repository, kind: "issue-label-events", ref: ($issue + ":" + $label)}
            ]')
    done < <(printf '%s\n' "$case_json" | jq -c '.issues[]')

    evidence=$(jq -cn --argjson assignedIssueCount "$(printf '%s\n' "$case_json" | jq '.issues | length')" --argjson retainedIssueCount "$retained_count" '{valid: true, assignedIssueCount: $assignedIssueCount, retainedIssueCount: $retainedIssueCount}')
    value=$(printf '%s\n' "$evidence" | metric)
    jq -cn --argjson value "$value" --arg opportunityKey "$opportunity_key" --argjson case "$case_json" --arg evidenceCutoff "$evidence_cutoff" --arg maturesAt "$matures_at" --argjson provenance "$provenance" '{value: $value, opportunityKey: $opportunityKey, case: $case, evidenceCutoff: $evidenceCutoff, maturesAt: $maturesAt, provenance: $provenance, diagnostics: {}}'
}

case ${1:-} in
    --definition) definition ;;
    --metric) metric ;;
    --grade-run) grade_run ;;
    *) printf 'usage: %s --definition|--metric|--grade-run\n' "$0" >&2; exit 1 ;;
esac
