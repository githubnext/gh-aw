#!/usr/bin/env bash

set -euo pipefail

export LC_ALL=C

REPOSITORY=github/gh-aw
WORKFLOW_NAME="Architecture Guardian"
DEFAULT_THRESHOLD_LINES=80
MATURATION_SECONDS=604800

tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/architecture-guardian-operational-value.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM

local_repo_root=''
if command -v git >/dev/null 2>&1; then
    local_repo_root=$(git rev-parse --show-toplevel 2>/dev/null || true)
fi

definition() {
    cat <<'JSON'
{
  "schemaVersion": 4, "grader": "operational-value",
  "repository": "github/gh-aw", "workflowName": "Architecture Guardian",
  "sourcePath": ".github/workflows/architecture-guardian.md",
  "adoption": {"commit": "559b2b35135e58298a50c265fbd1cba60a019cd7", "adoptedAt": "2026-04-08T17:05:30Z"},
  "operationalValue": "Decompose the run's longest oversized non-test Go function toward half of the configured function-size threshold.",
  "evidence": {
    "opportunity": "Longest span between consecutive `^func ` markers across pkg/**/*.go (excluding _test.go) at the run commit; function_lines threshold read from .architecture.yml (default 80), matching the workflow's own Step 3 function-size check.",
    "assignment": "Greatest line span; ties broken by reverse lexical (path, func-header-text). Key: go-function:<path>::<name> or repository-health:go-functions-under-threshold; duplicates repeat.",
    "accepted": "Git evidence that the assigned function's exact declaration header, re-scanned with the same marker method, shrinks toward thresholdLines/2, or file/header-proven absence; issue comments, PR review threads, and agent traces are excluded.",
    "repositories": ["github/gh-aw"],
    "collection": "With contents:read, scan the run commit's pkg/**/*.go archive for assignment and the evidence-cutoff commit's matching file content for the assigned function's current span.",
    "maturation": "Seven days. The only confirmed real remediation (issue #28113, opened 2026-04-23T15:04:17Z) was closed by merged PR #28114 in 0.06 days, but the two prior violations issues (#26048, #26230) auto-expired after 2 days and closed not_planned with no code change (#26230's linked PR #26231 was closed unmerged), so genuine fixes can land after the 2-day issue expiry.",
    "zeroRule": "No reduction from the initial oversized function scores 0.",
    "missingRule": "Invalid, unavailable, or truncated Git evidence scores null; file/header-proven absence of the assigned function is attainment."
  },
  "primaryMetric": {"id": "assigned-go-function-decomposition", "formula": "initialLines < thresholdLines => 1; else clamp((initialLines-currentLines)/(initialLines-targetLines),0,1) where targetLines=floor(thresholdLines/2). Proven header absence sets currentLines=0.", "direction": "higher_is_better"},
  "diagnosticMetrics": [
    {"id": "longest-function-health", "name": "Longest-function health at assignment", "formula": "min(1, (thresholdLines-1) / initialLines) when the assignment archive contains eligible functions.", "direction": "higher_is_better", "aggregation": "latest"},
    {"id": "compliant-function-line-share", "name": "Compliant function-line share at assignment", "formula": "compliantFunctionLines / totalFunctionLines when the assignment archive contains eligible functions and positive function-line mass.", "direction": "higher_is_better", "aggregation": "latest"}
  ],
  "baseline": {"mode": "attainment-only", "value": null, "evidenceCutoff": null, "provenance": []},
  "validationExamples": {
    "targetAttained": {"valid":true,"initialLines":852,"currentLines":40,"thresholdLines":80,"targetLines":40},
    "targetMissed": {"valid":true,"initialLines":852,"currentLines":852,"thresholdLines":80,"targetLines":40},
    "missing": {"valid":false},
    "malformed": {"valid":"yes","initialLines":"852"}
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

# Reads the `function_lines` threshold from an .architecture.yml file's content
# (or the default when absent/invalid). Mirrors the workflow's own Step 1 parsing.
read_threshold_from_config() {
    local config_content=$1 value
    value=$(printf '%s\n' "$config_content" | grep -E '^\s*function_lines:' 2>/dev/null | awk '{print $2}' | tr -d '"' | head -1 || true)
    if [[ -n $value && $value =~ ^[0-9]+$ ]]; then
        printf '%s\n' "$value"
    else
        printf '%s\n' "$DEFAULT_THRESHOLD_LINES"
    fi
}

# Prints "header\tlines" pairs for each `^func ` span in a file, using the same
# marker-to-marker line-count method as the workflow's own pre-step.
scan_functions() {
    local file=$1
    awk '/^func /{if(start>0 && header!="") printf "%s\t%d\n", header, NR-start; header=$0; start=NR} END{if(start>0 && header!="") printf "%s\t%d\n", header, NR-start+1}' "$file" 2>/dev/null || true
}

extract_func_name() {
    printf '%s\n' "$1" | sed -E 's/^func[[:space:]]+(\([^)]*\)[[:space:]]*)?([A-Za-z0-9_]+).*/\2/'
}

assign_case() {
    local repository=$1 commit_sha=$2
    local archive_file="$tmp_dir/assignment-archive.tar.gz"
    local extract_dir="$tmp_dir/assignment-archive"
    local root_dir path header lines threshold_lines=$DEFAULT_THRESHOLD_LINES target_lines
    local largest_path='' largest_header='' largest_lines=-1
    local eligible_function_count=0 total_function_lines=0 compliant_function_lines=0

    mkdir -p "$extract_dir"
    if [[ -n $local_repo_root ]] \
        && git -C "$local_repo_root" cat-file -e "$commit_sha^{commit}" 2>/dev/null; then
        git -C "$local_repo_root" archive "$commit_sha" -- pkg .architecture.yml 2>/dev/null | tar -xf - -C "$extract_dir" \
            || git -C "$local_repo_root" archive "$commit_sha" -- pkg | tar -xf - -C "$extract_dir" || return 1
        root_dir=$extract_dir
    else
        github_api -H "Accept: application/vnd.github+json" \
            "repos/$repository/tarball/$commit_sha" >"$archive_file" || return 1
        tar -xzf "$archive_file" -C "$extract_dir" || return 1
        root_dir=$(find "$extract_dir" -mindepth 1 -maxdepth 1 -type d -print -quit)
    fi
    [[ -n $root_dir && -d $root_dir/pkg ]] || return 1

    if [[ -f "$root_dir/.architecture.yml" ]]; then
        threshold_lines=$(read_threshold_from_config "$(cat "$root_dir/.architecture.yml")")
    fi
    target_lines=$((threshold_lines / 2))
    (( threshold_lines > 0 && target_lines < threshold_lines )) || return 1

    while IFS= read -r -d '' path; do
        while IFS=$'\t' read -r header lines; do
            [[ -z $header ]] && continue
            eligible_function_count=$((eligible_function_count + 1))
            total_function_lines=$((total_function_lines + lines))
            if (( lines < threshold_lines )); then
                compliant_function_lines=$((compliant_function_lines + lines))
            fi
            if (( lines > largest_lines )) \
                || { (( lines == largest_lines )) && [[ "$path|$header" > "$largest_path|$largest_header" ]]; }; then
                largest_path=$path
                largest_header=$header
                largest_lines=$lines
            fi
        done < <(scan_functions "$root_dir/$path")
    done < <(cd "$root_dir" && find pkg -type f -name '*.go' ! -name '*_test.go' -print0)

    [[ -n $largest_path ]] || return 1

    jq -cn \
        --arg path "$largest_path" \
        --arg header "$largest_header" \
        --argjson initialLines "$largest_lines" \
        --argjson thresholdLines "$threshold_lines" \
        --argjson targetLines "$target_lines" \
        --arg subjectSha "$commit_sha" \
        --argjson eligibleFunctionCount "$eligible_function_count" \
        --argjson totalFunctionLines "$total_function_lines" \
        --argjson compliantFunctionLines "$compliant_function_lines" \
        '{path: $path, header: $header, initialLines: $initialLines, thresholdLines: $thresholdLines,
          targetLines: $targetLines, subjectSha: $subjectSha,
          eligibleFunctionCount: $eligibleFunctionCount, totalFunctionLines: $totalFunctionLines,
          compliantFunctionLines: $compliantFunctionLines}'
}

case_diagnostics() {
    local case_json=$1 current_lines=$2

    printf '%s\n' "$case_json" | jq -c \
        --argjson currentLines "$current_lines" '
        {
            "longest-function-health":
                (if (.thresholdLines | type) == "number" and (.initialLines | type) == "number" and .initialLines > 0
                 then ([1, ((.thresholdLines - 1) / .initialLines)] | min) else null end),
            "compliant-function-line-share":
                (if (.eligibleFunctionCount | type) == "number" and .eligibleFunctionCount > 0
                        and (.totalFunctionLines | type) == "number" and .totalFunctionLines > 0
                        and (.compliantFunctionLines | type) == "number"
                 then (.compliantFunctionLines / .totalFunctionLines) else null end),
            currentLines: $currentLines
        }'
}

load_tree() {
    local repository=$1 commit_sha=$2 output=$3

    if ! github_api "repos/$repository/git/trees/$commit_sha?recursive=1" >"$output"; then
        return 1
    fi
    jq -e '.truncated == false and (.tree | type == "array")' "$output" >/dev/null
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
    local request run_id repository workflow run_sha created_at evidence_at
    local matures_at evidence_cutoff evidence_epoch matures_epoch
    local case_json path header func_name initial_lines threshold_lines target_lines
    local opportunity_key evidence value diagnostics
    local cutoff_commit current_lines have_file=false blob_sha
    local tree_file="$tmp_dir/cutoff-tree.json"
    local blob_file="$tmp_dir/cutoff-blob"
    local file_content_file="$tmp_dir/cutoff-file"

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
    elif ! printf '%s\n' "$case_json" | jq -e '
            (.path | type == "string" and test("^pkg/.*\\.go$") and (endswith("_test.go") | not))
            and (.header | type == "string" and length > 0)
            and (.initialLines | type == "number" and . >= 0 and floor == .)
            and (.thresholdLines | type == "number" and . > 0 and floor == .)
            and (.targetLines | type == "number" and . >= 0 and floor == .)
            and (.targetLines < .thresholdLines)
            and (.subjectSha | type == "string" and test("^[0-9a-f]{40}$"))
            and ((has("eligibleFunctionCount") | not) or (.eligibleFunctionCount | type == "number" and . >= 0 and floor == .))
            and ((has("totalFunctionLines") | not) or (.totalFunctionLines | type == "number" and . >= 0 and floor == .))
            and ((has("compliantFunctionLines") | not) or (.compliantFunctionLines | type == "number" and . >= 0 and floor == .))
        ' >/dev/null; then
        emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" \
            "invalid-case"
        return
    fi

    path=$(printf '%s\n' "$case_json" | jq -r '.path')
    header=$(printf '%s\n' "$case_json" | jq -r '.header')
    initial_lines=$(printf '%s\n' "$case_json" | jq -r '.initialLines')
    threshold_lines=$(printf '%s\n' "$case_json" | jq -r '.thresholdLines')
    target_lines=$(printf '%s\n' "$case_json" | jq -r '.targetLines')
    func_name=$(extract_func_name "$header")

    if (( initial_lines < threshold_lines )); then
        opportunity_key="repository-health:go-functions-under-threshold"
        evidence=$(jq -cn \
            --argjson initialLines "$initial_lines" \
            --argjson thresholdLines "$threshold_lines" \
            --argjson targetLines "$target_lines" \
            '{valid: true, initialLines: $initialLines, currentLines: $initialLines, thresholdLines: $thresholdLines, targetLines: $targetLines}')
        value=$(printf '%s\n' "$evidence" | metric)
        diagnostics=$(case_diagnostics "$case_json" "$initial_lines")
        jq -cn \
            --argjson value "$value" \
            --arg opportunityKey "$opportunity_key" \
            --argjson case "$case_json" \
            --arg evidenceCutoff "$evidence_cutoff" \
            --arg maturesAt "$matures_at" \
            --arg repository "$repository" \
            --arg sha "$run_sha" \
            --argjson diagnostics "$diagnostics" \
            '{value: $value, opportunityKey: $opportunityKey, case: $case,
              evidenceCutoff: $evidenceCutoff, maturesAt: $maturesAt,
              provenance: [{repository: $repository, kind: "git-tree", ref: $sha}],
              diagnostics: ($diagnostics + {repositoryHealthyAtAssignment: true})}'
        return
    fi

    opportunity_key="go-function:$path::$func_name"
    if ! cutoff_commit=$(latest_commit_at_cutoff "$repository" "$evidence_cutoff"); then
        emit_null "$opportunity_key" "$case_json" "$evidence_cutoff" "$matures_at" \
            "cutoff-commit-unavailable"
        return
    fi

    have_file=false
    if [[ -n $local_repo_root ]] \
        && git -C "$local_repo_root" cat-file -e "$cutoff_commit^{commit}" 2>/dev/null; then
        if git -C "$local_repo_root" cat-file -e "$cutoff_commit:$path" 2>/dev/null; then
            git -C "$local_repo_root" show "$cutoff_commit:$path" >"$file_content_file" || return 1
            have_file=true
        fi
    else
        if ! load_tree "$repository" "$cutoff_commit" "$tree_file"; then
            emit_null "$opportunity_key" "$case_json" "$evidence_cutoff" "$matures_at" \
                "cutoff-tree-unavailable"
            return
        fi
        blob_sha=$(jq -r --arg path "$path" \
            '.tree[] | select(.type == "blob" and .path == $path) | .sha' "$tree_file")
        if [[ -n $blob_sha ]]; then
            if ! github_api -H "Accept: application/vnd.github.raw+json" \
                "repos/$repository/git/blobs/$blob_sha" >"$file_content_file"; then
                emit_null "$opportunity_key" "$case_json" "$evidence_cutoff" "$matures_at" \
                    "blob-unavailable"
                return
            fi
            have_file=true
        fi
    fi

    if [[ $have_file == true ]]; then
        current_lines=$(scan_functions "$file_content_file" | awk -F'\t' -v want="$header" '$1==want {print $2; exit}')
        [[ -z $current_lines ]] && current_lines=0
    else
        current_lines=0
    fi

    evidence=$(jq -cn \
        --argjson initialLines "$initial_lines" \
        --argjson currentLines "$current_lines" \
        --argjson thresholdLines "$threshold_lines" \
        --argjson targetLines "$target_lines" \
        '{valid: true, initialLines: $initialLines, currentLines: $currentLines, thresholdLines: $thresholdLines, targetLines: $targetLines}')
    value=$(printf '%s\n' "$evidence" | metric)
    diagnostics=$(case_diagnostics "$case_json" "$current_lines")
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
        --argjson diagnostics "$diagnostics" \
        '{value: $value, opportunityKey: $opportunityKey, case: $case,
          evidenceCutoff: $evidenceCutoff, maturesAt: $maturesAt,
          provenance: [{repository: $repository, kind: "git-commit", ref: $cutoffCommit},
                       {repository: $repository, kind: "go-source", ref: ($path + "@" + $cutoffCommit)}],
          diagnostics: $diagnostics}'
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
