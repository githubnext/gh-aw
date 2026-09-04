#!/usr/bin/env bash

set -euo pipefail

export LC_ALL=C

REPOSITORY=github/gh-aw
WORKFLOW_NAME="Daily Documentation Healer"
MATURATION_SECONDS=604800

tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/daily-doc-healer-operational-value.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM

definition() {
    cat <<'JSON'
{
  "schemaVersion": 4, "grader": "operational-value",
  "repository": "github/gh-aw", "workflowName": "Daily Documentation Healer",
  "sourcePath": ".github/workflows/daily-doc-healer.md",
  "adoption": {"commit": "bbb769d71c66f9f0bf5aa29483340063c9b60216", "adoptedAt": "2026-02-26T02:40:45Z"},
  "operationalValue": "Resolve the assigned recently closed documentation defect with an accepted documentation-automation pull request.",
  "evidence": {
    "opportunity": "An issue labeled documentation, closed during the seven days before a run, that had no accepted [docs] documentation-automation pull request by the run start.",
    "assignment": "Oldest closedAt then lowest issue number among eligible issues. Key: issue:<number>; when none exists, repository-health:no-unremediated-recent-documentation-issue. Duplicate runs retain the same key.",
    "accepted": "Only a merged github/gh-aw pull request with both documentation and automation labels, a [docs] title prefix, and #<issue> in its title or body is remediation evidence.",
    "repositories": ["github/gh-aw"],
    "collection": "With issues:read and pull-requests:read, GitHub Search REST responses enumerate issues and matching merged pull requests at the run start and evidence cutoff.",
    "maturation": "Seven days. The adoption workflow searches a seven-day issue window; its first two accepted corrections (#18743 and #19101) merged about one and three days after adoption.",
    "zeroRule": "An assigned issue without accepted remediation by the capped cutoff scores 0.",
    "missingRule": "Invalid requests or unavailable issue or pull-request search evidence score null; no eligible issue is a healthy, attained state."
  },
  "primaryMetric": {"id": "accepted-documentation-defect-remediation", "formula": "1 when no eligible issue exists, or when the assigned issue has accepted remediation merged after run start and by evidenceCutoff; 0 when assigned and no such remediation exists.", "direction": "higher_is_better"},
  "baseline": {"mode": "baseline-comparable", "value": 1, "evidenceCutoff": "2026-02-24T22:00:49Z", "provenance": [{"repository": "github/gh-aw", "kind": "issue", "ref": "18202"}, {"repository": "github/gh-aw", "kind": "pull-request", "ref": "18218"}]},
  "validationExamples": {
    "targetAttained": {"valid":true,"eligibleIssue":42,"remediationMerged":true},
    "targetMissed": {"valid":true,"eligibleIssue":42,"remediationMerged":false},
    "missing": {"valid":false},
    "malformed": {"valid":"yes","eligibleIssue":"42"}
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
        or (.eligibleIssue != null and (.eligibleIssue | type != "number" or . < 1 or floor != .))
        or (.remediationMerged | type != "boolean")
      then null
      elif .eligibleIssue == null or .remediationMerged then 1
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

github_api() {
    gh api "$@" 2>"$tmp_dir/gh-api-error"
}

eligible_issue() {
    local repository=$1 created_at=$2 start_at=$3 issues
    issues=$(github_api -X GET search/issues \
        -f "q=repo:$repository is:issue is:closed label:documentation closed:>=$(printf '%s' "$start_at" | cut -c1-10)" \
        -f sort=created -f order=asc -f per_page=100) || return 1
    printf '%s\n' "$issues" | jq -cer --arg start "$start_at" --arg end "$created_at" '
        [.items[]
         | select(.number | type == "number")
         | select(.closed_at | type == "string" and . >= $start and . <= $end)
         | {issue: .number, closedAt: .closed_at}]
        | sort_by(.closedAt, .issue)[]'
}

has_remediation() {
    local repository=$1 issue_number=$2 start_at=$3 end_at=$4 pull_requests
    pull_requests=$(github_api -X GET search/issues \
        -f "q=repo:$repository is:pr is:merged label:documentation label:automation #$issue_number" \
        -f per_page=100) || return 1
    printf '%s\n' "$pull_requests" | jq -c --arg issue_number "$issue_number" --arg start "$start_at" --arg end "$end_at" '
        any(.items[];
            (.title | type == "string" and startswith("[docs] "))
            and (.pull_request.merged_at | type == "string" and . <= $end)
            and ($start == "" or .pull_request.merged_at > $start)
            and ((.title + "\n" + (.body // "")) | test("#" + $issue_number | tostring))
        )'
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
    local request run_id repository workflow created_at evidence_at start_at matures_at evidence_cutoff
    local evidence_epoch matures_epoch case_json candidate issue_number opportunity_key remediation value

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
        start_at=$(add_seconds "$created_at" "-$MATURATION_SECONDS")
        if ! candidate=$(eligible_issue "$repository" "$created_at" "$start_at"); then
            emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" "issue-search-unavailable"
            return
        fi
        case_json=''
        while IFS= read -r candidate; do
            [[ -n $candidate ]] || continue
            if ! remediation=$(has_remediation "$repository" "$(printf '%s\n' "$candidate" | jq -r '.issue')" "" "$created_at"); then
                emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" "pull-request-search-unavailable"
                return
            fi
            if [[ $remediation == false ]]; then
                case_json=$candidate
                break
            fi
        done <<<"$candidate"
        if [[ -z $case_json ]]; then case_json='{"issue":null,"healthy":true}'; fi
    elif ! printf '%s\n' "$case_json" | jq -e '
        (.issue == null and .healthy == true)
        or (.issue | type == "number" and . >= 1 and floor == .)' >/dev/null; then
        emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" "invalid-case"
        return
    fi

    issue_number=$(printf '%s\n' "$case_json" | jq -r '.issue // empty')
    if [[ -z $issue_number ]]; then
        jq -cn --argjson case "$case_json" --arg evidenceCutoff "$evidence_cutoff" --arg maturesAt "$matures_at" \
            --arg repository "$repository" \
            '{value: 1, opportunityKey: "repository-health:no-unremediated-recent-documentation-issue", case: $case,
              evidenceCutoff: $evidenceCutoff, maturesAt: $maturesAt,
              provenance: [{repository: $repository, kind: "issue-search", ref: "closed-documentation-issues"}]}'
        return
    fi

    opportunity_key="issue:$issue_number"
    if ! remediation=$(has_remediation "$repository" "$issue_number" "$created_at" "$evidence_cutoff"); then
        emit_null "$opportunity_key" "$case_json" "$evidence_cutoff" "$matures_at" "pull-request-search-unavailable"
        return
    fi
    value=$(jq -cn --argjson issue "$issue_number" --argjson remediation "$remediation" \
        '{valid: true, eligibleIssue: $issue, remediationMerged: $remediation}' | metric)
    jq -cn --argjson value "$value" --arg opportunityKey "$opportunity_key" --argjson case "$case_json" \
        --arg evidenceCutoff "$evidence_cutoff" --arg maturesAt "$matures_at" --arg repository "$repository" \
        --arg issue "$issue_number" --argjson remediation "$remediation" \
        '{value: $value, opportunityKey: $opportunityKey, case: $case,
          evidenceCutoff: $evidenceCutoff, maturesAt: $maturesAt,
          provenance: [{repository: $repository, kind: "issue", ref: $issue},
                       {repository: $repository, kind: "pull-request-search", ref: ("documentation-automation:#" + $issue)}],
          diagnostics: {acceptedRemediation: $remediation}}'
}

case ${1:-} in
    --definition) definition ;;
    --metric) metric ;;
    --grade-run) grade_run ;;
    *) printf 'usage: %s --definition|--metric|--grade-run\n' "$0" >&2; exit 1 ;;
esac
