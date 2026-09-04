#!/usr/bin/env bash

set -euo pipefail

export LC_ALL=C

REPOSITORY=github/gh-aw
WORKFLOW_NAME="Commit Changes Analyzer"
ADOPTION_COMMIT=134fc10e16a282117bfb386199c1d775a0a8f288
ADOPTION_AT=2026-09-04T13:32:06Z
MATURATION_SECONDS=604800

tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/commit-changes-analyzer-operational-value.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM

definition() {
    cat <<'JSON'
{
  "schemaVersion": 4,
  "grader": "operational-value",
  "repository": "github/gh-aw",
  "workflowName": "Commit Changes Analyzer",
  "sourcePath": ".github/workflows/commit-changes-analyzer.md",
  "adoption": {
    "commit": "134fc10e16a282117bfb386199c1d775a0a8f288",
    "adoptedAt": "2026-09-04T13:32:06Z"
  },
  "operationalValue": "For the requested commit range, attain the workflow's required developer report in a development discussion.",
  "evidence": {
    "opportunity": "The commit URL supplied to a workflow_dispatch run.",
    "assignment": "Use case.baseSha or event.inputs.commit_url; when both are null, select the deterministic closest post-run development discussion titled 'Changes Analysis: Since commit <short-SHA> - ...' and resolve its short SHA. Repeated keys are retained as the same commit opportunity.",
    "accepted": "A same-repository development discussion created by the run's maturity cutoff has the required title prefix, references the assigned base commit, and contains every report section mandated by the workflow.",
    "repositories": ["github/gh-aw"],
    "collection": "Read repository discussions with discussions:read and resolve the report title's commit through the repository commit API; use only discussions created at or after the run and no later than the capped cutoff.",
    "maturation": "Seven days after run creation; evidenceCutoff is the earlier of evidenceAt and maturesAt.",
    "zeroRule": "An eligible discussion that lacks any required report section scores zero.",
    "missingRule": "Unavailable discussions, commit resolution, malformed assignment, or no eligible discussion scores null; an eligible discussion with an incomplete report is zero."
  },
  "primaryMetric": {
    "id": "report-contract-completeness",
    "formula": "1 when the assigned discussion exists and contains all nine required report sections; 0 when it exists but any section is absent; null when assignment or evidence is unavailable.",
    "direction": "higher_is_better"
  },
  "baseline": {
    "mode": "attainment-only",
    "value": null,
    "evidenceCutoff": null,
    "provenance": []
  },
  "validationExamples": {
    "targetAttained": {"valid": true, "discussionFound": true, "contractComplete": true},
    "targetMissed": {"valid": true, "discussionFound": true, "contractComplete": false},
    "missing": {"valid": false},
    "malformed": {"valid": "yes", "discussionFound": true}
  }
}
JSON
}

