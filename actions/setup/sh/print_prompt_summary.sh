# Print prompt to workflow logs (equivalent to core.info)
echo "Generated Prompt:"
cat "$GH_AW_PROMPT"

# Check for unresolved placeholders (@@VAR@@)
if grep -q '@@GH_AW_[A-Z0-9_]*@@' "$GH_AW_PROMPT"; then
  echo "::warning::Unresolved placeholders found in prompt:"
  grep -o '@@GH_AW_[A-Z0-9_]*@@' "$GH_AW_PROMPT" | sort | uniq | while read -r placeholder; do
    echo "::warning::  - $placeholder"
  done
fi

# Print prompt to step summary
{
  echo "<details>"
  echo "<summary>Generated Prompt</summary>"
  echo ""
  echo '``````markdown'
  cat "$GH_AW_PROMPT"
  echo '``````'
  echo ""
  echo "</details>"
} >> "$GITHUB_STEP_SUMMARY"
