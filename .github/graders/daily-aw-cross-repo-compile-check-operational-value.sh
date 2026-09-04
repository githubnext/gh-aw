#!/usr/bin/env bash

set -euo pipefail

export LC_ALL=C

REPOSITORY=github/gh-aw
WORKFLOW_NAME="Daily AW Cross-Repo Compile Check"
MATURATION_SECONDS=604800
ADOPTION_COMMIT=596c5c8aa2cf4866b7a6e85964ade809d17a45f8
ADOPTION_TIME=2026-04-17T04:50:20Z

definition() {
    cat <<'JSON'
{
  "schemaVersion": 4,
  "grader": "operational-value",
  "repository": "github/gh-aw",
  "workflowName": "Daily AW Cross-Repo Compile Check",
  "sourcePath": ".github/workflows/daily-aw-cross-repo-compile-check.md",
  "adoption": {
    "commit": "596c5c8aa2cf4866b7a6e85964ade809d17a45f8",
    "adoptedAt": "2026-04-17T04:50:20Z"
  },
  "operationalValue": "For each selected public repository, attain a compatible gh-aw workflow state demonstrated by a successful post-fix compile.",
  "evidence": {
    "opportunity": "A public repository selected by the workflow's lock-file search and star ranking.",
    "assignment": "Use the supplied repository result; when absent, select the first valid lock-file search result in API order. Repeated repository keys are retained and reruns share their run ID.",
    "accepted": "A repository's recorded post-fix gh-aw compile status, with the repository and run commit as provenance.",
    "repositories": ["github/gh-aw"],
    "collection": "Read the run's selected-repository result or reconstruct one through the GitHub code-search API; accept only explicit success or failure statuses.",
    "maturation": "Seven days after run creation; evidenceCutoff is the earlier of evidenceAt and maturesAt.",
    "zeroRule": "An explicit post-fix compile failure scores zero.",
    "missingRule": "Missing, malformed, or unavailable repository result evidence scores null; unavailable is never treated as failure."
  },
  "primaryMetric": {
    "id": "post-fix-compile-attainment",
    "formula": "1 for explicit post-fix compile success; 0 for explicit post-fix compile failure; null otherwise.",
    "direction": "higher_is_better"
  },
  "diagnosticMetrics": [
    {
      "id": "pre-fix-compile-attainment",
      "name": "Pre-fix compile attainment",
      "formula": "1 for explicit pre-fix compile success, otherwise 0 only for explicit pre-fix failure.",
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
    "targetAttained": {"eligible": true, "postFixStatus": "success", "preFixStatus": "failure"},
    "targetMissed": {"eligible": true, "postFixStatus": "failure", "preFixStatus": "failure"},
    "missing": {"eligible": false},
    "malformed": {"eligible": true, "postFixStatus": "success"}
  }
}
JSON
}

metric() {
    jq -r '
      if .eligible != true or (.postFixStatus | type) != "string"
        or (.postFixStatus != "success" and .postFixStatus != "failure")
        or (.preFixStatus | type) != "string"
        or (.preFixStatus != "success" and .preFixStatus != "failure") then null
      elif .postFixStatus == "success" then 1 else 0 end
    '
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

assign_case() {
    local result
    result=$(gh api "search/code?q=gh-aw-metadata+in:file+filename:.lock.yml&per_page=1" 2>/dev/null) || return 1
    printf '%s\n' "$result" | jq -ce '
      .items[0].repository.full_name as $repository
      | select($repository | type == "string" and test("^[^/]+/[^/]+$"))
      | {repository: $repository}
    '
}

grade_run() {
    local request run_id repository workflow run_sha created_at evidence_at
    local matures_at evidence_cutoff case_json evidence value diagnostics

    request=$(cat)
    run_id=$(printf '%s\n' "$request" | jq -r '.run.id // "unknown"')
    if ! printf '%s\n' "$request" | jq -e '
      .schemaVersion == 1 and (.run.id | type == "string" and length > 0)
      and (.run.repository | type == "string") and (.run.workflow | type == "string")
      and (.run.sha | type == "string" and test("^[0-9a-f]{40}$"))
      and (.run.createdAt | type == "string") and (.evidenceAt | type == "string")
      and (.case == null or (.case | type == "object"))
    ' >/dev/null 2>&1; then
        printf '%s\n' '{"value":null,"opportunityKey":"invalid-request","case":{"invalidRequest":true},"evidenceCutoff":"1970-01-01T00:00:00Z","maturesAt":"1970-01-01T00:00:00Z","provenance":[],"diagnostics":{"missingReason":"invalid request"}}'
        return
    fi

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
    if [[ $(jq -nr --arg a "$evidence_at" --arg b "$matures_at" '$a|fromdateiso8601 < ($b|fromdateiso8601)') == true ]]; then
        evidence_cutoff=$evidence_at
    else
        evidence_cutoff=$matures_at
    fi

    case_json=$(printf '%s\n' "$request" | jq -c '.case')
    if [[ $case_json == null ]]; then
        case_json=$(assign_case 2>/dev/null || true)
    fi
    if [[ -z $case_json || $case_json == null ]]; then
        jq -cn --arg run "$run_id" --arg cutoff "$evidence_cutoff" --arg matures "$matures_at" \
          '{value:null,opportunityKey:("run:"+$run),case:{assignmentMissing:true},evidenceCutoff:$cutoff,maturesAt:$matures,provenance:[],diagnostics:{missingReason:"assignment-unavailable"}}'
        return
    fi
    evidence=$(printf '%s\n' "$request" | jq -c --argjson fallback "$case_json" \
      'if .event? | type == "object" and (.event.result? | type == "object") then .event.result else $fallback end')
    value=$(printf '%s\n' "$evidence" | metric)
    diagnostics=$(printf '%s\n' "$evidence" | jq -c '
      if .eligible == true and (.preFixStatus == "success" or .preFixStatus == "failure")
      then {"pre-fix-compile-attainment": (if .preFixStatus == "success" then 1 else 0 end)}
      else {"pre-fix-compile-attainment": null} end')
    jq -cn --argjson value "$value" --argjson case "$case_json" \
      --arg key "repository:$(printf '%s\n' "$case_json" | jq -r '.repository // "unknown"')" \
      --arg cutoff "$evidence_cutoff" --arg matures "$matures_at" --arg repo "$repository" \
      --arg sha "$run_sha" --argjson diagnostics "$diagnostics" \
      '{value:$value,opportunityKey:$key,case:$case,evidenceCutoff:$cutoff,maturesAt:$matures,provenance:[{repository:$repo,kind:"compile-result",ref:$sha}],diagnostics:$diagnostics}'
}

case ${1:-} in
  --definition) definition ;;
  --metric) metric ;;
  --grade-run) grade_run ;;
  *) printf 'usage: %s --definition|--metric|--grade-run\n' "$0" >&2; exit 1 ;;
esac
