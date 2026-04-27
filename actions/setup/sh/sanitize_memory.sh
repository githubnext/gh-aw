#!/usr/bin/env bash

# sanitize_memory.sh
# Pre-agent content scanning for prompt injection in memory files.
#
# This script scans text files in a memory directory for known prompt injection
# patterns (system prompt overrides, role-play injections, instruction-ignoring
# directives) per OWASP Agentic Top 10 — ASI-06 (Memory & Context Poisoning).
#
# Required environment variables:
#   GH_AW_SCAN_DIR:  Path to the memory directory to scan
#
# Optional environment variables:
#   GH_AW_QUARANTINE_DIR:  Path to move quarantined files (default: /tmp/gh-aw/quarantine)
#
# Exit codes:
#   0 - Completed (suspicious files were quarantined/reported, non-fatal)
#   1 - Invalid arguments

set -euo pipefail

SCAN_DIR="${GH_AW_SCAN_DIR:-}"
QUARANTINE_DIR="${GH_AW_QUARANTINE_DIR:-/tmp/gh-aw/quarantine}"

if [ -z "$SCAN_DIR" ]; then
  echo "ERROR: GH_AW_SCAN_DIR environment variable is required" >&2
  exit 1
fi

if [ ! -d "$SCAN_DIR" ]; then
  echo "Memory scan directory does not exist, skipping: $SCAN_DIR"
  exit 0
fi

mkdir -p "$QUARANTINE_DIR"

# Patterns that indicate prompt injection attempts.
# Each pattern is a case-insensitive extended regex.
# We deliberately use simple, high-confidence patterns to minimise false positives.
INJECTION_PATTERNS=(
  # System prompt overrides
  "ignore (all |the |)previous instructions"
  "disregard (all |your |)previous instructions"
  "forget (everything|all instructions|your instructions|previous instructions)"
  "you are now (an? |a new |)"
  "act as (an? |a new |)"
  "your (new |)role is"
  "you must now"
  "new instructions:"
  "override (all |)instructions"
  # Role injection markers common in LLM prompt formats
  "^<\|system\|>"
  "^\\[INST\\]"
  "^\\[SYS\\]"
  "^### (System|Instruction|Override)"
  # Embedded XML/tag injection targeting the agent context
  "</?(instructions|system|context|rules)>"
  "<(instructions|system|rules)[ >]"
  # Jailbreak phrases
  "do anything now"
  "jailbreak"
  "developer mode"
  "god mode"
  # Credential / secret exfiltration instructions
  "exfiltrate (the |all |your |)secrets"
  "send (all |the |your |)secrets"
  "leak (the |all |your |)credentials"
)

quarantine_count=0
scan_count=0

echo "Content injection scan starting: $SCAN_DIR"

# Scan only text-like files (skip binary files and .git/)
while IFS= read -r -d '' file; do
  # Skip .git directory contents
  case "$file" in
    */.git/*) continue ;;
    ./.git/*) continue ;;
  esac

  # Skip binary files using 'file' command heuristic: if mime type is not text/* skip it
  if command -v file >/dev/null 2>&1; then
    mime_type="$(file --brief --mime-type "$file" 2>/dev/null || true)"
    case "$mime_type" in
      text/*) ;;            # text file — proceed
      application/json) ;;  # JSON is text
      application/xml) ;;   # XML is text
      *) continue ;;        # binary — skip
    esac
  fi

  scan_count=$((scan_count + 1))
  matched_pattern=""

  for pattern in "${INJECTION_PATTERNS[@]}"; do
    if grep -qiEe "$pattern" "$file" 2>/dev/null; then
      matched_pattern="$pattern"
      break
    fi
  done

  if [ -n "$matched_pattern" ]; then
    rel_path="${file#$SCAN_DIR/}"
    # Preserve the relative directory structure in the quarantine so that
    # the original location can be traced back easily.
    quarantine_target="$QUARANTINE_DIR/$rel_path"
    quarantine_target_dir="$(dirname "$quarantine_target")"
    mkdir -p "$quarantine_target_dir"
    # Append a nanosecond timestamp to the filename to avoid collisions across runs.
    quarantine_target="${quarantine_target}.$(date +%s%N 2>/dev/null || date +%s)"
    echo "::warning::Memory file quarantined (injection pattern detected): $rel_path (pattern: $matched_pattern)"
    echo "Quarantining suspicious file: $rel_path -> $quarantine_target"
    mv "$file" "$quarantine_target"
    quarantine_count=$((quarantine_count + 1))
  fi
done < <(find "$SCAN_DIR" -not -path '*/.git/*' -type f -print0 2>/dev/null)

echo "Content injection scan complete: scanned=${scan_count} quarantined=${quarantine_count} dir=${SCAN_DIR}"
