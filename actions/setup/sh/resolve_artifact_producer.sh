#!/usr/bin/env bash
set -euo pipefail

[[ "$INVOCATION_ID" =~ ^[a-f0-9]{8}-$ ]] || { echo "::error::Invalid artifact invocation identity"; exit 1; }
[[ "$GITHUB_RUN_ATTEMPT" =~ ^[1-9][0-9]*$ ]] || { echo "::error::Invalid consumer run attempt"; exit 1; }
ATTEMPT=$(jq -er --arg producer "$PRODUCER_JOB" --arg identity "$INVOCATION_ID" --argjson current "$GITHUB_RUN_ATTEMPT" '
  if type == "object" and keys == ["attempt", "identity", "producer"]
    and .producer == $producer and .identity == $identity
    and (.attempt | type == "number") and (.attempt | floor == .)
    and .attempt >= 1 and .attempt <= $current
  then .attempt else error("Invalid artifact producer metadata") end
' "$METADATA_DIRECTORY/producer.json")
if (( ATTEMPT < GITHUB_RUN_ATTEMPT )) && [[ "$PRODUCER_RESULT" != "success" ]]; then
  echo "::error::Refusing retained artifacts from a failed or skipped producer"
  exit 1
fi
export GITHUB_RUN_ATTEMPT="$ATTEMPT"
