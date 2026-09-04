#!/usr/bin/env bash

set -euo pipefail

export LC_ALL=C

REPOSITORY=github/gh-aw
WORKFLOW_NAME="Daily AstroStyleLite Markdown Spellcheck"
MATURATION_SECONDS=604800

tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/daily-astrostylelite-markdown-spellcheck-operational-value.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM

definition() {
    cat <<'JSON'
{
  "schemaVersion": 4,
  "grader": "operational-value",
  "repository": "github/gh-aw",
  "workflowName": "Daily AstroStyleLite Markdown Spellcheck",
  "sourcePath": ".github/workflows/daily-astrostylelite-markdown-spellcheck.md",
  "adoption": {
    "commit": "e4729b3d90afe948d7294dd6365efa8c01547137",
    "adoptedAt": "2026-04-20T05:42:44Z"
  },
  "operationalValue": "Remove the American-English spelling findings assigned from the run's AstroStyleLite Markdown snapshot.",
  "evidence": {
    "opportunity": "All cspell findings in docs/src/content/**/*.md and **/*.mdx at the run commit, including a zero-finding healthy snapshot.",
    "assignment": "Run cspell@8.19.4 with the workflow's optional-dictionary runtime configuration on the run commit. Key: spellcheck:<run SHA>; duplicate runs at the same SHA retain that key.",
    "accepted": "At the capped cutoff, cspell@8.19.4 no longer flags each assigned file-and-word finding in the default-branch snapshot.",
    "repositories": ["github/gh-aw"],
    "collection": "With contents:read, obtain commit archives and the default-branch commit at the cutoff, then run the adoption-pinned cspell configuration locally; PRs, traces, and agent output are excluded.",
    "maturation": "Seven days after run creation, after which the evidence cutoff is capped and observations are stable.",
    "zeroRule": "When assigned findings exist and every assigned finding remains flagged at the capped cutoff, score 0. A zero-finding assigned snapshot scores 1.",
    "missingRule": "Unavailable archives, cutoff commit, cspell executable or valid cspell output score null; unavailable evidence is never treated as zero."
  },
  "primaryMetric": {
    "id": "assigned-spelling-findings-resolved",
    "formula": "For a valid assignment with N findings, (N - remaining assigned file-and-word findings) / N; N=0 scores 1.",
    "direction": "higher_is_better"
  },
  "baseline": {
    "mode": "attainment-only",
    "value": null,
    "evidenceCutoff": null,
    "provenance": []
  },
  "validationExamples": {
    "targetAttained": {"valid": true, "initialFindings": 4, "remainingFindings": 0},
    "targetMissed": {"valid": true, "initialFindings": 4, "remainingFindings": 4},
    "missing": {"valid": false},
    "malformed": {"valid": true, "initialFindings": "4", "remainingFindings": 0}
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
      if (.valid != true)
        or (([.initialFindings, .remainingFindings] | all(.[]; type == "number" and floor == .)) | not)
        or (.initialFindings < 0) or (.remainingFindings < 0)
        or (.remainingFindings > .initialFindings)
      then null
      elif .initialFindings == 0 then 1
      else (.initialFindings - .remainingFindings) / .initialFindings
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
    command -v gh >/dev/null 2>&1 || return 1
    gh api "$@" 2>"$tmp_dir/gh-api-error"
}

archive_commit() {
    local repository=$1 commit_sha=$2 destination=$3 archive="$tmp_dir/archive.tar.gz"
    local root

    mkdir -p "$destination"
    if git cat-file -e "$commit_sha^{commit}" 2>/dev/null; then
        git archive "$commit_sha" | tar -xf - -C "$destination" || return 1
    else
        github_api -H "Accept: application/vnd.github+json" \
            "repos/$repository/tarball/$commit_sha" >"$archive" || return 1
        tar -xzf "$archive" -C "$destination" || return 1
        if [[ ! -d $destination/docs ]]; then
            root=$(find "$destination" -mindepth 1 -maxdepth 1 -type d -print -quit)
            [[ -n $root ]] || return 1
            printf '%s\n' "$root"
            return
        fi
    fi
    printf '%s\n' "$destination"
}

collect_findings() {
    local root=$1 output=$2 runtime_config="$tmp_dir/runtime-config.json"
    local files="$tmp_dir/files.txt" dictionary=''
    local candidate

    [[ -d $root/docs/src/content && -f $root/docs/.cspell.docs.json ]] || return 1
    for candidate in docs/.cspell-words.txt docs/.spellcheck-ignore.txt .cspell-words.txt \
        .spellcheck-ignore.txt .github/spellcheck-ignore.txt; do
        if [[ -f $root/$candidate ]]; then
            dictionary=$root/$candidate
            break
        fi
    done
    jq --arg dict "$dictionary" '
        .dictionaryDefinitions = (if $dict == "" then [] else [{name: "workflow-dictionary", path: $dict, addWords: true}] end)
        | .dictionaries = (if $dict == "" then ["en", "en-US"] else ["en", "en-US", "workflow-dictionary"] end)
    ' "$root/docs/.cspell.docs.json" >"$runtime_config" || return 1
    find "$root/docs/src/content" -type f \( -name '*.md' -o -name '*.mdx' \) | sort >"$files"
    if [[ ! -s $files ]]; then
        printf '[]\n' >"$output"
        return
    fi
    command -v npx >/dev/null 2>&1 || return 1
    npx --yes cspell@8.19.4 lint --no-progress --no-summary --show-suggestions \
        --reporter @cspell/cspell-json-reporter --config "$runtime_config" --file-list "$files" \
        >"$tmp_dir/cspell-results.json" || true
    jq -e . "$tmp_dir/cspell-results.json" >/dev/null || return 1
    jq -e '[.. | objects | select(has("error")) | .error[]?] | length == 0' \
        "$tmp_dir/cspell-results.json" >/dev/null || return 1
    jq -c --arg root "$root" '
        [.. | objects | select(has("issues")) | .issues[]? | select((.isFlagged // true) == true)
         | {path: ((.uri // .filename // .file // "") | sub("^file://"; "") | ltrimstr($root + "/")),
            word: (.text // .word // "")}
         | select(.path | test("^docs/src/content/.*\\.(md|mdx)$"))
         | select(.word | type == "string" and length > 0)]
    ' "$tmp_dir/cspell-results.json" >"$output"
}

assign_case() {
    local repository=$1 commit_sha=$2 root findings
    root=$(archive_commit "$repository" "$commit_sha" "$tmp_dir/assignment") || return 1
    findings="$tmp_dir/assignment-findings.json"
    collect_findings "$root" "$findings" || return 1
    jq -cn --arg subjectSha "$commit_sha" --slurpfile findings "$findings" \
        '{subjectSha: $subjectSha, findings: $findings[0]}'
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

emit_null() {
    local opportunity_key=$1 case_json=$2 evidence_cutoff=$3 matures_at=$4 reason=$5
    jq -cn --arg opportunityKey "$opportunity_key" --argjson case "$case_json" \
        --arg evidenceCutoff "$evidence_cutoff" --arg maturesAt "$matures_at" --arg reason "$reason" \
        '{value: null, opportunityKey: $opportunityKey, case: $case, evidenceCutoff: $evidenceCutoff,
          maturesAt: $maturesAt, provenance: [], diagnostics: {missingReason: $reason}}'
}

grade_run() {
    local request run_id repository workflow run_sha created_at evidence_at matures_at evidence_cutoff
    local case_json opportunity_key cutoff_commit root current_findings initial_findings remaining_findings evidence value

    request=$(cat)
    if ! printf '%s\n' "$request" | jq -e '
        .schemaVersion == 1 and (.run.id | type == "string" and length > 0)
        and (.run.repository | type == "string") and (.run.workflow | type == "string")
        and (.run.sha | type == "string" and test("^[0-9a-f]{40}$"))
        and (.run.createdAt | type == "string") and (.evidenceAt | type == "string")
        and (.case == null or (.case | type == "object"))' >/dev/null 2>&1; then
        emit_null "invalid-request" '{"invalidRequest":true}' "1970-01-01T00:00:00Z" "1970-01-01T00:00:00Z" "invalid request"
        return
    fi
    run_id=$(printf '%s\n' "$request" | jq -r '.run.id')
    repository=$(printf '%s\n' "$request" | jq -r '.run.repository')
    workflow=$(printf '%s\n' "$request" | jq -r '.run.workflow')
    run_sha=$(printf '%s\n' "$request" | jq -r '.run.sha')
    created_at=$(normalize_timestamp "$(printf '%s\n' "$request" | jq -r '.run.createdAt')") \
        || { emit_null "run:$run_id" '{"invalidTimestamp":true}' "1970-01-01T00:00:00Z" "1970-01-01T00:00:00Z" "invalid timestamp"; return; }
    evidence_at=$(normalize_timestamp "$(printf '%s\n' "$request" | jq -r '.evidenceAt')") \
        || { emit_null "run:$run_id" '{"invalidTimestamp":true}' "1970-01-01T00:00:00Z" "1970-01-01T00:00:00Z" "invalid timestamp"; return; }
    matures_at=$(add_seconds "$created_at" "$MATURATION_SECONDS")
    if (( $(timestamp_epoch "$evidence_at") < $(timestamp_epoch "$matures_at") )); then
        evidence_cutoff=$evidence_at
    else
        evidence_cutoff=$matures_at
    fi
    if [[ $repository != "$REPOSITORY" || $workflow != "$WORKFLOW_NAME" ]]; then
        emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" "run repository or workflow does not match the frozen contract"
        return
    fi
    case_json=$(printf '%s\n' "$request" | jq -c '.case')
    if [[ $case_json == null ]]; then
        case_json=$(assign_case "$repository" "$run_sha") \
            || { emit_null "spellcheck:$run_sha" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" "assignment-unavailable"; return; }
    elif ! printf '%s\n' "$case_json" | jq -e --arg sha "$run_sha" '
        .subjectSha == $sha and (.findings | type == "array")
        and all(.findings[]; (.path | type == "string" and test("^docs/src/content/.*\\.(md|mdx)$"))
            and (.word | type == "string" and length > 0))' >/dev/null; then
        emit_null "spellcheck:$run_sha" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" "invalid-case"
        return
    fi
    opportunity_key="spellcheck:$run_sha"
    cutoff_commit=$(latest_commit_at_cutoff "$repository" "$evidence_cutoff") \
        || { emit_null "$opportunity_key" "$case_json" "$evidence_cutoff" "$matures_at" "cutoff-commit-unavailable"; return; }
    root=$(archive_commit "$repository" "$cutoff_commit" "$tmp_dir/cutoff") \
        || { emit_null "$opportunity_key" "$case_json" "$evidence_cutoff" "$matures_at" "cutoff-archive-unavailable"; return; }
    current_findings="$tmp_dir/cutoff-findings.json"
    collect_findings "$root" "$current_findings" \
        || { emit_null "$opportunity_key" "$case_json" "$evidence_cutoff" "$matures_at" "cutoff-spellcheck-unavailable"; return; }
    initial_findings=$(printf '%s\n' "$case_json" | jq '.findings | length')
    remaining_findings=$(jq --slurpfile current "$current_findings" '
        [.findings[] | (.path + "\u0000" + .word) as $finding
         | select(($current[0] | map(.path + "\u0000" + .word) | index($finding)) != null)] | length
    ' <<<"$case_json")
    evidence=$(jq -cn --argjson initialFindings "$initial_findings" --argjson remainingFindings "$remaining_findings" \
        '{valid: true, initialFindings: $initialFindings, remainingFindings: $remainingFindings}')
    value=$(printf '%s\n' "$evidence" | metric)
    jq -cn --argjson value "$value" --arg opportunityKey "$opportunity_key" --argjson case "$case_json" \
        --arg evidenceCutoff "$evidence_cutoff" --arg maturesAt "$matures_at" --arg repository "$repository" \
        --arg assignmentSha "$run_sha" --arg cutoffCommit "$cutoff_commit" \
        '{value: $value, opportunityKey: $opportunityKey, case: $case, evidenceCutoff: $evidenceCutoff,
          maturesAt: $maturesAt, provenance: [{repository: $repository, kind: "git-commit", ref: $assignmentSha},
          {repository: $repository, kind: "git-commit", ref: $cutoffCommit}], diagnostics: {}}'
}

case ${1:-} in
    --definition) definition ;;
    --metric) metric ;;
    --grade-run) grade_run ;;
    *) printf 'usage: %s --definition|--metric|--grade-run\n' "$0" >&2; exit 1 ;;
esac
