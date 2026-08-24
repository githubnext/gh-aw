#!/usr/bin/env bash

set -euo pipefail

fail() {
    printf 'error: %s\n' "$*" >&2
    exit 1
}

[[ $# -eq 1 ]] || fail "usage: verify-operational-value-evaluator.sh <operational-value-evaluator.sh>"

evaluator=$1
[[ -f $evaluator ]] || fail "operational-value evaluator not found: $evaluator"
[[ -x $evaluator ]] || fail "operational-value evaluator is not executable: $evaluator"
command -v jq >/dev/null 2>&1 || fail "jq is required"
bash -n "$evaluator"

definition=$("$evaluator" --definition)
printf '%s\n' "$definition" | jq -e '
    .schemaVersion == 4
    and .grader == "operational-value"
    and (.repository | type == "string" and test("^[^/]+/[^/]+$"))
    and (.workflowName | type == "string" and length > 0)
    and (.sourcePath | type == "string" and startswith(".github/workflows/") and endswith(".md"))
    and (.adoption.commit | type == "string" and test("^[0-9a-f]{40}$"))
    and (.adoption.adoptedAt | type == "string" and test("^[0-9]{4}-[0-9]{2}-[0-9]{2}T"))
    and (.operationalValue | type == "string" and length > 0)
    and (.evidence.opportunity | type == "string" and length > 0)
    and (.evidence.assignment | type == "string" and length > 0)
    and (.evidence.accepted | type == "string" and length > 0)
    and (.evidence.repositories | type == "array" and length > 0)
    and (all(.evidence.repositories[]; type == "string" and test("^[^/]+/[^/]+$")))
    and (.evidence.collection | type == "string" and length > 0)
    and (.evidence.maturation | type == "string" and length > 0)
    and (.evidence.zeroRule | type == "string" and length > 0)
    and (.evidence.missingRule | type == "string" and length > 0)
    and (.primaryMetric.id | type == "string" and length > 0)
    and (.primaryMetric.formula | type == "string" and length > 0)
    and (.primaryMetric.direction == "higher_is_better")
    and (.validationExamples | has("targetAttained") and has("targetMissed") and has("missing") and has("malformed"))
    and (.baseline.mode == "baseline-comparable" or .baseline.mode == "attainment-only")
    and (if .baseline.mode == "baseline-comparable" then
        (.baseline.value | type == "number" and . >= 0 and . <= 1)
        and (.baseline.evidenceCutoff | type == "string" and length > 0)
        and (.baseline.provenance | type == "array" and length > 0)
      else
        .baseline.value == null
        and .baseline.evidenceCutoff == null
      end)
' >/dev/null || fail "operational-value evaluator definition is invalid"

for example_name in targetAttained targetMissed missing malformed; do
    evidence=$(printf '%s\n' "$definition" | jq -c --arg name "$example_name" '.validationExamples[$name]')
    result=$(printf '%s\n' "$evidence" | "$evaluator" --metric)
    printf '%s\n' "$result" | jq -e '. == null or (type == "number" and . >= 0 and . <= 1)' >/dev/null \
        || fail "--metric returned an invalid score for $example_name"
    case $example_name in
        targetAttained) target_attained=$result ;;
        targetMissed) target_missed=$result ;;
        missing|malformed)
            [[ $result == null ]] || fail "--metric must return null for $example_name"
            ;;
    esac
done

jq -en --argjson attained "$target_attained" --argjson missed "$target_missed" \
    '$attained != null and $missed != null and $attained > $missed' >/dev/null \
    || fail "targetAttained must score higher than targetMissed"

printf 'verified %s\n' "$evaluator"