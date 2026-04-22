#!/usr/bin/env bash
set +o histexpand

#
# compute_artifact_prefix.sh - Compute a stable artifact name prefix for workflow_call runs
#
# When the same reusable workflow is called by multiple jobs within a single parent
# workflow run, all invocations would upload artifacts with identical names
# (e.g. "activation", "agent") causing 409 Conflict errors.
#
# This script derives a unique prefix from the workflow inputs and run attempt so
# that each distinct invocation produces separate artifact names (e.g.
# "a1b2c3d4-activation", "e5f6a7b8-activation").
#
# Environment variables:
#   INPUTS_JSON          (required) JSON-serialised workflow inputs, typically
#                        set via ${{ toJSON(inputs) }} in the workflow step env.
#                        Passed through an env-var to prevent template injection.
#   GITHUB_RUN_ATTEMPT   (auto-provided by GitHub Actions) run attempt number,
#                        ensures prefixes differ across retries of the same run.
#
# GitHub Actions output:
#   prefix               8-hex-char SHA256 digest followed by "-"
#                        (e.g. "a1b2c3d4-"), or empty string on error.
#
# Uniqueness guarantee:
#   - Two calls with different inputs → different prefixes (collision-resistant SHA256).
#   - Two calls with the same inputs on different run attempts → different prefixes
#     because GITHUB_RUN_ATTEMPT is included in the digest.
#   - Two calls with identical inputs on the same run attempt → same prefix (conflict).
#     Callers MUST provide different inputs to avoid this edge case.
#
# Security:
#   Inputs are consumed via an environment variable, never interpolated directly
#   into the shell, preventing script-injection attacks.

set -euo pipefail

# INPUTS_JSON is set to ${{ toJSON(inputs) }} in the step env.
# When a workflow_call has no inputs, toJSON(inputs) returns "{}", so "{}" is
# the correct default for the no-inputs case (not an error).
INPUTS="${INPUTS_JSON:-{}}"
# GITHUB_RUN_ATTEMPT is provided automatically by GitHub Actions. Default to 1
# for local/test environments where it may not be set.
ATTEMPT="${GITHUB_RUN_ATTEMPT:-1}"

echo "Computing artifact prefix from workflow inputs and run attempt..."
echo "  GITHUB_RUN_ATTEMPT: ${ATTEMPT}"
echo "  Inputs JSON length: ${#INPUTS} chars"

# Combine inputs JSON with run attempt using a separator unlikely to appear in
# valid JSON ("::attempt=" contains characters that would be escaped in JSON
# string values), so different inputs cannot produce the same pre-hash string
# as each other or as a different run attempt.
HASH_INPUT="${INPUTS}::attempt=${ATTEMPT}"
PREFIX=$(printf '%s' "$HASH_INPUT" | sha256sum | cut -c1-8)

echo "  SHA256 digest (first 8 chars): ${PREFIX}"
echo "  Artifact prefix: ${PREFIX}-"
echo "prefix=${PREFIX}-" >> "$GITHUB_OUTPUT"
