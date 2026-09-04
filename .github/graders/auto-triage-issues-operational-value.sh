#!/usr/bin/env bash

set -euo pipefail

REPOSITORY="github/gh-aw"
WORKFLOW_NAME="Auto-Triage Issues"
SOURCE_PATH=".github/workflows/auto-triage-issues.md"
ADOPTION_COMMIT="134fc10e16a282117bfb386199c1d775a0a8f288"
ADOPTED_AT="2026-09-04T13:32:06Z"
MATURATION_SECONDS=604800

definition() {
    cat <<'JSON'
{
  "schemaVersion": 4,
  "grader": "operational-value",
  "repository": "github/gh-aw",
  "workflowName": "Auto-Triage Issues",
  "sourcePath": ".github/workflows/auto-triage-issues.md",
  "adoption": {
    "commit": "134fc10e16a282117bfb386199c1d775a0a8f288",
    "adoptedAt": "2026-09-04T13:32:06Z"
  },
  "operationalValue": "Attain a labeled state for the issue assigned to the run's triage opportunity.",
  "evidence": {
    "opportunity": "One open issue needing triage, identified by issue number.",
    "assignment": "Use case.issue or event.issue.number; otherwise choose the oldest open issue with no labels at run creation. The key is issue:<number>, and duplicate keys are retained.",
    "accepted": "The issue's GitHub label state at the evidence cutoff. Any label is attainment because the workflow's direct objective is to reduce unlabeled issues.",
    "repositories": ["github/gh-aw"],
    "collection": "Read the issue and issue label events with issues:read; reverse events after the cutoff from the current label snapshot to reconstruct labels at the cutoff.",
    "maturation": "Seven days after run creation, matching the workflow's repeated scheduled triage cadence and allowing delayed maintainer label corrections.",
    "zeroRule": "A valid issue with zero labels scores 0.",
    "missingRule": "Unavailable issue data, an invalid assignment, or an issue that did not yet exist at the cutoff scores null; unavailable evidence is never zero."
  },
  "primaryMetric": {
    "id": "issue-labeled",
    "formula": "1 when the assigned issue has at least one label at the evidence cutoff; 0 when it has none; null when evidence is unavailable.",
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
    "targetAttained": {"valid": true, "labels": ["bug"]},
    "targetMissed": {"valid": true, "labels": []},
    "missing": {"valid": false},
    "malformed": {"valid": true, "labels": "bug"}
  }
}
JSON
}

metric() {
    local evidence
    evidence=$(cat)
    if ! printf '%s\n' "$evidence" | jq -e '
        .valid == true and (.labels | type == "array")
    ' >/dev/null 2>&1; then
        printf 'null\n'
        return
    fi
    printf '%s\n' "$evidence" | jq 'if (.labels | length) > 0 then 1 else 0 end'
}

normalize_timestamp() {
    jq -nr --arg timestamp "$1" '
      ($timestamp | sub("\\.[0-9]+Z$"; "Z")) as $normalized
      | if ($normalized | test("^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$"))
        and (try (($normalized | fromdateiso8601 | todateiso8601) == $normalized) catch false)
        then $normalized else error("invalid timestamp") end
    ' 2>/dev/null
}

add_seconds() {
    jq -nr --arg timestamp "$1" --argjson seconds "$2" \
        '$timestamp | fromdateiso8601 + $seconds | todateiso8601'
}

github_api() {
    gh api "$@" 2>/dev/null
}

issue_number_from_request() {
    local request=$1
    printf '%s\n' "$request" | jq -r '
      if (.case.issue | type) == "number" then .case.issue
      elif (.case.issue | type) == "string" and (.case.issue | test("^[0-9]+$")) then .case.issue
      elif (.event.issue.number | type) == "number" then .event.issue.number
      else empty end
    '
}

find_historical_issue() {
    local repository=$1 created_at=$2 issues
    issues=$(github_api "repos/$repository/issues?state=open&sort=created&direction=asc&per_page=100") || return 1
    printf '%s\n' "$issues" | jq -er --arg createdAt "$created_at" '
      [.[] | select(.pull_request == null)
        | select((.created_at | type) == "string" and .created_at <= $createdAt)
        | select((.labels | type) == "array" and (.labels | length) == 0)]
      | .[0].number
    '
}

