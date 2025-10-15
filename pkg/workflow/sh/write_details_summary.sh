#!/bin/bash
# Helper function to write content with HTML details/summary tags to step summary
# This script defines a function that can be sourced and used by other scripts

# Function: write_details_to_summary
# Writes content wrapped in HTML details/summary tags to GITHUB_STEP_SUMMARY
# Args:
#   $1 - Summary title (the clickable summary text)
#   $2 - Content file path (file to read content from)
#   $3 - Language for code block (optional, defaults to 'text')
write_details_to_summary() {
  local title="$1"
  local content_file="$2"
  local lang="${3:-text}"
  
  echo "<details>" >> "$GITHUB_STEP_SUMMARY"
  echo "<summary>$title</summary>" >> "$GITHUB_STEP_SUMMARY"
  echo "" >> "$GITHUB_STEP_SUMMARY"
  echo '```'"$lang" >> "$GITHUB_STEP_SUMMARY"
  
  if [ -f "$content_file" ]; then
    cat "$content_file" >> "$GITHUB_STEP_SUMMARY"
  fi
  
  echo '```' >> "$GITHUB_STEP_SUMMARY"
  echo "</details>" >> "$GITHUB_STEP_SUMMARY"
}

