#!/usr/bin/env bash
# Parse token-usage.jsonl from the firewall proxy and append a markdown table
# to $GITHUB_STEP_SUMMARY. This script runs after the agent completes and the
# firewall logs are available at the known path.
#
# The token-usage.jsonl file is produced by AWF v0.25.8+ and contains one JSON
# object per line with per-request token usage data from the AI provider API.

set -euo pipefail

TOKEN_USAGE_FILE="/tmp/gh-aw/sandbox/firewall/logs/api-proxy-logs/token-usage.jsonl"

if [ ! -f "$TOKEN_USAGE_FILE" ] || [ ! -s "$TOKEN_USAGE_FILE" ]; then
  echo "No token usage data found, skipping summary"
  exit 0
fi

echo "Parsing token usage from: $TOKEN_USAGE_FILE"

# Use awk to aggregate token usage by model and compute totals.
# This avoids requiring jq or other dependencies beyond coreutils.
awk '
BEGIN {
  FS=","
  total_input = 0
  total_output = 0
  total_cache_read = 0
  total_cache_write = 0
  total_requests = 0
  total_duration = 0
}
{
  # Extract fields from JSON using simple pattern matching
  model = ""
  provider = ""
  input = 0; output = 0; cache_read = 0; cache_write = 0; duration = 0

  if (match($0, /"model":"([^"]*)"/, m)) model = m[1]
  if (match($0, /"provider":"([^"]*)"/, m)) provider = m[1]
  if (match($0, /"input_tokens":([0-9]+)/, m)) input = m[1] + 0
  if (match($0, /"output_tokens":([0-9]+)/, m)) output = m[1] + 0
  if (match($0, /"cache_read_tokens":([0-9]+)/, m)) cache_read = m[1] + 0
  if (match($0, /"cache_write_tokens":([0-9]+)/, m)) cache_write = m[1] + 0
  if (match($0, /"duration_ms":([0-9]+)/, m)) duration = m[1] + 0

  if (model == "") model = "unknown"

  # Aggregate by model
  models[model] = 1
  providers[model] = provider
  model_input[model] += input
  model_output[model] += output
  model_cache_read[model] += cache_read
  model_cache_write[model] += cache_write
  model_requests[model] += 1
  model_duration[model] += duration

  total_input += input
  total_output += output
  total_cache_read += cache_read
  total_cache_write += cache_write
  total_requests += 1
  total_duration += duration
}
END {
  if (total_requests == 0) exit

  # Format duration
  total_dur_s = total_duration / 1000.0

  printf "\n### 📊 Token Usage\n\n"
  printf "| Model | Input | Output | Cache Read | Cache Write | Requests | Duration |\n"
  printf "|-------|------:|-------:|-----------:|------------:|---------:|---------:|\n"

  for (model in models) {
    dur_s = model_duration[model] / 1000.0
    printf "| %s | %d | %d | %d | %d | %d | %.1fs |\n", \
      model, model_input[model], model_output[model], \
      model_cache_read[model], model_cache_write[model], \
      model_requests[model], dur_s
  }

  printf "| **Total** | **%d** | **%d** | **%d** | **%d** | **%d** | **%.1fs** |\n", \
    total_input, total_output, total_cache_read, total_cache_write, \
    total_requests, total_dur_s

  # Cache efficiency
  total_input_plus_cache = total_input + total_cache_read
  if (total_input_plus_cache > 0) {
    efficiency = (total_cache_read / total_input_plus_cache) * 100
    printf "\n_Cache efficiency: %.1f%%_\n", efficiency
  }
}
' "$TOKEN_USAGE_FILE" >> "$GITHUB_STEP_SUMMARY"

echo "Token usage summary appended to step summary"
