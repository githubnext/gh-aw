#!/usr/bin/env bash

set -euo pipefail

export LC_ALL=C

REPOSITORY=github/gh-aw
WORKFLOW_NAME="Daily Ambient Context Optimizer"
OWN_WORKFLOW_FILE="daily-ambient-context-optimizer.md"
THRESHOLD_CHARS=20000
TARGET_CHARS=10000
MATURATION_SECONDS=259200

tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/daily-ambient-context-optimizer-operational-value.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM

local_repo_root=''
if command -v git >/dev/null 2>&1; then
    local_repo_root=$(git rev-parse --show-toplevel 2>/dev/null || true)
fi

definition() {
    cat <<'JSON'
{
  "schemaVersion": 4, "grader": "operational-value",
  "repository": "github/gh-aw", "workflowName": "Daily Ambient Context Optimizer",
  "sourcePath": ".github/workflows/daily-ambient-context-optimizer.md",
  "adoption": {"commit": "89383a71adbad150ef5b80ebe4b1342370bdf91b", "adoptedAt": "2026-06-03T13:28:03Z"},
  "operationalValue": "Shrink the run's largest ambient-context workflow markdown file toward a 10000-character healthy size.",
  "evidence": {
    "opportunity": "Largest top-level .github/workflows/*.md file (excluding this optimizer and shared/ fragments) at the run commit; below 20000 characters is healthy.",
    "assignment": "Greatest wc -c over frontmatter+body, reverse lexical tie-break. Key: workflow-md:<path> or repository-health:workflows-under-20000; duplicates repeat across days until resolved.",
    "accepted": "Git evidence of assigned-path character reduction toward 10000 or tree-proven absence (file removed/merged away); issues and traces are excluded.",
    "repositories": ["github/gh-aw"],
    "collection": "With contents:read, count bytes in the run commit archive for assignment and in the cutoff commit blob for evidence.",
    "maturation": "Three days; PR #36992 (a same-day implementation of this workflow's first recommendation issue) was created 2026-06-04T20:43Z and merged 2026-06-04T22:33Z, and follow-on trims (#43619, #43910, #40874, #171847) similarly landed within a day or two of the recommending run.",
    "zeroRule": "No reduction from the initial oversized file scores 0.",
    "missingRule": "Invalid, unavailable, or truncated Git evidence scores null; tree-proven path absence is attainment."
  },
  "primaryMetric": {"id": "assigned-workflow-md-shrink", "formula": "initialChars < 20000 => 1; else clamp((initialChars-currentChars)/(initialChars-10000),0,1). Proven absence sets currentChars=0.", "direction": "higher_is_better"},
  "diagnosticMetrics": [
    {"id": "largest-workflow-health", "name": "Largest-workflow health at assignment", "formula": "min(1, 19999 / initialChars) when the assignment archive contains eligible files.", "direction": "higher_is_better", "aggregation": "latest"},
    {"id": "compliant-char-mass-share", "name": "Compliant char-mass share at assignment", "formula": "compliantChars / totalChars when the assignment archive contains eligible files and positive char mass.", "direction": "higher_is_better", "aggregation": "latest"}
  ],
  "baseline": {"mode": "attainment-only", "value": null, "evidenceCutoff": null, "provenance": []},
  "validationExamples": {
    "targetAttained": {"valid":true,"initialChars":27299,"currentChars":11876,"thresholdChars":20000,"targetChars":10000},
    "targetMissed": {"valid":true,"initialChars":27299,"currentChars":27299,"thresholdChars":20000,"targetChars":10000},
    "missing": {"valid":false},
    "malformed": {"valid":"yes","initialChars":"16546"}
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
      if .valid != true or ([.initialChars,.currentChars,.thresholdChars,.targetChars]|all(.[];type=="number")|not)
        or .initialChars<0 or .currentChars<0 or .targetChars<0 or .thresholdChars<=.targetChars then null
      elif .initialChars<.thresholdChars then 1
      elif .initialChars<=.targetChars then null
      else ((.initialChars-.currentChars)/(.initialChars-.targetChars)) as $v
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

blob_char_count() {
    local repository=$1 blob_sha=$2 output=$3

    if ! github_api -H "Accept: application/vnd.github.raw+json" \
        "repos/$repository/git/blobs/$blob_sha" >"$output"; then
        return 1
    fi
    wc -c <"$output" | tr -d ' '
}

load_tree() {
    local repository=$1 commit_sha=$2 output=$3

    if ! github_api "repos/$repository/git/trees/$commit_sha?recursive=1" >"$output"; then
        return 1
    fi
    jq -e '.truncated == false and (.tree | type == "array")' "$output" >/dev/null
}

assign_case() {
    local repository=$1 commit_sha=$2
    local archive_file="$tmp_dir/assignment-archive.tar.gz"
    local extract_dir="$tmp_dir/assignment-archive"
    local root_dir path chars
    local largest_path='' largest_chars=-1
    local eligible_file_count=0 total_chars=0 compliant_chars=0

    mkdir -p "$extract_dir"
    if [[ -n $local_repo_root ]] \
        && git -C "$local_repo_root" cat-file -e "$commit_sha^{commit}" 2>/dev/null; then
        git -C "$local_repo_root" archive "$commit_sha" -- .github/workflows | tar -xf - -C "$extract_dir" || return 1
        root_dir=$extract_dir
    else
        github_api -H "Accept: application/vnd.github+json" \
            "repos/$repository/tarball/$commit_sha" >"$archive_file" || return 1
        tar -xzf "$archive_file" -C "$extract_dir" || return 1
        root_dir=$(find "$extract_dir" -mindepth 1 -maxdepth 1 -type d -print -quit)
    fi
    [[ -n $root_dir && -d $root_dir/.github/workflows ]] || return 1

    while IFS= read -r -d '' path; do
        chars=$(wc -c <"$root_dir/$path" | tr -d ' ') || return 1
        eligible_file_count=$((eligible_file_count + 1))
        total_chars=$((total_chars + chars))
        if (( chars < THRESHOLD_CHARS )); then
            compliant_chars=$((compliant_chars + chars))
        fi
        if (( chars > largest_chars )) \
            || { (( chars == largest_chars )) && [[ $path > $largest_path ]]; }; then
            largest_path=$path
            largest_chars=$chars
        fi
    done < <(cd "$root_dir" && find .github/workflows -maxdepth 1 -type f -name '*.md' \
        ! -name "$OWN_WORKFLOW_FILE" -print0)

    [[ -n $largest_path ]] || return 1

    jq -cn \
        --arg path "$largest_path" \
        --argjson initialChars "$largest_chars" \
        --argjson thresholdChars "$THRESHOLD_CHARS" \
        --argjson targetChars "$TARGET_CHARS" \
        --argjson eligibleFileCount "$eligible_file_count" \
        --argjson totalChars "$total_chars" \
        --argjson compliantChars "$compliant_chars" \
        --arg subjectSha "$commit_sha" \
        '{path: $path, initialChars: $initialChars, thresholdChars: $thresholdChars,
          targetChars: $targetChars, subjectSha: $subjectSha,
          eligibleFileCount: $eligibleFileCount, totalChars: $totalChars,
          compliantChars: $compliantChars}'
}

case_diagnostics() {
    local case_json=$1 current_chars=$2

    printf '%s\n' "$case_json" | jq -c \
        --argjson currentChars "$current_chars" \
        --argjson healthyLimit "$((THRESHOLD_CHARS - 1))" '
        {
            "largest-workflow-health":
                (if (.initialChars | type) == "number" and .initialChars > 0
                 then ([1, ($healthyLimit / .initialChars)] | min) else null end),
            "compliant-char-mass-share":
                (if (.eligibleFileCount | type) == "number" and .eligibleFileCount > 0
                    and (.totalChars | type) == "number" and .totalChars > 0
                    and (.compliantChars | type) == "number"
                 then (.compliantChars / .totalChars) else null end),
            currentChars: $currentChars
        }'
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
    local case_json path initial_chars opportunity_key evidence value diagnostics
    local cutoff_commit blob_sha current_chars
    local tree_file="$tmp_dir/cutoff-tree.json"
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
        --argjson threshold "$THRESHOLD_CHARS" \
        --argjson target "$TARGET_CHARS" '
            (.path | type == "string" and test("^\\.github/workflows/[^/]+\\.md$"))
            and (.initialChars | type == "number" and . >= 0 and floor == .)
            and .thresholdChars == $threshold
            and .targetChars == $target
            and (.subjectSha | type == "string" and test("^[0-9a-f]{40}$"))
            and ((has("eligibleFileCount") | not) or (.eligibleFileCount | type == "number" and . >= 0 and floor == .))
            and ((has("totalChars") | not) or (.totalChars | type == "number" and . >= 0 and floor == .))
            and ((has("compliantChars") | not) or (.compliantChars | type == "number" and . >= 0 and floor == .))
        ' >/dev/null; then
        emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" \
            "invalid-case"
        return
    fi

    path=$(printf '%s\n' "$case_json" | jq -r '.path')
    initial_chars=$(printf '%s\n' "$case_json" | jq -r '.initialChars')
    if (( initial_chars < THRESHOLD_CHARS )); then
        opportunity_key="repository-health:workflows-under-20000"
        evidence=$(jq -cn \
            --argjson initialChars "$initial_chars" \
            --argjson thresholdChars "$THRESHOLD_CHARS" \
            --argjson targetChars "$TARGET_CHARS" \
            '{valid: true, initialChars: $initialChars, currentChars: $initialChars, thresholdChars: $thresholdChars, targetChars: $targetChars}')
        value=$(printf '%s\n' "$evidence" | metric)
        diagnostics=$(case_diagnostics "$case_json" "$initial_chars")
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

    opportunity_key="workflow-md:$path"
    if ! cutoff_commit=$(latest_commit_at_cutoff "$repository" "$evidence_cutoff"); then
        emit_null "$opportunity_key" "$case_json" "$evidence_cutoff" "$matures_at" \
            "cutoff-commit-unavailable"
        return
    fi
    if [[ -n $local_repo_root ]] \
        && git -C "$local_repo_root" cat-file -e "$cutoff_commit^{commit}" 2>/dev/null; then
        if git -C "$local_repo_root" cat-file -e "$cutoff_commit:$path" 2>/dev/null; then
            current_chars=$(git -C "$local_repo_root" show "$cutoff_commit:$path" | wc -c | tr -d ' ') || return 1
        else
            current_chars=0
        fi
    else
        if ! load_tree "$repository" "$cutoff_commit" "$tree_file"; then
            emit_null "$opportunity_key" "$case_json" "$evidence_cutoff" "$matures_at" \
                "cutoff-tree-unavailable"
            return
        fi

        blob_sha=$(jq -r --arg path "$path" \
            '.tree[] | select(.type == "blob" and .path == $path) | .sha' "$tree_file")
        if [[ -z $blob_sha ]]; then
            current_chars=0
        elif ! current_chars=$(blob_char_count "$repository" "$blob_sha" "$blob_file"); then
            emit_null "$opportunity_key" "$case_json" "$evidence_cutoff" "$matures_at" \
                "blob-unavailable"
            return
        fi
    fi

    evidence=$(jq -cn \
        --argjson initialChars "$initial_chars" \
        --argjson currentChars "$current_chars" \
        --argjson thresholdChars "$THRESHOLD_CHARS" \
        --argjson targetChars "$TARGET_CHARS" \
        '{valid: true, initialChars: $initialChars, currentChars: $currentChars, thresholdChars: $thresholdChars, targetChars: $targetChars}')
    value=$(printf '%s\n' "$evidence" | metric)
    diagnostics=$(case_diagnostics "$case_json" "$current_chars")
    jq -cn \
        --argjson value "$value" \
        --arg opportunityKey "$opportunity_key" \
        --argjson case "$case_json" \
        --arg evidenceCutoff "$evidence_cutoff" \
        --arg maturesAt "$matures_at" \
        --arg repository "$repository" \
        --arg cutoffCommit "$cutoff_commit" \
        --arg path "$path" \
        --argjson currentChars "$current_chars" \
        --argjson diagnostics "$diagnostics" \
        '{value: $value, opportunityKey: $opportunityKey, case: $case,
          evidenceCutoff: $evidenceCutoff, maturesAt: $maturesAt,
          provenance: [{repository: $repository, kind: "git-commit", ref: $cutoffCommit},
                       {repository: $repository, kind: "workflow-markdown", ref: ($path + "@" + $cutoffCommit)}],
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
