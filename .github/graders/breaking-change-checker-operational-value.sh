#!/usr/bin/env bash

set -euo pipefail

export LC_ALL=C

REPOSITORY=github/gh-aw
WORKFLOW_NAME="Breaking Change Checker"
WINDOW_SECONDS=86400
MATURATION_SECONDS=172800
CLI_PATHS=(cmd pkg/cli pkg/workflow pkg/parser/schemas)

tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/breaking-change-checker-operational-value.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM

local_repo_root=''
if command -v git >/dev/null 2>&1; then
    local_repo_root=$(git rev-parse --show-toplevel 2>/dev/null || true)
fi

definition() {
    cat <<'JSON'
{
  "schemaVersion": 4, "grader": "operational-value",
  "repository": "github/gh-aw", "workflowName": "Breaking Change Checker",
  "sourcePath": ".github/workflows/breaking-change-checker.md",
  "adoption": {"commit": "98d551bad739f8bf2eb43a1d5158d73e0bb24e98", "adoptedAt": "2025-11-27T11:54:52Z"},
  "operationalValue": "Repository-recorded compliance with the CONTRIBUTING.md CLI-breaking-change rule: a self-declared major changeset landing alongside a CLI-path change in the run's 24-hour analysis window includes actual migration guidance.",
  "evidence": {
    "opportunity": "The single earliest commit reachable from the run commit, dated within the 24 hours before run.createdAt, that both touches cmd/**, pkg/cli/**, pkg/workflow/**, or pkg/parser/schemas/** and adds or modifies a .changeset/*.md file whose frontmatter declares \"gh-aw\": major -- mirroring this workflow's own adoption-time analysis window and focus paths.",
    "assignment": "Earliest qualifying commit by commit date, ties broken by lexically-smallest commit SHA. Key: changeset:<path>@<commitSha> or repository-health:no-major-changeset-in-window when no commit in the window pairs a CLI-path change with a major changeset; duplicates repeat.",
    "accepted": "Git evidence only: the assigned changeset file's content at the evidence-cutoff commit (or, if the file was since removed by a version-bump release, its last known content at the assignment commit) contains migration-guidance text; issue comments, PR review threads, and agent traces are excluded.",
    "repositories": ["github/gh-aw"],
    "collection": "With contents:read, scan commit history for the window with git/commits metadata, then read the assigned changeset file's blob content at the evidence-cutoff commit.",
    "maturation": "Two days, matching this workflow's own `expires: \"2d\"` create-issue window (shared/daily-issue-base.md), since a major changeset's content is fixed at merge and any follow-up correction is expected before the workflow's own issue would otherwise expire.",
    "zeroRule": "A self-declared major changeset present in the window without any migration-guidance text scores 0.",
    "missingRule": "Invalid, unavailable, or malformed changeset evidence scores null; a window with no self-declared major changeset touching the focus paths is healthy and scores 1."
  },
  "primaryMetric": {"id": "assigned-major-changeset-migration-guidance", "formula": "repository-health case => 1; changeset case => 1 if the assigned changeset's accepted content contains migration-guidance text, else 0; null when evidence is invalid or unavailable.", "direction": "higher_is_better"},
  "diagnosticMetrics": [
    {"id": "migration-guidance-present-at-merge", "name": "Migration guidance present at merge time", "formula": "1 if the assigned changeset already contained migration-guidance text in the commit that introduced it, else 0, evaluated once at assignment.", "direction": "higher_is_better", "aggregation": "latest"}
  ],
  "baseline": {"mode": "attainment-only", "value": null, "evidenceCutoff": null, "provenance": []},
  "validationExamples": {
    "targetAttained": {"valid":true,"kind":"changeset","hasMigrationGuidance":true},
    "targetMissed": {"valid":true,"kind":"changeset","hasMigrationGuidance":false},
    "missing": {"valid":false},
    "malformed": {"valid":"yes","kind":"changeset","hasMigrationGuidance":"yes"}
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
      if .valid != true or (.kind != "changeset" and .kind != "repository-health") then null
      elif .kind == "repository-health" then 1
      elif (.hasMigrationGuidance | type) != "boolean" then null
      elif .hasMigrationGuidance then 1
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

has_migration_guidance() {
    grep -qi 'migrat' "$1" 2>/dev/null
}

is_major_changeset() {
    head -5 "$1" 2>/dev/null | grep -q '"gh-aw"[[:space:]]*:[[:space:]]*major'
}

# Finds the earliest commit reachable from $2, dated within ($3, $4], that
# touches one of CLI_PATHS and adds/modifies a .changeset/*.md file declaring
# a major bump. Prints "commitSha\tpath" on success.
find_window_commit_local() {
    local commit_sha=$1 window_start=$2 window_end=$3
    local sha changeset_file content_file="$tmp_dir/candidate-changeset.md"
    local -a candidates=()

    while IFS= read -r sha; do
        [[ -n $sha ]] && candidates+=("$sha")
    done < <(git -C "$local_repo_root" log \
        --since="$window_start" --until="$window_end" --format='%H' "$commit_sha" -- \
        "${CLI_PATHS[@]}" 2>/dev/null | tac)

    if [[ ${#candidates[@]} -gt 0 ]]; then
        for sha in "${candidates[@]}"; do
            [[ -n $sha ]] || continue
            while IFS= read -r changeset_file; do
                [[ -n $changeset_file ]] || continue
                git -C "$local_repo_root" show "$sha:$changeset_file" >"$content_file" 2>/dev/null || continue
                if is_major_changeset "$content_file"; then
                    printf '%s\t%s\n' "$sha" "$changeset_file"
                    return 0
                fi
            done < <(git -C "$local_repo_root" show --name-only --diff-filter=AM --format='' "$sha" -- '.changeset/*.md' 2>/dev/null)
        done
    fi
    return 1
}

commit_touches_cli_paths() {
    local files_json=$1
    printf '%s\n' "$files_json" | jq -e '
        any(.[]; .filename as $f
            | (["cmd/", "pkg/cli/", "pkg/workflow/", "pkg/parser/schemas/"] | any(. as $p | $f | startswith($p))))
    ' >/dev/null 2>&1
}

find_window_commit_api() {
    local repository=$1 commit_sha=$2 window_start=$3 window_end=$4
    local commits_json sha changeset_file content_file="$tmp_dir/candidate-changeset.md"
    local files_json page

    for page in 1 2 3; do
        commits_json=$(github_api -X GET "repos/$repository/commits" \
            -f sha="$commit_sha" -f since="$window_start" -f until="$window_end" \
            -f per_page=100 -F page="$page") || return 1
        [[ $(printf '%s\n' "$commits_json" | jq 'length') -gt 0 ]] || break

        while IFS= read -r sha; do
            [[ -n $sha ]] || continue
            files_json=$(github_api "repos/$repository/commits/$sha" | jq -c '.files // []') || continue
            commit_touches_cli_paths "$files_json" || continue
            changeset_file=$(printf '%s\n' "$files_json" | jq -r '
                .[] | select((.status == "added" or .status == "modified")
                    and (.filename | test("^\\.changeset/.*\\.md$"))) | .filename' | head -1)
            [[ -n $changeset_file ]] || continue
            if github_api -H "Accept: application/vnd.github.raw+json" \
                "repos/$repository/contents/$changeset_file?ref=$sha" >"$content_file" 2>/dev/null \
                && is_major_changeset "$content_file"; then
                printf '%s\t%s\n' "$sha" "$changeset_file"
                return 0
            fi
        done < <(printf '%s\n' "$commits_json" | jq -r 'reverse[].sha')
    done
    return 1
}

assign_case() {
    local repository=$1 commit_sha=$2 created_at=$3
    local window_start result sha path

    window_start=$(add_seconds "$created_at" "-$WINDOW_SECONDS") || return 1

    if [[ -n $local_repo_root ]] \
        && git -C "$local_repo_root" cat-file -e "$commit_sha^{commit}" 2>/dev/null; then
        if result=$(find_window_commit_local "$commit_sha" "$window_start" "$created_at"); then
            sha=$(printf '%s\n' "$result" | cut -f1)
            path=$(printf '%s\n' "$result" | cut -f2)
            jq -cn --arg path "$path" --arg commitSha "$sha" \
                --arg windowStart "$window_start" --arg windowEnd "$created_at" \
                '{kind: "changeset", path: $path, commitSha: $commitSha, windowStart: $windowStart, windowEnd: $windowEnd}'
            return 0
        fi
    else
        if result=$(find_window_commit_api "$repository" "$commit_sha" "$window_start" "$created_at"); then
            sha=$(printf '%s\n' "$result" | cut -f1)
            path=$(printf '%s\n' "$result" | cut -f2)
            jq -cn --arg path "$path" --arg commitSha "$sha" \
                --arg windowStart "$window_start" --arg windowEnd "$created_at" \
                '{kind: "changeset", path: $path, commitSha: $commitSha, windowStart: $windowStart, windowEnd: $windowEnd}'
            return 0
        fi
    fi

    jq -cn --arg windowStart "$window_start" --arg windowEnd "$created_at" \
        '{kind: "repository-health", windowStart: $windowStart, windowEnd: $windowEnd}'
}

get_file_at() {
    local repository=$1 commit_sha=$2 path=$3 output=$4

    if [[ -n $local_repo_root ]] \
        && git -C "$local_repo_root" cat-file -e "$commit_sha^{commit}" 2>/dev/null; then
        if git -C "$local_repo_root" cat-file -e "$commit_sha:$path" 2>/dev/null; then
            git -C "$local_repo_root" show "$commit_sha:$path" >"$output"
            return 0
        fi
        return 1
    fi

    if tree_json=$(github_api "repos/$repository/git/trees/$commit_sha?recursive=1") \
        && printf '%s\n' "$tree_json" | jq -e '.truncated == false and (.tree | type == "array")' >/dev/null 2>&1; then
        blob_sha=$(printf '%s\n' "$tree_json" | jq -r --arg path "$path" \
            '.tree[] | select(.type == "blob" and .path == $path) | .sha')
        if [[ -n $blob_sha ]]; then
            github_api -H "Accept: application/vnd.github.raw+json" \
                "repos/$repository/git/blobs/$blob_sha" >"$output"
            return 0
        fi
        return 1
    fi
    return 1
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
    local case_json kind path commit_sha window_start window_end
    local opportunity_key evidence value diagnostics migration_at_merge
    local cutoff_commit content_file="$tmp_dir/cutoff-file" merge_file="$tmp_dir/merge-file"
    local has_guidance

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
        if ! case_json=$(assign_case "$repository" "$run_sha" "$created_at"); then
            emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" \
                "assignment-unavailable"
            return
        fi
    elif ! printf '%s\n' "$case_json" | jq -e '
            (.kind == "repository-health"
                or (.kind == "changeset"
                    and (.path | type == "string" and test("^\\.changeset/.*\\.md$"))
                    and (.commitSha | type == "string" and test("^[0-9a-f]{40}$"))))
            and (.windowStart | type == "string" and length > 0)
            and (.windowEnd | type == "string" and length > 0)
        ' >/dev/null; then
        emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" \
            "invalid-case"
        return
    fi

    kind=$(printf '%s\n' "$case_json" | jq -r '.kind')
    window_start=$(printf '%s\n' "$case_json" | jq -r '.windowStart')
    window_end=$(printf '%s\n' "$case_json" | jq -r '.windowEnd')

    if [[ $kind == "repository-health" ]]; then
        opportunity_key="repository-health:no-major-changeset-in-window"
        evidence='{"valid":true,"kind":"repository-health"}'
        value=$(printf '%s\n' "$evidence" | metric)
        jq -cn \
            --argjson value "$value" \
            --arg opportunityKey "$opportunity_key" \
            --argjson case "$case_json" \
            --arg evidenceCutoff "$evidence_cutoff" \
            --arg maturesAt "$matures_at" \
            --arg repository "$repository" \
            --arg windowStart "$window_start" --arg windowEnd "$window_end" \
            '{value: $value, opportunityKey: $opportunityKey, case: $case,
              evidenceCutoff: $evidenceCutoff, maturesAt: $maturesAt,
              provenance: [{repository: $repository, kind: "git-log-window", ref: ($windowStart + ".." + $windowEnd)}],
              diagnostics: {repositoryHealthyInWindow: true}}'
        return
    fi

    path=$(printf '%s\n' "$case_json" | jq -r '.path')
    commit_sha=$(printf '%s\n' "$case_json" | jq -r '.commitSha')
    opportunity_key="changeset:$path@$commit_sha"

    if get_file_at "$repository" "$commit_sha" "$path" "$merge_file"; then
        if has_migration_guidance "$merge_file"; then
            migration_at_merge=1
        else
            migration_at_merge=0
        fi
    else
        emit_null "$opportunity_key" "$case_json" "$evidence_cutoff" "$matures_at" \
            "assignment-commit-unavailable"
        return
    fi

    if ! cutoff_commit=$(latest_commit_at_cutoff "$repository" "$evidence_cutoff"); then
        emit_null "$opportunity_key" "$case_json" "$evidence_cutoff" "$matures_at" \
            "cutoff-commit-unavailable"
        return
    fi

    if get_file_at "$repository" "$cutoff_commit" "$path" "$content_file"; then
        if has_migration_guidance "$content_file"; then
            has_guidance=true
        else
            has_guidance=false
        fi
    else
        # The changeset file no longer exists at the cutoff commit (typically
        # consumed by a version-bump release). Fall back to its last known
        # content at the assignment commit rather than treating removal as a
        # regression or an improvement.
        has_guidance=$([[ $migration_at_merge -eq 1 ]] && printf 'true' || printf 'false')
    fi

    evidence=$(jq -cn --argjson hasMigrationGuidance "$has_guidance" \
        '{valid: true, kind: "changeset", hasMigrationGuidance: $hasMigrationGuidance}')
    value=$(printf '%s\n' "$evidence" | metric)
    diagnostics=$(jq -cn --argjson migrationAtMerge "$migration_at_merge" \
        '{"migration-guidance-present-at-merge": $migrationAtMerge}')
    jq -cn \
        --argjson value "$value" \
        --arg opportunityKey "$opportunity_key" \
        --argjson case "$case_json" \
        --arg evidenceCutoff "$evidence_cutoff" \
        --arg maturesAt "$matures_at" \
        --arg repository "$repository" \
        --arg commitSha "$commit_sha" \
        --arg cutoffCommit "$cutoff_commit" \
        --arg path "$path" \
        --argjson diagnostics "$diagnostics" \
        '{value: $value, opportunityKey: $opportunityKey, case: $case,
          evidenceCutoff: $evidenceCutoff, maturesAt: $maturesAt,
          provenance: [{repository: $repository, kind: "git-commit", ref: $commitSha},
                       {repository: $repository, kind: "changeset-source", ref: ($path + "@" + $cutoffCommit)}],
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
