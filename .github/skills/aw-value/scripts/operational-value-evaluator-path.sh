#!/usr/bin/env bash

set -euo pipefail

fail() {
    printf 'error: %s\n' "$*" >&2
    exit 1
}

[[ $# -eq 1 ]] || fail "usage: operational-value-evaluator-path.sh WORKFLOW-NAME"

workflow_name=$1
[[ $workflow_name =~ ^[a-z0-9]+(-[a-z0-9]+)*$ ]] \
    || fail "workflow name must contain lowercase letters, numbers, and single hyphens"

printf '.github/graders/%s-operational-value.sh\n' "$workflow_name"