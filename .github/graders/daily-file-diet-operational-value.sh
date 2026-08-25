#!/usr/bin/env bash

set -euo pipefail

REPOSITORY=github/gh-aw
WORKFLOW_NAME="Daily File Diet"
THRESHOLD_LINES=1000
TARGET_LINES=500
MATURATION_SECONDS=172800

tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/daily-file-diet-operational-value.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM

definition() {
    cat <<'JSON'
{
  "schemaVersion": 4, "grader": "operational-value",
  "repository": "github/gh-aw", "workflowName": "Daily File Diet",
  "sourcePath": ".github/workflows/daily-file-diet.md",
  "adoption": {"commit": "1186030a620f4113f655c156bedf70cf2c164f79", "adoptedAt": "2025-11-15T13:36:21Z"},
  "operationalValue": "Decompose the run's largest oversized non-test Go file toward 500 lines.",
  "evidence": {
    "opportunity": "Largest non-test pkg/**/*.go blob at the run commit; below 1000 is healthy.",
    "assignment": "Greatest wc -l, lexical tie-break. Key: go-file:<path> or repository-health:non-test-go-under-1000; duplicates repeat.",
    "accepted": "Git evidence of assigned-path reduction toward 500 lines or tree-proven absence; issues and traces are excluded.",
    "repositories": ["github/gh-aw"],
    "collection": "With contents:read, inspect the run tree and latest default-branch commit by cutoff; count blob newlines.",
    "maturation": "Two days; five pre-adoption issue/PR pairs (#1636-#3564) took 0.04-0.84 days.",
    "zeroRule": "No reduction from the initial oversized file scores 0.",
    "missingRule": "Invalid, unavailable, or truncated Git evidence scores null; tree-proven path absence is attainment."
  },
  "primaryMetric": {"id": "assigned-go-file-decomposition", "formula": "initialLines < 1000 => 1; else clamp((initialLines-currentLines)/(initialLines-500),0,1). Proven absence sets currentLines=0.", "direction": "higher_is_better"},
  "baseline": {"mode": "attainment-only", "value": null, "evidenceCutoff": null, "provenance": []},
  "validationExamples": {
    "targetAttained": {"valid":true,"initialLines":1907,"currentLines":500,"thresholdLines":1000,"targetLines":500},
    "targetMissed": {"valid":true,"initialLines":1907,"currentLines":1907,"thresholdLines":1000,"targetLines":500},
    "missing": {"valid":false},
    "malformed": {"valid":"yes","initialLines":"1907"}
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
      if .valid != true or ([.initialLines,.currentLines,.thresholdLines,.targetLines]|all(.[];type=="number")|not)
        or .initialLines<0 or .currentLines<0 or .targetLines<0 or .thresholdLines<=.targetLines then null
      elif .initialLines<.thresholdLines then 1
      elif .initialLines<=.targetLines then null
      else ((.initialLines-.currentLines)/(.initialLines-.targetLines)) as $v
        | if $v<0 then 0 elif $v>1 then 1 else $v end
      end'
}

normalize_timestamp() {
    jq -nr --arg timestamp "$1" '
        if $timestamp
            | test("^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\\.[0-9]+)?Z$")
        then $timestamp | sub("\\.[0-9]+Z$"; "Z")
        else error("invalid timestamp")
        end
    '
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

blob_line_count() {
    local repository=$1
    local blob_sha=$2
    local output=$3

    if ! github_api -H "Accept: application/vnd.github.raw+json" \
        "repos/$repository/git/blobs/$blob_sha" >"$output"; then
        return 1
    fi
    wc -l <"$output" | tr -d ' '
}

load_tree() {
    local repository=$1
    local commit_sha=$2
    local output=$3

    if ! github_api "repos/$repository/git/trees/$commit_sha?recursive=1" >"$output"; then
        return 1
    fi
    jq -e '.truncated == false and (.tree | type == "array")' "$output" >/dev/null
}

assign_case() {
    local repository=$1
    local commit_sha=$2
    local tree_file="$tmp_dir/assignment-tree.json"
    local entries_file="$tmp_dir/assignment-entries.tsv"
    local blob_file="$tmp_dir/assignment-blob"
    local path
    local blob_sha
    local lines
    local largest_path=
    local largest_lines=-1

    load_tree "$repository" "$commit_sha" "$tree_file" || return 1
    jq -r '
        .tree[]
        | select(.type == "blob")
        | select(.path | test("^pkg/.*\\.go$") and (endswith("_test.go") | not))
        | [.path, .sha]
        | @tsv
    ' "$tree_file" | sort >"$entries_file"

    [[ -s $entries_file ]] || return 1
    while IFS=$'\t' read -r path blob_sha; do
        lines=$(blob_line_count "$repository" "$blob_sha" "$blob_file") || return 1
        if (( lines > largest_lines )); then
            largest_path=$path
            largest_lines=$lines
        fi
    done <"$entries_file"

    jq -cn \
        --arg path "$largest_path" \
        --argjson initialLines "$largest_lines" \
        --argjson thresholdLines "$THRESHOLD_LINES" \
        --argjson targetLines "$TARGET_LINES" \
        --arg subjectSha "$commit_sha" \
        '{
            path: $path,
            initialLines: $initialLines,
            thresholdLines: $thresholdLines,
            targetLines: $targetLines,
            subjectSha: $subjectSha
        }'
}

latest_commit_at_cutoff() {
    local repository=$1
    local cutoff=$2
    local repository_json
    local default_branch
    local commits_json

    repository_json=$(github_api "repos/$repository") || return 1
    default_branch=$(printf '%s\n' "$repository_json" | jq -er '.default_branch | select(type == "string" and length > 0)') \
        || return 1
    commits_json=$(github_api -X GET "repos/$repository/commits" \
        -f sha="$default_branch" -f until="$cutoff" -f per_page=1) || return 1
    printf '%s\n' "$commits_json" | jq -er '.[0].sha | select(type == "string" and test("^[0-9a-f]{40}$"))'
}

emit_null() {
    local opportunity_key=$1
    local case_json=$2
    local evidence_cutoff=$3
    local matures_at=$4
    local reason=$5

    jq -cn \
        --arg opportunityKey "$opportunity_key" \
        --argjson case "$case_json" \
        --arg evidenceCutoff "$evidence_cutoff" \
        --arg maturesAt "$matures_at" \
        --arg reason "$reason" \
        '{
            value: null,
            opportunityKey: $opportunityKey,
            case: $case,
            evidenceCutoff: $evidenceCutoff,
            maturesAt: $maturesAt,
            provenance: [],
            diagnostics: {missingReason: $reason}
        }'
}

grade_run() {
    local request
    local run_id
    local repository
    local workflow
    local run_sha
    local created_at
    local evidence_at
    local matures_at
    local evidence_cutoff
    local evidence_epoch
    local matures_epoch
    local case_json
    local path
    local initial_lines
    local opportunity_key
    local evidence
    local value
    local cutoff_commit
    local tree_file="$tmp_dir/cutoff-tree.json"
    local blob_sha
    local current_lines
    local blob_file="$tmp_dir/cutoff-blob"

    request=$(cat)
    if ! printf '%s\n' "$request" | jq -e '
        .schemaVersion == 1
        and (.run.id | type == "string" and length > 0)
        and (.run.repository | type == "string")
        and (.run.workflow | type == "string")
        and (.run.sha | type == "string" and test("^[0-9a-f]{40}$"))
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
    run_sha=$(printf '%s\n' "$request" | jq -r '.run.sha')
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

    if [[ $repository != "$REPOSITORY" || $workflow != "$WORKFLOW_NAME" ]]; then
        emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" \
            "run repository or workflow does not match the frozen contract"
        return
    fi

    case_json=$(printf '%s\n' "$request" | jq -c '.case')
    if [[ $case_json == null ]]; then
        if ! case_json=$(assign_case "$repository" "$run_sha"); then
            emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" \
                "assignment-unavailable"
            return
        fi
    elif ! printf '%s\n' "$case_json" | jq -e \
        --argjson threshold "$THRESHOLD_LINES" \
        --argjson target "$TARGET_LINES" '
            (.path | type == "string" and test("^pkg/.*\\.go$") and (endswith("_test.go") | not))
            and (.initialLines | type == "number" and . >= 0 and floor == .)
            and .thresholdLines == $threshold
            and .targetLines == $target
            and (.subjectSha | type == "string" and test("^[0-9a-f]{40}$"))
        ' >/dev/null; then
        emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" \
            "invalid-case"
        return
    fi

    path=$(printf '%s\n' "$case_json" | jq -r '.path')
    initial_lines=$(printf '%s\n' "$case_json" | jq -r '.initialLines')
    if (( initial_lines < THRESHOLD_LINES )); then
        opportunity_key="repository-health:non-test-go-under-1000"
        evidence=$(jq -cn \
            --argjson initialLines "$initial_lines" \
            --argjson thresholdLines "$THRESHOLD_LINES" \
            --argjson targetLines "$TARGET_LINES" \
            '{valid: true, initialLines: $initialLines, currentLines: $initialLines, thresholdLines: $thresholdLines, targetLines: $targetLines}')
        value=$(printf '%s\n' "$evidence" | metric)
        jq -cn \
            --argjson value "$value" \
            --arg opportunityKey "$opportunity_key" \
            --argjson case "$case_json" \
            --arg evidenceCutoff "$evidence_cutoff" \
            --arg maturesAt "$matures_at" \
            --arg repository "$repository" \
            --arg sha "$run_sha" \
            '{
                value: $value,
                opportunityKey: $opportunityKey,
                case: $case,
                evidenceCutoff: $evidenceCutoff,
                maturesAt: $maturesAt,
                provenance: [{repository: $repository, kind: "git-tree", ref: $sha}],
                diagnostics: {repositoryHealthyAtAssignment: true}
            }'
        return
    fi

    opportunity_key="go-file:$path"
    if ! cutoff_commit=$(latest_commit_at_cutoff "$repository" "$evidence_cutoff"); then
        emit_null "$opportunity_key" "$case_json" "$evidence_cutoff" "$matures_at" \
            "cutoff-commit-unavailable"
        return
    fi
    if ! load_tree "$repository" "$cutoff_commit" "$tree_file"; then
        emit_null "$opportunity_key" "$case_json" "$evidence_cutoff" "$matures_at" \
            "cutoff-tree-unavailable"
        return
    fi

    blob_sha=$(jq -r --arg path "$path" \
        '.tree[] | select(.type == "blob" and .path == $path) | .sha' "$tree_file")
    if [[ -z $blob_sha ]]; then
        current_lines=0
    elif ! current_lines=$(blob_line_count "$repository" "$blob_sha" "$blob_file"); then
        emit_null "$opportunity_key" "$case_json" "$evidence_cutoff" "$matures_at" \
            "blob-unavailable"
        return
    fi

    evidence=$(jq -cn \
        --argjson initialLines "$initial_lines" \
        --argjson currentLines "$current_lines" \
        --argjson thresholdLines "$THRESHOLD_LINES" \
        --argjson targetLines "$TARGET_LINES" \
        '{valid: true, initialLines: $initialLines, currentLines: $currentLines, thresholdLines: $thresholdLines, targetLines: $targetLines}')
    value=$(printf '%s\n' "$evidence" | metric)
    jq -cn \
        --argjson value "$value" \
        --arg opportunityKey "$opportunity_key" \
        --argjson case "$case_json" \
        --arg evidenceCutoff "$evidence_cutoff" \
        --arg maturesAt "$matures_at" \
        --arg repository "$repository" \
        --arg cutoffCommit "$cutoff_commit" \
        --arg path "$path" \
        --argjson currentLines "$current_lines" \
        '{
            value: $value,
            opportunityKey: $opportunityKey,
            case: $case,
            evidenceCutoff: $evidenceCutoff,
            maturesAt: $maturesAt,
            provenance: [
                {repository: $repository, kind: "git-commit", ref: $cutoffCommit},
                {repository: $repository, kind: "go-source", ref: ($path + "@" + $cutoffCommit)}
            ],
            diagnostics: {currentLines: $currentLines}
        }'
}

case ${1:-} in
    --definition)
        definition
        ;;
    --metric)
        metric
        ;;
    --grade-run)
        grade_run
        ;;
    *)
        printf 'usage: %s --definition|--metric|--grade-run\n' "$0" >&2
        exit 1
        ;;
esac
