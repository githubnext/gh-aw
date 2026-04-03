#!/usr/bin/env bash
# Parse token-usage.jsonl from the firewall proxy and append a markdown table
# to $GITHUB_STEP_SUMMARY. This script runs after the agent completes and the
# firewall logs are available at the known path.
#
# The token-usage.jsonl file is produced by AWF v0.25.8+ and contains one JSON
# object per line with per-request token usage data from the AI provider API.
#
# Aggregated token totals are also written to /tmp/gh-aw/agent_usage.json so
# the data is bundled in the agent artifact and accessible to third-party tools.

set -euo pipefail

TOKEN_USAGE_FILE="/tmp/gh-aw/sandbox/firewall/logs/api-proxy-logs/token-usage.jsonl"

if [ ! -f "$TOKEN_USAGE_FILE" ] || [ ! -s "$TOKEN_USAGE_FILE" ]; then
  echo "No token usage data found, skipping summary"
  exit 0
fi

echo "Parsing token usage from: $TOKEN_USAGE_FILE"

# Use awk to aggregate token usage by model, then pipe through sort for
# deterministic output (sorted by total tokens descending).
# Regexes tolerate optional whitespace around ":" per standard JSON formatting.
awk '
BEGIN {
  total_input = 0
  total_output = 0
  total_cache_read = 0
  total_cache_write = 0
  total_requests = 0
  total_duration = 0
}
{
  # Extract fields from JSON using pattern matching.
  # Patterns tolerate optional whitespace after the colon and handle both
  # "key":"value" and "key": "value" forms.
  model = ""
  provider = ""
  input = 0; output = 0; cache_read = 0; cache_write = 0; duration = 0

  if (match($0, /"model" *: *"([^"]*)"/, m)) model = m[1]
  if (match($0, /"provider" *: *"([^"]*)"/, m)) provider = m[1]
  if (match($0, /"input_tokens" *: *([0-9]+)/, m)) input = m[1] + 0
  if (match($0, /"output_tokens" *: *([0-9]+)/, m)) output = m[1] + 0
  if (match($0, /"cache_read_tokens" *: *([0-9]+)/, m)) cache_read = m[1] + 0
  if (match($0, /"cache_write_tokens" *: *([0-9]+)/, m)) cache_write = m[1] + 0
  if (match($0, /"duration_ms" *: *([0-9]+)/, m)) duration = m[1] + 0

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

  total_dur_s = total_duration / 1000.0

  printf "\n### 📊 Token Usage\n\n"
  printf "| Model | Input | Output | Cache Read | Cache Write | Requests | Duration |\n"
  printf "|-------|------:|-------:|-----------:|------------:|---------:|---------:|\n"

  # Emit model rows with a sort key (total tokens, tab-separated) so we can
  # pipe through sort for deterministic ordering (highest tokens first).
  for (model in models) {
    dur_s = model_duration[model] / 1000.0
    total_tok = model_input[model] + model_output[model] + model_cache_read[model] + model_cache_write[model]
    printf "%d\t| %s | %d | %d | %d | %d | %d | %.1fs |\n", \
      total_tok, model, model_input[model], model_output[model], \
      model_cache_read[model], model_cache_write[model], \
      model_requests[model], dur_s
  }

  # Totals row (prefixed with 0 sort key so it always appears last after reverse sort,
  # but we will append it separately below)
}
' "$TOKEN_USAGE_FILE" | sort -t$'\t' -k1 -rn | cut -f2- > /tmp/gh-aw-token-rows.tmp

# Build the final table
{
  # Header (already written by awk above — re-emit here since awk output was redirected)
  printf "\n### 📊 Token Usage\n\n"
  printf "| Model | Input | Output | Cache Read | Cache Write | Requests | Duration |\n"
  printf "|-------|------:|-------:|-----------:|------------:|---------:|---------:|\n"

  # Sorted model rows
  cat /tmp/gh-aw-token-rows.tmp

  # Totals row
  awk '
  BEGIN { ti=0; to=0; cr=0; cw=0; tr=0; td=0 }
  {
    if (match($0, /"input_tokens" *: *([0-9]+)/, m)) ti += m[1]+0
    if (match($0, /"output_tokens" *: *([0-9]+)/, m)) to += m[1]+0
    if (match($0, /"cache_read_tokens" *: *([0-9]+)/, m)) cr += m[1]+0
    if (match($0, /"cache_write_tokens" *: *([0-9]+)/, m)) cw += m[1]+0
    if (match($0, /"duration_ms" *: *([0-9]+)/, m)) td += m[1]+0
    tr += 1
  }
  END {
    if (tr == 0) exit
    dur_s = td / 1000.0
    printf "| **Total** | **%d** | **%d** | **%d** | **%d** | **%d** | **%.1fs** |\n", \
      ti, to, cr, cw, tr, dur_s
    total_input_plus_cache = ti + cr
    if (total_input_plus_cache > 0) {
      eff = (cr / total_input_plus_cache) * 100
      printf "\n_Cache efficiency: %.1f%%_\n", eff
    }
  }
  ' "$TOKEN_USAGE_FILE"
} >> "$GITHUB_STEP_SUMMARY"

rm -f /tmp/gh-aw-token-rows.tmp

# Write agent_usage.json to the artifact folder so the data is bundled in the
# agent artifact and accessible to third-party tools.
awk '
BEGIN { ti=0; to=0; cr=0; cw=0 }
{
  if (match($0, /"input_tokens" *: *([0-9]+)/, m)) ti += m[1]+0
  if (match($0, /"output_tokens" *: *([0-9]+)/, m)) to += m[1]+0
  if (match($0, /"cache_read_tokens" *: *([0-9]+)/, m)) cr += m[1]+0
  if (match($0, /"cache_write_tokens" *: *([0-9]+)/, m)) cw += m[1]+0
}
END {
  printf "{\"input_tokens\":%d,\"output_tokens\":%d,\"cache_read_tokens\":%d,\"cache_write_tokens\":%d}\n", \
    ti, to, cr, cw
}
' "$TOKEN_USAGE_FILE" > /tmp/gh-aw/agent_usage.json

echo "Token usage summary appended to step summary"
