#!/usr/bin/env bash

set -euo pipefail

export LC_ALL=C

REPOSITORY=github/gh-aw
WORKFLOW_NAME="Changeset Generator"
MATURATION_SECONDS=86400

tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/changeset-operational-value.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM

definition() {
    cat <<'JSON'
{
  "schemaVersion": 4,
  "grader": "operational-value",
  "repository": "github/gh-aw",
  "workflowName": "Changeset Generator",
  "sourcePath": ".github/workflows/changeset.md",
  "adoption": {
    "commit": "06d7770e80228207b6436fd22cd01ce145daf71c",
    "adoptedAt": "2025-11-01T16:31:43Z"
  },
  "operationalValue": "For a pull request assigned by the changeset label trigger, attain a new valid .changeset/*.md file on the pull request branch.",
  "evidence": {
    "opportunity": "A pull request run activated by the changeset or smoke label.",
    "assignment": "Use the supplied pull-request case or event number; when both are absent, recover the pull request from the historical workflow-run record. Key: pull-request:<number>; duplicate runs retain the key and reruns retain the run ID subject.",
    "accepted": "At the capped evidence cutoff, the pull request's latest commit has a new non-README .changeset/*.md file whose first three lines are valid changeset frontmatter and whose body is non-empty.",
    "repositories": ["github/gh-aw"],
    "collection": "Read the workflow run, pull-request commits, commit contents, and raw changeset files through the GitHub API with actions, contents, and pull-requests read access.",
    "maturation": "One day after run creation, allowing the safe-output commit to reach the pull-request branch; evidenceCutoff is the earlier of evidenceAt and maturesAt.",
    "zeroRule": "An eligible assigned pull request with no new valid changeset by the cutoff scores 0.",
    "missingRule": "Unavailable run, pull-request, commit, tree, or file evidence scores null; an empty or malformed changeset is evidence of non-attainment and scores 0."
  },
  "primaryMetric": {
    "id": "new-valid-changeset",
    "formula": "1 when at least one new valid changeset exists at the cutoff, otherwise 0",
    "direction": "higher_is_better"
  },
  "diagnosticMetrics": [
    {
      "id": "new-valid-changeset-count",
      "name": "New valid changeset count",
      "formula": "Count of new valid non-README changeset files at the cutoff",
      "direction": "higher_is_better",
      "aggregation": "latest"
    }
  ],
  "baseline": {
    "mode": "attainment-only",
    "value": null,
    "evidenceCutoff": null,
    "provenance": []
  },
  "validationExamples": {
    "targetAttained": {"valid": true, "newValidCount": 1},
    "targetMissed": {"valid": true, "newValidCount": 0},
    "missing": {"valid": false},
    "malformed": {"valid": "yes", "newValidCount": 1}
  }
}
JSON
}

