#!/usr/bin/env bash

set -euo pipefail

export LC_ALL=C

REPOSITORY=github/gh-aw
WORKFLOW_NAME="Blog Auditor"
SOURCE_PATH=".github/workflows/blog-auditor.md"
ADOPTION_COMMIT=31dbf0d7997d44bd86983726e83b7ba3813eafac
ADOPTED_AT=2025-10-22T20:45:14Z
TARGET_URL="https://githubnext.com/projects/agentic-workflows/"
MATURATION_SECONDS=86400

definition() {
    cat <<'JSON'
{
  "schemaVersion": 4,
  "grader": "operational-value",
  "repository": "github/gh-aw",
  "workflowName": "Blog Auditor",
  "sourcePath": ".github/workflows/blog-auditor.md",
  "adoption": {
    "commit": "31dbf0d7997d44bd86983726e83b7ba3813eafac",
    "adoptedAt": "2025-10-22T20:45:14Z"
  },
  "operationalValue": "Publish a repository-local audit verdict for the assigned weekly GitHub Next Agentic Workflows blog availability opportunity.",
  "evidence": {
    "opportunity": "One weekly opportunity to audit https://githubnext.com/projects/agentic-workflows/ for accessibility, expected content, and workflow snippet validity.",
    "assignment": "Bind by run createdAt to the Monday-based UTC week containing the run. Key: blog-audit:https://githubnext.com/projects/agentic-workflows/:week:<YYYY-MM-DD>. Duplicate runs in the same week retain the same key.",
    "accepted": "A GitHub Discussion in github/gh-aw Audits whose body references the exact run URL, target URL, and required validation sections, and whose title/body contains a recognized PASSED or FAILED audit verdict. Agent traces, evals, and token/activity counts are excluded.",
    "repositories": ["github/gh-aw"],
    "collection": "With discussions:read, search GitHub Discussions for the run URL and inspect matching discussion title, category, body, URL, number, and createdAt no later than the evidence cutoff.",
    "maturation": "One day after run.createdAt; audit discussions are intended to be created during the 10 minute run, and the 24 hour window makes delayed safe-output publication explicit and stable.",
    "zeroRule": "A successful discussion search before cutoff with no matching recognized audit discussion scores 0.",
    "missingRule": "Invalid requests, unavailable GitHub API/search evidence, inaccessible discussion bodies, malformed timestamps, or evidence requiring unavailable permissions score null, never zero."
  },
  "primaryMetric": {
    "id": "published-audit-verdict",
    "formula": "Return 1 when valid evidence contains a matching discussion with recognized PASSED or FAILED verdict, exact target URL, run URL, and required validation sections; return 0 when search completed and no match is found; otherwise null.",
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
    "targetAttained": {
      "valid": true,
      "searched": true,
      "matched": true,
      "recognizedVerdict": true,
      "hasTargetUrl": true,
      "hasRunUrl": true,
      "hasRequiredChecks": true
    },
    "targetMissed": {
      "valid": true,
      "searched": true,
      "matched": false,
      "recognizedVerdict": false,
      "hasTargetUrl": false,
      "hasRunUrl": false,
      "hasRequiredChecks": false
    },
    "missing": {
      "valid": false,
      "searched": false,
      "missingReason": "discussion-search-unavailable"
    },
    "malformed": {
      "valid": "yes",
      "searched": true,
      "matched": true
    }
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
      if .valid != true or (.searched | type) != "boolean" then null
      elif .matched == true
        and .recognizedVerdict == true
        and .hasTargetUrl == true
        and .hasRunUrl == true
        and .hasRequiredChecks == true then 1
      elif .searched == true then 0
      else null
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

week_start() {
    jq -nr --arg timestamp "$1" '
        ($timestamp | fromdateiso8601) as $epoch
        | (($epoch / 86400) | floor) as $day
        | (($day + 3) % 7) as $daysSinceMonday
        | (($day - $daysSinceMonday) * 86400 | todateiso8601 | split("T")[0])
    '
}

run_url() {
    local repository=$1 run_id=$2
    printf 'https://github.com/%s/actions/runs/%s\n' "$repository" "$run_id"
}

case_for_run() {
    local repository=$1 run_id=$2 created_at=$3
    local week run_url_value
    week=$(week_start "$created_at") || return 1
    run_url_value=$(run_url "$repository" "$run_id")
    jq -cn \
        --arg targetUrl "$TARGET_URL" \
        --arg weekStart "$week" \
        --arg runUrl "$run_url_value" \
        '{targetUrl: $targetUrl, weekStart: $weekStart, runUrl: $runUrl}'
}

valid_case() {
    jq -e --arg targetUrl "$TARGET_URL" '
        (.targetUrl == $targetUrl)
        and (.weekStart | type == "string" and test("^[0-9]{4}-[0-9]{2}-[0-9]{2}$"))
        and (.runUrl | type == "string" and test("^https://github\\.com/[^/]+/[^/]+/actions/runs/[0-9]+$"))
    ' >/dev/null
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

verification_observation() {
    local request=$1 evidence_cutoff=$2 matures_at=$3
    local case_json opportunity_key value evidence
    case_json=$(printf '%s\n' "$request" | jq -c '.case')
    if [[ $case_json == null ]]; then
        case_json=$(case_for_run "$REPOSITORY" "1" "$ADOPTED_AT")
    fi
    opportunity_key=$(printf '%s\n' "$case_json" | jq -r '"blog-audit:" + .targetUrl + ":week:" + .weekStart')
    evidence=$(jq -cn '{valid: true, searched: true, matched: true, recognizedVerdict: true, hasTargetUrl: true, hasRunUrl: true, hasRequiredChecks: true}')
    value=$(printf '%s\n' "$evidence" | metric)
    jq -cn \
        --argjson value "$value" \
        --arg opportunityKey "$opportunity_key" \
        --argjson case "$case_json" \
        --arg evidenceCutoff "$evidence_cutoff" \
        --arg maturesAt "$matures_at" \
        '{value: $value, opportunityKey: $opportunityKey, case: $case,
          evidenceCutoff: $evidenceCutoff, maturesAt: $maturesAt,
          provenance: [{repository: "github/gh-aw", kind: "verification", ref: "validation-example"}],
          diagnostics: {verification: true}}'
}

github_discussion_search() {
    local query=$1 output=$2
    local token=${GH_TOKEN:-${GITHUB_TOKEN:-}}
    [[ -n $token ]] || return 2
    GH_TOKEN=$token gh api graphql \
        -f query='query($q:String!){ search(type: DISCUSSION, query: $q, first: 10) { nodes { ... on Discussion { number title body createdAt url category { name } } } } }' \
        -f q="$query" >"$output"
}

collect_evidence() {
    local repository=$1 case_json=$2 evidence_cutoff=$3 output=$4
    local run_url_value query search_file created_at
    run_url_value=$(printf '%s\n' "$case_json" | jq -r '.runUrl')
    query="repo:$repository \"$run_url_value\""
    search_file=$(mktemp "${TMPDIR:-/tmp}/blog-auditor-discussions.XXXXXX")
    if ! github_discussion_search "$query" "$search_file"; then
        rm -f "$search_file"
        jq -cn '{valid: false, searched: false, missingReason: "discussion-search-unavailable"}' >"$output"
        return
    fi

    jq -c \
        --arg runUrl "$run_url_value" \
        --arg targetUrl "$TARGET_URL" \
        --arg evidenceCutoff "$evidence_cutoff" '
        def text: ((.title // "") + "\n" + (.body // ""));
        def recognized_verdict:
          ((.title // "") | test("Agentic Workflows blog audit - PASSED"; "i"))
          or ((.title // "") | test("Agentic Workflows blog out-of-date or unavailable"; "i"))
          or (text | test("Agentic Workflows Blog Audit - (PASSED|FAILED)"; "i"));
        def has_required_checks:
          (text | test("HTTP Status"; "i"))
          and (text | test("Final URL"; "i"))
          and (text | test("Content Length"; "i"))
          and (text | test("Keywords?"; "i"))
          and (text | test("Code Snippets?|snippet"; "i"));
        [.data.search.nodes[]?
          | select((.createdAt // "9999-12-31T23:59:59Z") <= $evidenceCutoff)
          | select(((.category.name // "") | ascii_downcase) == "audits")
          | select((.body // "") | contains($runUrl))
          | select(text | contains($targetUrl))
          | . + {
              recognizedVerdict: recognized_verdict,
              hasTargetUrl: (text | contains($targetUrl)),
              hasRunUrl: ((.body // "") | contains($runUrl)),
              hasRequiredChecks: has_required_checks
            }
        ] | sort_by(.createdAt) | last as $match
        | if $match == null then
            {valid: true, searched: true, matched: false,
             recognizedVerdict: false, hasTargetUrl: false, hasRunUrl: false, hasRequiredChecks: false}
          else
            {valid: true, searched: true, matched: true,
             discussionNumber: $match.number, discussionUrl: $match.url,
             discussionCreatedAt: $match.createdAt, discussionTitle: $match.title,
             recognizedVerdict: $match.recognizedVerdict,
             hasTargetUrl: $match.hasTargetUrl,
             hasRunUrl: $match.hasRunUrl,
             hasRequiredChecks: $match.hasRequiredChecks}
          end
    ' "$search_file" >"$output" || jq -cn '{valid: false, searched: false, missingReason: "discussion-search-malformed"}' >"$output"
    rm -f "$search_file"
    created_at=$(jq -r '.discussionCreatedAt // empty' "$output")
    [[ -n $created_at ]] || return
}

grade_run() {
    local request run_id repository workflow created_at evidence_at run_sha
    local matures_at evidence_cutoff evidence_epoch matures_epoch
    local case_json opportunity_key evidence_file evidence value discussion_url discussion_number

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

    case_json=$(printf '%s\n' "$request" | jq -c '.case')
    if [[ $case_json == null ]]; then
        if ! case_json=$(case_for_run "$repository" "$run_id" "$created_at"); then
            emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" "assignment-unavailable"
            return
        fi
    elif ! printf '%s\n' "$case_json" | valid_case; then
        emit_null "run:$run_id" '{"assignmentMissing":true}' "$evidence_cutoff" "$matures_at" "invalid-case"
        return
    fi

    opportunity_key=$(printf '%s\n' "$case_json" | jq -r '"blog-audit:" + .targetUrl + ":week:" + .weekStart')

    if [[ $repository != "$REPOSITORY" || $workflow != "$WORKFLOW_NAME" ]]; then
        emit_null "$opportunity_key" "$case_json" "$evidence_cutoff" "$matures_at" "run repository or workflow does not match the frozen contract"
        return
    fi

    if printf '%s\n' "$request" | jq -e '.config.verification == true' >/dev/null 2>&1; then
        verification_observation "$request" "$evidence_cutoff" "$matures_at"
        return
    fi

    evidence_file=$(mktemp "${TMPDIR:-/tmp}/blog-auditor-evidence.XXXXXX")
    trap 'rm -f "$evidence_file"' RETURN
    collect_evidence "$repository" "$case_json" "$evidence_cutoff" "$evidence_file"
    evidence=$(cat "$evidence_file")
    value=$(printf '%s\n' "$evidence" | metric)

    if [[ $value == null ]]; then
        jq -cn \
            --arg opportunityKey "$opportunity_key" \
            --argjson case "$case_json" \
            --arg evidenceCutoff "$evidence_cutoff" \
            --arg maturesAt "$matures_at" \
            --argjson diagnostics "$evidence" \
            '{value: null, opportunityKey: $opportunityKey, case: $case,
              evidenceCutoff: $evidenceCutoff, maturesAt: $maturesAt,
              provenance: [], diagnostics: $diagnostics}'
        return
    fi

    discussion_url=$(printf '%s\n' "$evidence" | jq -r '.discussionUrl // empty')
    discussion_number=$(printf '%s\n' "$evidence" | jq -r '.discussionNumber // empty')
    if [[ -n $discussion_url && -n $discussion_number ]]; then
        jq -cn \
            --argjson value "$value" \
            --arg opportunityKey "$opportunity_key" \
            --argjson case "$case_json" \
            --arg evidenceCutoff "$evidence_cutoff" \
            --arg maturesAt "$matures_at" \
            --arg repository "$repository" \
            --arg discussionNumber "$discussion_number" \
            --arg discussionUrl "$discussion_url" \
            --arg runSha "$run_sha" \
            --argjson diagnostics "$evidence" \
            '{value: $value, opportunityKey: $opportunityKey, case: $case,
              evidenceCutoff: $evidenceCutoff, maturesAt: $maturesAt,
              provenance: [{repository: $repository, kind: "discussion", ref: $discussionNumber},
                           {repository: $repository, kind: "workflow-run", ref: $case.runUrl},
                           {repository: $repository, kind: "commit", ref: $runSha}],
              diagnostics: ($diagnostics + {discussionUrl: $discussionUrl})}'
    else
        jq -cn \
            --argjson value "$value" \
            --arg opportunityKey "$opportunity_key" \
            --argjson case "$case_json" \
            --arg evidenceCutoff "$evidence_cutoff" \
            --arg maturesAt "$matures_at" \
            --arg repository "$repository" \
            --arg runSha "$run_sha" \
            --argjson diagnostics "$evidence" \
            '{value: $value, opportunityKey: $opportunityKey, case: $case,
              evidenceCutoff: $evidenceCutoff, maturesAt: $maturesAt,
              provenance: (if $value == 0 then [{repository: $repository, kind: "discussion-search", ref: $case.runUrl},
                                                {repository: $repository, kind: "commit", ref: $runSha}] else [] end),
              diagnostics: $diagnostics}'
    fi
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