metric() {
    jq '
      if .valid != true or .discussionFound != true
        or (.contractComplete | type) != "boolean" then null
      elif .contractComplete then 1 else 0 end
    '
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

discussion_pages() {
    local page output
    : >"$tmp_dir/discussions.jsonl"
    for page in 1 2 3 4 5; do
        if ! output=$(github_api "repos/$REPOSITORY/discussions?per_page=100&page=$page"); then
            return 1
        fi
        printf '%s\n' "$output" | jq -c '.[]' >>"$tmp_dir/discussions.jsonl" || return 1
        [[ $(printf '%s\n' "$output" | jq 'length') -lt 100 ]] && break
    done
}

required_sections='["## Executive Summary","## Detailed Changes","### Files Changed Summary","### Code Impact","### Commit History","### Functional Areas","### Notable Changes","### Related Work","## Developer Notes"]'

metric_for_discussion() {
    local discussion=$1
    printf '%s\n' "$discussion" | jq -c --argjson sections "$required_sections" '
        . as $discussion |
        {
          valid: true,
          discussionFound: true,
          id: $discussion.id,
          contractComplete: (
            ($discussion.title | startswith("Changes Analysis: Since commit "))
            and ($discussion.category.name == "dev")
            and ($discussion.body | type == "string")
            and (all($sections[] as $section; $discussion.body | contains($section)))
          )
        }'
}

resolve_commit() {
    local short_sha=$1 response
    response=$(github_api "repos/$REPOSITORY/commits/$short_sha") || return 1
    printf '%s\n' "$response" | jq -er '.sha | select(type == "string" and test("^[0-9a-f]{40}$"))'
}

extract_case() {
    local request=$1 commit_url base_sha
    base_sha=$(printf '%s\n' "$request" | jq -r '.case.baseSha // empty')
    if [[ -n $base_sha ]]; then
        printf '%s\n' "$base_sha"
        return
    fi
    commit_url=$(printf '%s\n' "$request" | jq -r '.event.inputs.commit_url // empty')
    if [[ $commit_url =~ ^https://github.com/([^/]+)/([^/]+)/commit/([0-9a-fA-F]{7,40})$ ]] \
        && [[ ${BASH_REMATCH[1]}/${BASH_REMATCH[2]} == "$REPOSITORY" ]]; then
        resolve_commit "${BASH_REMATCH[3]}"
    fi
}

grade_run() {
    local request run_id repository workflow created_at evidence_at matures_at evidence_cutoff
    local evidence_epoch matures_epoch base_sha opportunity_key discussion
    request=$(cat)
    run_id=$(printf '%s\n' "$request" | jq -r '.run.id // empty')
    repository=$(printf '%s\n' "$request" | jq -r '.run.repository // empty')
    workflow=$(printf '%s\n' "$request" | jq -r '.run.workflow // empty')
    created_at=$(printf '%s\n' "$request" | jq -r '.run.createdAt // empty')
    evidence_at=$(printf '%s\n' "$request" | jq -r '.evidenceAt // empty')
    if ! created_at=$(normalize_timestamp "$created_at") || ! evidence_at=$(normalize_timestamp "$evidence_at"); then
        printf '%s\n' '{"value":null,"opportunityKey":"invalid-request","case":{"invalidRequest":true},"evidenceCutoff":"1970-01-01T00:00:00Z","maturesAt":"1970-01-01T00:00:00Z","provenance":[]}'
        return
    fi
    matures_at=$(add_seconds "$created_at" "$MATURATION_SECONDS")
    evidence_epoch=$(timestamp_epoch "$evidence_at")
    matures_epoch=$(timestamp_epoch "$matures_at")
    if (( evidence_epoch < matures_epoch )); then evidence_cutoff=$evidence_at; else evidence_cutoff=$matures_at; fi
    opportunity_key="run:$run_id"
    if [[ $repository != "$REPOSITORY" || $workflow != "$WORKFLOW_NAME" ]]; then
        jq -cn --arg key "$opportunity_key" --arg cutoff "$evidence_cutoff" --arg matures "$matures_at" \
            '{value:null,opportunityKey:$key,case:{assignmentMissing:true},evidenceCutoff:$cutoff,maturesAt:$matures,provenance:[]}'
        return
    fi
    base_sha=$(extract_case "$request" || true)
    if [[ -z $base_sha ]] && discussion_pages; then
        base_sha=$(while IFS= read -r discussion; do
            created=$(printf '%s\n' "$discussion" | jq -r '.created_at // empty')
            if [[ -n $created ]] && [[ $(timestamp_epoch "$created" 2>/dev/null || printf 0) -ge $(timestamp_epoch "$created_at") ]] \
                && [[ $(timestamp_epoch "$created" 2>/dev/null || printf 9999999999) -le $evidence_epoch ]]; then
                title=$(printf '%s\n' "$discussion" | jq -r '.title // empty')
                if [[ $title =~ ^Changes\ Analysis:\ Since\ commit\ ([0-9a-fA-F]{7,40})\ - ]]; then
                    resolve_commit "${BASH_REMATCH[1]}" && break
                fi
            fi
        done <"$tmp_dir/discussions.jsonl")
    fi
    [[ $base_sha =~ ^[0-9a-f]{40}$ ]] || {
        jq -cn --arg key "$opportunity_key" --arg cutoff "$evidence_cutoff" --arg matures "$matures_at" \
            '{value:null,opportunityKey:$key,case:{assignmentMissing:true},evidenceCutoff:$cutoff,maturesAt:$matures,provenance:[]}'
        return
    }
    opportunity_key="commit:$base_sha"
    discussion=''
    discussion_pages && discussion=$(while IFS= read -r candidate; do
        created=$(printf '%s\n' "$candidate" | jq -r '.created_at // empty')
        title=$(printf '%s\n' "$candidate" | jq -r '.title // empty')
        created_epoch=$(timestamp_epoch "$created" 2>/dev/null || printf 0)
        if [[ $created_epoch -ge $(timestamp_epoch "$created_at") ]] \
            && [[ $created_epoch -le $evidence_epoch ]] \
            && [[ $title =~ ^Changes\ Analysis:\ Since\ commit\ ([0-9a-fA-F]{7,40})\ - ]] \
            && [[ ${BASH_REMATCH[1]} == "${base_sha:0:${#BASH_REMATCH[1]}}" ]]; then
            printf '%s\n' "$candidate"
            break
        fi
    done <"$tmp_dir/discussions.jsonl")
    if [[ -z $discussion ]]; then
        jq -cn --arg key "$opportunity_key" --arg sha "$base_sha" --arg cutoff "$evidence_cutoff" --arg matures "$matures_at" \
            '{value:null,opportunityKey:$key,case:{baseSha:$sha},evidenceCutoff:$cutoff,maturesAt:$matures,provenance:[]}'
        return
    fi
    evidence=$(metric_for_discussion "$discussion")
    value=$(printf '%s\n' "$evidence" | metric)
    jq -cn --argjson value "$value" --arg key "$opportunity_key" --arg sha "$base_sha" \
        --arg cutoff "$evidence_cutoff" --arg matures "$matures_at" --argjson evidence "$evidence" \
        '{value:$value,opportunityKey:$key,case:{baseSha:$sha},evidenceCutoff:$cutoff,maturesAt:$matures,
            provenance:[{repository:"github/gh-aw",kind:"discussion",ref:($evidence.id|tostring)}],
          diagnostics:{reportContractComplete:$evidence.contractComplete}}'
}

case ${1:-} in
    --definition) definition ;;
    --metric) metric ;;
    --grade-run) grade_run ;;
    *) printf 'usage: %s --definition|--metric|--grade-run\n' "$0" >&2; exit 1 ;;
esac
