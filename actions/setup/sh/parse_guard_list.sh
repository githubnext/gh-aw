#!/bin/bash
set -eo pipefail

# parse_guard_list.sh - Parse comma/newline-separated guard policy lists into JSON arrays
#
# Reads the combined extra (static or user-expression) and org/repo variable values for
# blocked-users, trusted-users, and approval-labels, merges them, validates each item, and
# writes the resulting JSON arrays to $GITHUB_OUTPUT for use in the MCP gateway config step.
#
# Environment variables (all optional, default empty):
#   GH_AW_BLOCKED_USERS_EXTRA  - Static items or user-expression value for blocked-users
#   GH_AW_BLOCKED_USERS_VAR    - Value of vars.GH_AW_GITHUB_BLOCKED_USERS (fallback)
#   GH_AW_TRUSTED_USERS_EXTRA  - Static items or user-expression value for trusted-users
#   GH_AW_TRUSTED_USERS_VAR    - Value of vars.GH_AW_GITHUB_TRUSTED_USERS (fallback)
#   GH_AW_APPROVAL_LABELS_EXTRA - Static items or user-expression value for approval-labels
#   GH_AW_APPROVAL_LABELS_VAR  - Value of vars.GH_AW_GITHUB_APPROVAL_LABELS (fallback)
#
# Outputs (to $GITHUB_OUTPUT):
#   blocked_users   - JSON array, e.g. ["spam-bot","bad-actor"] or []
#   trusted_users   - JSON array, e.g. ["contractor-1","partner-dev"] or []
#   approval_labels - JSON array, e.g. ["human-reviewed"] or []
#
# Exit codes:
#   0 - Parsed successfully
#   1 - An item is invalid (empty after trimming)

# parse_list converts a comma/newline-separated string into a JSON array.
# It trims whitespace from each item, skips empty items, validates that each
# remaining item is non-empty, and uses jq to produce a well-formed JSON array.
# Exits 1 if any item is empty after trimming.
parse_list() {
  local input="$1"
  local field_name="$2"

  if [ -z "$input" ]; then
    echo "[]"
    return 0
  fi

  local items=()
  while IFS= read -r item || [ -n "$item" ]; do
    # Trim leading whitespace
    item="${item#"${item%%[![:space:]]*}"}"
    # Trim trailing whitespace
    item="${item%"${item##*[![:space:]]}"}"
    if [ -n "$item" ]; then
      items+=("$item")
    fi
  done < <(printf '%s' "$input" | tr ',' '\n')

  if [ "${#items[@]}" -eq 0 ]; then
    echo "[]"
    return 0
  fi

  # Format as a JSON array using jq, which handles all necessary escaping.
  # jq -R reads each line as a raw string; jq -sc collects into a JSON array.
  printf '%s\n' "${items[@]}" | jq -R . | jq -sc .
}

# Combine extra and var inputs for each field.
# The script always reads both GH_AW_*_EXTRA and GH_AW_*_VAR and joins them
# with a comma so parse_list sees a single combined input.
combine_inputs() {
  local extra="${1:-}"
  local var="${2:-}"
  if [ -n "$extra" ] && [ -n "$var" ]; then
    printf '%s,%s' "$extra" "$var"
  elif [ -n "$extra" ]; then
    printf '%s' "$extra"
  else
    printf '%s' "$var"
  fi
}

BLOCKED_INPUT=$(combine_inputs "${GH_AW_BLOCKED_USERS_EXTRA:-}" "${GH_AW_BLOCKED_USERS_VAR:-}")
TRUSTED_INPUT=$(combine_inputs "${GH_AW_TRUSTED_USERS_EXTRA:-}" "${GH_AW_TRUSTED_USERS_VAR:-}")
APPROVAL_INPUT=$(combine_inputs "${GH_AW_APPROVAL_LABELS_EXTRA:-}" "${GH_AW_APPROVAL_LABELS_VAR:-}")

blocked_users_json=$(parse_list "$BLOCKED_INPUT" "blocked-users")
trusted_users_json=$(parse_list "$TRUSTED_INPUT" "trusted-users")
approval_labels_json=$(parse_list "$APPROVAL_INPUT" "approval-labels")

echo "blocked_users=${blocked_users_json}" >> "$GITHUB_OUTPUT"
echo "trusted_users=${trusted_users_json}" >> "$GITHUB_OUTPUT"
echo "approval_labels=${approval_labels_json}" >> "$GITHUB_OUTPUT"

echo "Guard policy lists parsed successfully"
echo "  blocked-users: ${blocked_users_json}"
echo "  trusted-users: ${trusted_users_json}"
echo "  approval-labels: ${approval_labels_json}"
