#!/usr/bin/env bash

set -euo pipefail

export LC_ALL=C

REPOSITORY=github/gh-aw
WORKFLOW_NAME="Daily Caveman Optimizer"
ELIGIBLE_PATH_REGEX='^\.github/(aw|agents)/.*\.md$'
SEARCH_WINDOW_SECONDS=10800
MATURATION_SECONDS=604800

tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/daily-caveman-optimizer-operational-value.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM

local_repo_root=''
if command -v git >/dev/null 2>&1; then
    local_repo_root=$(git rev-parse --show-toplevel 2>/dev/null || true)
fi

definition() {
    cat <<'JSON'
{
  "schemaVersion": 4, "grader": "operational-value",
  "repository": "github/gh-aw", "workflowName": "Daily Caveman Optimizer",
  "sourcePath": ".github/workflows/daily-caveman-optimizer.md",
  "adoption": {"commit": "9b89368509e8c0a5520d90448853ad28bcece346", "adoptedAt": "2026-04-29T12:47:03Z"},
  "operationalValue": "Fraction of the run's own merged pull-request line reduction (in .github/aw/** and .github/agents/** instruction files) that is still present in the repository at evidence time.",
  "evidence": {
    "opportunity": "The pull request this run's safe-output job opened for its round-robin batch, identified within a bounded time window after the run started (title-prefix \"[caveman] \", labels documentation/automation/prompt-quality, author github-actions[bot]).",
    "assignment": "Nearest-created-time match in [run.createdAt, run.createdAt+3h) among candidate PRs; ties broken by lowest PR number. Key: pr:<number>; a run with no matching PR (legitimate no-op batch) has no assignable opportunity.",
    "accepted": "PR files API (deletions/additions) for the claimed reduction, plus git-evidence line counts of the same paths at the merge commit and at the evidence cutoff commit for durability; issues, comments, and traces are excluded.",
    "repositories": ["github/gh-aw"],
    "collection": "With contents:read and pull-requests:read, search merged/open PRs for assignment, read pulls/<n>/files for the claimed reduction, and count newlines of the touched paths via git (local checkout or blob API) at the merge commit and cutoff commit.",
    "maturation": "Seven days, covering the workflow's 3-day PR auto-expiry plus a buffer to observe reverts (e.g. #40158, #29246 closed unmerged within hours).",
    "zeroRule": "A PR that never merges, or a merged PR whose eligible files fully regrow to erase its claimed reduction, scores 0.",
    "missingRule": "PR search failures, unreadable file content, or a claimed reduction that is not strictly positive score null; a run with no matching PR at all (no-op) scores null (no opportunity to assign, not zero)."
  },
  "primaryMetric": {"id": "pr-reduction-durability", "formula": "reductionClaimed = sum(deletions-additions) over eligible PR files; if unmerged => 0; else netRetained = reductionClaimed - growthSinceMerge, value = clamp(netRetained/reductionClaimed, 0, 1).", "direction": "higher_is_better"},
  "diagnosticMetrics": [
    {"id": "batch-size-ratio", "name": "Assigned batch size relative to target", "formula": "min(1, eligibleFileCount/5) using the workflow's documented batch size of 5 files per run.", "direction": "higher_is_better", "aggregation": "latest"}
  ],
  "baseline": {"mode": "attainment-only", "value": null, "evidenceCutoff": null, "provenance": []},
  "validationExamples": {
    "targetAttained": {"valid":true,"merged":true,"reductionClaimed":24,"growthSinceMerge":0},
    "targetMissed": {"valid":true,"merged":false,"reductionClaimed":24,"growthSinceMerge":0},
    "missing": {"valid":false},
    "malformed": {"valid":"yes","merged":"true","reductionClaimed":"24","growthSinceMerge":"0"}
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
      if .valid != true or (.merged|type) != "boolean"
        or (.reductionClaimed|type) != "number" or (.growthSinceMerge|type) != "number"
        or .reductionClaimed <= 0 then null
      elif .merged != true then 0
      else ((.reductionClaimed - .growthSinceMerge) / .reductionClaimed) as $v
        | if $v<0 then 0 elif $v>1 then 1 else $v end
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

file_line_count_at() {
    local repository=$1 commit_sha=$2 path=$3 output=$4

    if [[ -n $local_repo_root ]] \
        && git -C "$local_repo_root" cat-file -e "$commit_sha^{commit}" 2>/dev/null; then
        if git -C "$local_repo_root" cat-file -e "$commit_sha:$path" 2>/dev/null; then
            git -C "$local_repo_root" show "$commit_sha:$path" | wc -l | tr -d ' '
        else
            printf '0\n'
        fi
        return
    fi

    local tree_file="$output.tree"
    if ! github_api "repos/$repository/git/trees/$commit_sha?recursive=1" >"$tree_file"; then
        return 1
    fi
    jq -e '.truncated == false and (.tree | type == "array")' "$tree_file" >/dev/null || return 1

    local blob_sha
    blob_sha=$(jq -r --arg path "$path" \
        '.tree[] | select(.type == "blob" and .path == $path) | .sha' "$tree_file")
    if [[ -z $blob_sha ]]; then
        printf '0\n'
        return
    fi
    if ! github_api -H "Accept: application/vnd.github.raw+json" \
        "repos/$repository/git/blobs/$blob_sha" >"$output"; then
        return 1
    fi
    wc -l <"$output" | tr -d ' '
}

paths_total_lines_at() {
    local repository=$1 commit_sha=$2 paths_file=$3
    local path lines total=0 idx=0

    while IFS= read -r path; do
        [[ -n $path ]] || continue
        idx=$((idx + 1))
        if ! lines=$(file_line_count_at "$repository" "$commit_sha" "$path" "$tmp_dir/blob-$idx"); then
            return 1
        fi
        total=$((total + lines))
    done <"$paths_file"
    printf '%s\n' "$total"
}

latest_commit_at_cutoff() {
    local repository=$1 cutoff=$2
    local repository_json default_branch commits_json local_commit

    if [[ -n $local_repo_root ]] \
        && git -C "$local_repo_root" show-ref --verify --quiet refs/remotes/origin/main; then
        local_commit=$(git -C "$local_repo_root" rev-list -1 --before="$cutoff" refs/remotes/origin/main) || return 1
        if [[ -n $local_commit ]]; then
            printf '%s\n' "$local_commit"
            return
        fi
    fi

    repository_json=$(github_api "repos/$repository") || return 1
    default_branch=$(printf '%s\n' "$repository_json" | jq -er '.default_branch | select(type == "string" and length > 0)') \
        || return 1
    commits_json=$(github_api -X GET "repos/$repository/commits" \
        -f sha="$default_branch" -f until="$cutoff" -f per_page=1) || return 1
    printf '%s\n' "$commits_json" | jq -er '.[0].sha | select(type == "string" and test("^[0-9a-f]{40}$"))'
}

# Search for the pull request this run's safe-output job opened, within a bounded
# window after the run started. Returns the PR number, or nothing if none found.
search_run_pr() {
    local repository=$1 created_at=$2
    local window_end results_json count

    window_end=$(add_seconds "$created_at" "$SEARCH_WINDOW_SECONDS")
    results_json=$(github_api -X GET "search/issues" \
        -f q="repo:$repository is:pr in:title caveman label:prompt-quality author:app/github-actions created:${created_at}..${window_end}" \
        -f sort=created -f order=asc -f per_page=10) || return 1

    count=$(printf '%s\n' "$results_json" | jq -r '.items | length')
    if [[ $count -eq 0 ]]; then
        printf '\n'
        return
    fi
    printf '%s\n' "$results_json" | jq -r \
        '.items | sort_by(.created_at, .number) | .[0].number'
}

# Build the frozen case (opportunity assignment) for a PR: the eligible instruction
# files it touched and the claimed line reduction (deletions minus additions).
assign_case() {
    local repository=$1 pr_number=$2
    local pr_json files_json base_sha reduction_claimed files_array

    pr_json=$(github_api "repos/$repository/pulls/$pr_number") || return 1
    base_sha=$(printf '%s\n' "$pr_json" | jq -er '.base.sha | select(type == "string" and test("^[0-9a-f]{40}$"))') \
        || return 1

    files_json=$(github_api -X GET "repos/$repository/pulls/$pr_number/files" -f per_page=100) || return 1
    files_array=$(printf '%s\n' "$files_json" | jq -c --arg re "$ELIGIBLE_PATH_REGEX" '
        [.[] | select(.filename | test($re)) | {path: .filename, additions: .additions, deletions: .deletions}]')
    reduction_claimed=$(printf '%s\n' "$files_array" | jq '[.[] | (.deletions - .additions)] | add // 0')

    [[ -n $files_array ]] && [[ $(printf '%s\n' "$files_array" | jq 'length') -gt 0 ]] || return 1
    (( reduction_claimed > 0 )) || return 1

    jq -cn \
        --argjson prNumber "$pr_number" \
        --arg baseSha "$base_sha" \
        --argjson files "$files_array" \
        --argjson reductionClaimed "$reduction_claimed" \
        '{prNumber: $prNumber, baseSha: $baseSha, files: $files, reductionClaimed: $reductionClaimed}'
}

case_diagnostics() {
    local case_json=$1

    printf '%s\n' "$case_json" | jq -c '
        {"batch-size-ratio": ([1, ((.files | length) / 5)] | min)}'
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
    local matures_at evidence_cutoff evidence_epoch matures_epoch
    local case_json opportunity_key evidence value diagnostics
    local pr_number pr_json merged merge_commit_sha cutoff_commit
    local paths_file merged_total cutoff_total growth_since_merge

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
        if ! pr_number=$(search_run_pr "$repository" "$created_at") || [[ -z $pr_number ]]; then
            emit_null "run:$run_id:noop" '{"noOpportunity":true}' "$evidence_cutoff" "$matures_at" \
                "no-matching-pull-request"
            return
        fi
        if ! case_json=$(assign_case "$repository" "$pr_number"); then
            emit_null "run:$run_id:noop" '{"noOpportunity":true}' "$evidence_cutoff" "$matures_at" \
                "assignment-unavailable"
            return
        fi
    elif ! printf '%s\n' "$case_json" | jq -e --arg re "$ELIGIBLE_PATH_REGEX" '
            (.prNumber | type == "number" and . > 0 and floor == .)
            and (.baseSha | type == "string" and test("^[0-9a-f]{40}$"))
            and (.files | type == "array" and length > 0
                and all(.[];
                    (.path | type == "string" and test($re))
                    and (.additions | type == "number" and . >= 0 and floor == .)
                    and (.deletions | type == "number" and . >= 0 and floor == .)))
            and (.reductionClaimed | type == "number" and . > 0 and floor == .)
            and (.reductionClaimed == ([.files[] | (.deletions - .additions)] | add))
        ' >/dev/null; then
        emit_null "run:$run_id:noop" '{"noOpportunity":true}' "$evidence_cutoff" "$matures_at" \
            "invalid-case"
        return
    fi

    pr_number=$(printf '%s\n' "$case_json" | jq -r '.prNumber')
    opportunity_key="pr:$pr_number"

    if ! pr_json=$(github_api "repos/$repository/pulls/$pr_number"); then
        emit_null "$opportunity_key" "$case_json" "$evidence_cutoff" "$matures_at" \
            "pull-request-unavailable"
        return
    fi
    merged=$(printf '%s\n' "$pr_json" | jq -r '.merged')

    diagnostics=$(case_diagnostics "$case_json")

    if [[ $merged != true ]]; then
        evidence=$(jq -cn --argjson reductionClaimed "$(printf '%s\n' "$case_json" | jq '.reductionClaimed')" \
            '{valid: true, merged: false, reductionClaimed: $reductionClaimed, growthSinceMerge: 0}')
        value=$(printf '%s\n' "$evidence" | metric)
        jq -cn \
            --argjson value "$value" \
            --arg opportunityKey "$opportunity_key" \
            --argjson case "$case_json" \
            --arg evidenceCutoff "$evidence_cutoff" \
            --arg maturesAt "$matures_at" \
            --arg repository "$repository" \
            --argjson prNumber "$pr_number" \
            --argjson diagnostics "$diagnostics" \
            '{value: $value, opportunityKey: $opportunityKey, case: $case,
              evidenceCutoff: $evidenceCutoff, maturesAt: $maturesAt,
              provenance: [{repository: $repository, kind: "pull-request", ref: ($prNumber|tostring)}],
              diagnostics: ($diagnostics + {merged: false})}'
        return
    fi

    merge_commit_sha=$(printf '%s\n' "$pr_json" | jq -r '.merge_commit_sha | select(type == "string" and test("^[0-9a-f]{40}$"))')
    if [[ -z $merge_commit_sha ]]; then
        emit_null "$opportunity_key" "$case_json" "$evidence_cutoff" "$matures_at" \
            "merge-commit-unavailable"
        return
    fi
    if ! cutoff_commit=$(latest_commit_at_cutoff "$repository" "$evidence_cutoff"); then
        emit_null "$opportunity_key" "$case_json" "$evidence_cutoff" "$matures_at" \
            "cutoff-commit-unavailable"
        return
    fi

    paths_file="$tmp_dir/pr-$pr_number-paths.txt"
    printf '%s\n' "$case_json" | jq -r '.files[].path' >"$paths_file"

    if ! merged_total=$(paths_total_lines_at "$repository" "$merge_commit_sha" "$paths_file"); then
        emit_null "$opportunity_key" "$case_json" "$evidence_cutoff" "$matures_at" \
            "merge-commit-content-unavailable"
        return
    fi
    if ! cutoff_total=$(paths_total_lines_at "$repository" "$cutoff_commit" "$paths_file"); then
        emit_null "$opportunity_key" "$case_json" "$evidence_cutoff" "$matures_at" \
            "cutoff-content-unavailable"
        return
    fi
    growth_since_merge=$((cutoff_total - merged_total))

    evidence=$(jq -cn \
        --argjson reductionClaimed "$(printf '%s\n' "$case_json" | jq '.reductionClaimed')" \
        --argjson growthSinceMerge "$growth_since_merge" \
        '{valid: true, merged: true, reductionClaimed: $reductionClaimed, growthSinceMerge: $growthSinceMerge}')
    value=$(printf '%s\n' "$evidence" | metric)

    jq -cn \
        --argjson value "$value" \
        --arg opportunityKey "$opportunity_key" \
        --argjson case "$case_json" \
        --arg evidenceCutoff "$evidence_cutoff" \
        --arg maturesAt "$matures_at" \
        --arg repository "$repository" \
        --arg mergeCommit "$merge_commit_sha" \
        --arg cutoffCommit "$cutoff_commit" \
        --argjson prNumber "$pr_number" \
        --argjson diagnostics "$diagnostics" \
        '{value: $value, opportunityKey: $opportunityKey, case: $case,
          evidenceCutoff: $evidenceCutoff, maturesAt: $maturesAt,
          provenance: [{repository: $repository, kind: "pull-request", ref: ($prNumber|tostring)},
                       {repository: $repository, kind: "git-commit", ref: $mergeCommit},
                       {repository: $repository, kind: "git-commit", ref: $cutoffCommit}],
          diagnostics: ($diagnostics + {merged: true})}'
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