labels_at_cutoff() {
    local repository=$1 issue_number=$2 cutoff=$3 issue events
    issue=$(github_api "repos/$repository/issues/$issue_number") || return 1
    events=$(github_api "repos/$repository/issues/$issue_number/events?per_page=100") || return 1
    printf '%s\n' "$issue" | jq -c --arg cutoff "$cutoff" --argjson events "$events" '
      (.labels | map(.name)) as $current
      | (reduce ($events | map(select(.created_at > $cutoff)) | sort_by(.created_at) | reverse[]) as $event
          ($current;
          if $event.event == "labeled" and ($event.label.name | type) == "string"
          then map(select(. != $event.label.name))
          elif $event.event == "unlabeled" and ($event.label.name | type) == "string"
          then . + [$event.label.name] | unique
          else . end))
    '
}

grade_run() {
    local request run_id repository run_created evidence_at created_at matures_at evidence_cutoff
    local issue_number labels evidence value opportunity_key case_json
    request=$(cat)
    if ! printf '%s\n' "$request" | jq -e '
      .schemaVersion == 1 and (.run.id | type == "string" and length > 0)
      and (.run.repository | type == "string" and test("^[^/]+/[^/]+$"))
      and (.run.createdAt | type == "string") and (.evidenceAt | type == "string")
    ' >/dev/null 2>&1; then
        printf '%s\n' '{"value":null,"opportunityKey":"invalid-request","case":{"invalidRequest":true},"evidenceCutoff":"1970-01-01T00:00:00Z","maturesAt":"1970-01-01T00:00:00Z","provenance":[]}'
        return
    fi

    run_id=$(printf '%s\n' "$request" | jq -r '.run.id')
    repository=$(printf '%s\n' "$request" | jq -r '.run.repository')
    run_created=$(printf '%s\n' "$request" | jq -r '.run.createdAt')
    evidence_at=$(printf '%s\n' "$request" | jq -r '.evidenceAt')
    if ! created_at=$(normalize_timestamp "$run_created") || ! evidence_at=$(normalize_timestamp "$evidence_at"); then
        printf '%s\n' '{"value":null,"opportunityKey":"invalid-timestamp","case":{"invalidTimestamp":true},"evidenceCutoff":"1970-01-01T00:00:00Z","maturesAt":"1970-01-01T00:00:00Z","provenance":[]}'
        return
    fi
    matures_at=$(add_seconds "$created_at" "$MATURATION_SECONDS")
    evidence_cutoff=$(jq -nr --arg a "$evidence_at" --arg b "$matures_at" \
        'if ($a | fromdateiso8601) < ($b | fromdateiso8601) then $a else $b end')

    issue_number=$(issue_number_from_request "$request" || true)
    if [[ -z $issue_number ]]; then
        issue_number=$(find_historical_issue "$repository" "$created_at" || true)
    fi
    if [[ ! $issue_number =~ ^[0-9]+$ ]]; then
        jq -cn --arg key "repository:$repository:triage-queue" --arg cutoff "$evidence_cutoff" \
            --arg matures "$matures_at" --arg reason "no deterministically assignable issue" \
            '{value:null,opportunityKey:$key,case:{repository:$key},evidenceCutoff:$cutoff,maturesAt:$matures,provenance:[],diagnostics:{missingReason:$reason}}'
        return
    fi

    labels=$(labels_at_cutoff "$repository" "$issue_number" "$evidence_cutoff" || true)
    if ! printf '%s\n' "$labels" | jq -e 'type == "array"' >/dev/null 2>&1; then
        jq -cn --arg key "issue:$issue_number" --arg cutoff "$evidence_cutoff" \
            --arg matures "$matures_at" --arg reason "issue evidence unavailable" \
            --argjson issue "$issue_number" \
            '{value:null,opportunityKey:$key,case:{issue:$issue},evidenceCutoff:$cutoff,maturesAt:$matures,provenance:[],diagnostics:{missingReason:$reason}}'
        return
    fi
    evidence=$(jq -cn --argjson issue "$issue_number" --argjson labels "$labels" \
        '{valid:true,issue:$issue,labels:$labels}')
    value=$(printf '%s\n' "$evidence" | metric)
    opportunity_key="issue:$issue_number"
    case_json=$(jq -cn --argjson issue "$issue_number" '{issue:$issue}')
    jq -cn --argjson value "$value" --arg key "$opportunity_key" --argjson case "$case_json" \
        --arg cutoff "$evidence_cutoff" --arg matures "$matures_at" \
        --arg repository "$repository" --argjson issue "$issue_number" \
        '{value:$value,opportunityKey:$key,case:$case,evidenceCutoff:$cutoff,maturesAt:$matures,
          provenance:[{repository:$repository,kind:"issue",ref:($issue|tostring)}]}'
}

case "${1:-}" in
    --definition) definition ;;
    --metric) metric ;;
    --grade-run) grade_run ;;
    *) printf 'usage: %s --definition|--metric|--grade-run\n' "$0" >&2; exit 2 ;;
esac
