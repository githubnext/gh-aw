#!/usr/bin/env bash
set +o histexpand

#
# mask_otlp_attributes.sh - Mask GH_AW_OTLP_ATTRIBUTES values from GitHub Actions logs
#
# Issues the ::add-mask:: workflow command for every value found in the
# GH_AW_OTLP_ATTRIBUTES JSON object so that user-supplied custom OTLP span
# attribute values (e.g. session IDs, user IDs) do not leak into GitHub
# Actions runner logs (including debug/step-debug logs).
#
# GH_AW_OTLP_ATTRIBUTES is a JSON-encoded Record<string, string> injected by
# the gh-aw compiler from the `observability.otlp.attributes` frontmatter field.
# Each value is masked individually; empty values are skipped.
#
# Requires node to be available on PATH (it is always present on GitHub Actions
# runners when the gh-aw setup step has run).
#
# Exit codes:
#   0 - Success (variable may be absent or empty, which is a no-op)

set -euo pipefail

_attrs="${GH_AW_OTLP_ATTRIBUTES:-}"
[ -z "$_attrs" ] && exit 0

# Use node to extract the string values from the JSON object and print one
# per line (null/empty values are omitted).
_GH_AW_NODE=$(which node 2>/dev/null || command -v node 2>/dev/null || echo node)

# Read the values into an array, then issue ::add-mask:: for each non-empty one.
mapfile -t _values < <(
  printf '%s' "$_attrs" | "$_GH_AW_NODE" -e '
    let raw = "";
    process.stdin.on("data", d => { raw += d; });
    process.stdin.on("end", () => {
      try {
        const obj = JSON.parse(raw);
        if (obj !== null && typeof obj === "object" && !Array.isArray(obj)) {
          for (const v of Object.values(obj)) {
            if (typeof v === "string" && v.length > 0) {
              process.stdout.write(v + "\n");
            }
          }
        }
      } catch { /* invalid JSON – no-op */ }
    });
  '
)

for _val in "${_values[@]}"; do
  [ -n "$_val" ] && echo '::add-mask::'"$_val"
done
