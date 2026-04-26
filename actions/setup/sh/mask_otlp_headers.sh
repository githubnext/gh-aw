#!/usr/bin/env bash
set +o histexpand

#
# mask_otlp_headers.sh - Mask OTEL_EXPORTER_OTLP_HEADERS from GitHub Actions logs
#
# Issues the ::add-mask:: workflow command for OTEL_EXPORTER_OTLP_HEADERS so that
# authentication tokens in the header value do not leak into GitHub Actions runner
# logs (including debug/step-debug logs).
#
# Three levels of masking are applied:
#   1. The entire OTEL_EXPORTER_OTLP_HEADERS value (comma-separated header pairs).
#   2. Each individual header value extracted from the pairs, so that a token
#      appearing without its header name prefix is also redacted.
#   3. For Authorization-style "Bearer <token>" credentials, the raw token after
#      stripping the "Bearer " scheme prefix, so it is masked even when it appears
#      without the scheme (e.g. in downstream tool logs).
#
# Mixed quoting ('::add-mask::' followed by "$VAR") is used so the directive prefix
# is treated as a literal string while the variable values are expanded at runtime.
#
# Exit codes:
#   0 - Success (OTEL_EXPORTER_OTLP_HEADERS may be empty, which is a no-op)

set -euo pipefail

# Level 1: mask the entire comma-separated headers string.
echo '::add-mask::'"$OTEL_EXPORTER_OTLP_HEADERS"

# Levels 2 & 3: split on commas, extract each value, and mask it individually.
# For "Bearer <token>" values, also mask the raw token without the scheme prefix.
printf '%s' "$OTEL_EXPORTER_OTLP_HEADERS" | tr ',' '\n' | while IFS= read -r _pair; do
  _val="${_pair#*=}"
  [ -n "$_val" ] && echo '::add-mask::'"$_val"
  _no_bearer="${_val#Bearer }"
  [ "$_no_bearer" != "$_val" ] && echo '::add-mask::'"$_no_bearer"
done
