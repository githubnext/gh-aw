#!/usr/bin/env bash
set -euo pipefail

[[ "$INVOCATION_ID" =~ ^[a-f0-9]{8}-$ ]] || { echo "::error::Invalid artifact invocation identity"; exit 1; }
[[ "$PRODUCER_ATTEMPT" =~ ^[1-9][0-9]*$ ]] || { echo "::error::Invalid artifact producer attempt"; exit 1; }
mkdir -p "$METADATA_DIRECTORY"
jq -n --argjson attempt "$PRODUCER_ATTEMPT" --arg producer "$PRODUCER_JOB" --arg identity "$INVOCATION_ID" \
  '{attempt: $attempt, producer: $producer, identity: $identity}' > "$METADATA_DIRECTORY/producer.json"