metric() {
    jq '
      if .valid != true or (.newValidCount | type) != "number"
        or (.newValidCount < 0) or (.newValidCount | floor != .) then null
      elif .newValidCount > 0 then 1
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
        end' 2>/dev/null
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

valid_changeset() {
    awk '
      NR == 1 && $0 != "---" { exit 1 }
      NR == 2 && $0 !~ /^"gh-aw":[[:space:]]+(patch|minor|major)[[:space:]]*$/ { exit 1 }
      NR == 3 && $0 != "---" { exit 1 }
      NR > 3 && $0 !~ /^[[:space:]]*$/ { body = 1 }
      END { exit !(NR >= 4 && body) }
    ' "$1"
}

contents_names() {
    local repository=$1 commit_sha=$2 output=$3
    github_api "repos/$repository/contents/.changeset?ref=$commit_sha" >"$output"
    jq -e 'type == "array"' "$output" >/dev/null
}

count_new_valid_changesets() {
    local repository=$1 initial_sha=$2 cutoff_sha=$3
    local initial_file="$tmp_dir/initial-contents.json"
    local cutoff_file="$tmp_dir/cutoff-contents.json"
    local name encoded raw count=0

    contents_names "$repository" "$initial_sha" "$initial_file" || return 1
    contents_names "$repository" "$cutoff_sha" "$cutoff_file" || return 1
    while IFS= read -r name; do
        [[ $name != README ]] || continue
        if jq -e --arg name "$name" \
            'any(.[]; .type == "file" and .name == $name)' "$initial_file" >/dev/null; then
            continue
        fi
        encoded=$(github_api "repos/$repository/contents/.changeset/$name?ref=$cutoff_sha" \
            | jq -er '.encoding == "base64" and .content')
        raw="$tmp_dir/changeset"
        printf '%s' "$encoded" | tr -d '\n' | base64 --decode >"$raw" || return 1
        if valid_changeset "$raw"; then
            count=$((count + 1))
        fi
    done < <(jq -r '.[] | select(.type == "file" and (.name | endswith(".md"))) | .name' "$cutoff_file")
    printf '%s\n' "$count"
}

latest_commit_at_cutoff() {
    local repository=$1 pull_request=$2 cutoff=$3 commits_file
    commits_file="$tmp_dir/commits.json"
    github_api --paginate --slurp "repos/$repository/pulls/$pull_request/commits?per_page=100" >"$commits_file"
    jq -er --arg cutoff "$cutoff" '
      add
      | map(select((.commit.committer.date // .commit.author.date) <= $cutoff))
      | sort_by(.commit.committer.date // .commit.author.date)
      | last
      | .sha
      | select(type == "string" and test("^[0-9a-f]{40}$"))
    ' "$commits_file"
}

recover_pull_request() {
    local request=$1 run_id=$2 repository=$3 number
    number=$(printf '%s\n' "$request" | jq -r '
      if (.case | type) == "object" and (.case.pullRequest | type) == "number"
      then .case.pullRequest
      elif (.event | type) == "object" and (.event.pull_request.number | type) == "number"
      then .event.pull_request.number
      else empty end')
    if [[ -n $number ]]; then
        printf '%s\n' "$number"
        return
    fi
    github_api "repos/$repository/actions/runs/$run_id" \
        | jq -er '.pull_requests[0].number | select(type == "number")'
}

emit_null() {
    local key=$1 case_json=$2 evidence_cutoff=$3 matures_at=$4 reason=$5
    jq -cn --arg opportunityKey "$key" --argjson case "$case_json" \
        --arg evidenceCutoff "$evidence_cutoff" --arg maturesAt "$matures_at" \
        --arg reason "$reason" \
        '{value:null,opportunityKey:$opportunityKey,case:$case,
          evidenceCutoff:$evidenceCutoff,maturesAt:$maturesAt,provenance:[],
          diagnostics:{missingReason:$reason}}'
}

grade_run() {
    local request run_id repository workflow run_sha created_at evidence_at
    local matures_at evidence_cutoff number initial_sha cutoff_sha count value case_json
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
    if ! created_at=$(normalize_timestamp "$created_at") || ! evidence_at=$(normalize_timestamp "$evidence_at"); then
        printf '%s\n' '{"value":null,"opportunityKey":"invalid-timestamp","case":{"invalidTimestamp":true},"evidenceCutoff":"1970-01-01T00:00:00Z","maturesAt":"1970-01-01T00:00:00Z","provenance":[],"diagnostics":{"missingReason":"invalid timestamp"}}'
        return
    fi
    matures_at=$(add_seconds "$created_at" "$MATURATION_SECONDS")
    if (( $(timestamp_epoch "$evidence_at") < $(timestamp_epoch "$matures_at") )); then
        evidence_cutoff=$evidence_at
    else
        evidence_cutoff=$matures_at
    fi

    if [[ $repository != "$REPOSITORY" || $workflow != "$WORKFLOW_NAME" \
        || $(printf '%s\n' "$request" | jq -r '.run.eventName') != "pull_request" ]]; then
        emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" \
            "run is not an eligible pull-request assignment"
        return
    fi
    if ! number=$(recover_pull_request "$request" "$run_id" "$repository"); then
        emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" \
            "pull-request assignment unavailable"
        return
    fi
    case_json=$(jq -cn --argjson pullRequest "$number" '{pullRequest:$pullRequest}')
    if ! cutoff_sha=$(latest_commit_at_cutoff "$repository" "$number" "$evidence_cutoff") \
        || ! count=$(count_new_valid_changesets "$repository" "$run_sha" "$cutoff_sha"); then
        emit_null "pull-request:$number" "$case_json" "$evidence_cutoff" "$matures_at" \
            "pull-request evidence unavailable"
        return
    fi
    value=$(printf '%s\n' "{\"valid\":true,\"newValidCount\":$count}" | metric)
    jq -cn --argjson value "$value" --arg opportunityKey "pull-request:$number" \
        --argjson case "$case_json" --arg evidenceCutoff "$evidence_cutoff" \
        --arg maturesAt "$matures_at" --arg repository "$repository" \
        --arg pullRequest "$number" --arg cutoffSha "$cutoff_sha" --argjson count "$count" \
        '{value:$value,opportunityKey:$opportunityKey,case:$case,
          evidenceCutoff:$evidenceCutoff,maturesAt:$maturesAt,
          provenance:[{repository:$repository,kind:"pull-request",ref:$pullRequest},
                      {repository:$repository,kind:"pull-request-commit",ref:$cutoffSha}],
          diagnostics:{"new-valid-changeset-count":$count}}'
}

case ${1:-} in
    --definition) definition ;;
    --metric) metric ;;
    --grade-run) grade_run ;;
    *) printf 'usage: %s --definition|--metric|--grade-run\n' "$0" >&2; exit 1 ;;
esac
